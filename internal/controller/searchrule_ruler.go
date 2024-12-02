package controller

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/types"
	"prosimcorp.com/SearchRuler/api/v1alpha1"
)

var (
	queryConnectorCreds *Credentials
	actionCreds         *Credentials
	credsExists         bool
)

// DeleteAlertFromPool
func (r *SearchRuleReconciler) DeleteAlertFromPool(ctx context.Context, resource *v1alpha1.SearchRule) (err error) {
	alertKey := fmt.Sprintf("%s/%s", resource.Namespace, resource.Name)
	_, alertExists := SearchRuleAlertPool.Get(alertKey)
	if alertExists {
		SearchRuleAlertPool.Delete(alertKey)
	}
	return nil
}

// evaluateCondition
func evaluateCondition(value interface{}, operator string, threshold string) (bool, error) {
	//
	floatValue, ok := value.(float64)
	if !ok {
		return false, fmt.Errorf("conditionField does not return a valid number: %v", value)
	}

	//
	floatThreshold, err := strconv.ParseFloat(threshold, 64)
	if err != nil {
		return false, fmt.Errorf("configured threshold is not a valid float: %v", threshold)
	}

	// Evaluate
	switch operator {
	case "greaterThan":
		return floatValue > floatThreshold, nil
	case "lessThan":
		return floatValue < floatThreshold, nil
	case "equal":
		return floatValue == floatThreshold, nil
	default:
		return false, fmt.Errorf("unknown configured operator: %q", operator)
	}
}

// extractResponseField from elasticsearch response
func extractResponseField(data map[string]interface{}, fieldPath string) (interface{}, error) {
	// split with dots (e.g., "hits.total.value")
	fields := strings.Split(fieldPath, ".")

	//
	var current interface{} = data
	for _, field := range fields {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("error TODO %q", field)
		}
		current, ok = m[field]
		if !ok {
			return nil, fmt.Errorf("field %q not found", field)
		}
	}

	//
	return current, nil
}

// SyncTarget call Kubernetes API to actually perform actions over the resource
func (r *SearchRuleReconciler) CheckRule(ctx context.Context, resource *v1alpha1.SearchRule) (err error) {

	// Get SearchRulerQueryConnector associated to the rule
	searchRulerQueryConnectorResource := &v1alpha1.SearchRulerQueryConnector{}
	searchRulerQueryConnectorNamespacedName := types.NamespacedName{
		Namespace: resource.Namespace,
		Name:      resource.Spec.QueryConnectorRef.Name,
	}
	err = r.Get(ctx, searchRulerQueryConnectorNamespacedName, searchRulerQueryConnectorResource)
	if searchRulerQueryConnectorResource.Name == "" {
		return fmt.Errorf("SearchRulerQueryConnector %s not found in the resource namespace %s", resource.Spec.QueryConnectorRef.Name, resource.Namespace)
	}

	// Get credentials for SearchRulerQueryConnector attached
	if searchRulerQueryConnectorResource.Spec.Credentials.SecretRef.Name != "" {
		key := fmt.Sprintf("%s/%s", resource.Namespace, searchRulerQueryConnectorResource.Name)
		queryConnectorCreds, credsExists = SearchRulerQueryConnectorCredentialsPool.Get(key)
		if !credsExists {
			return fmt.Errorf("credentials not found for %s", key)
		}
	}

	// Get SearchRulerAction associated to the rule
	searchRulerActionResource := &v1alpha1.SearchRulerAction{}
	searchRulerActionNamespacedName := types.NamespacedName{
		Namespace: resource.Namespace,
		Name:      resource.Spec.ActionRef.Name,
	}
	err = r.Get(ctx, searchRulerActionNamespacedName, searchRulerActionResource)
	if searchRulerActionResource.Name == "" {
		return fmt.Errorf("SearchRulerAction %s not found in the resource namespace %s", resource.Spec.ActionRef.Name, resource.Namespace)
	}

	// Get credentials for SearchRulerAction attached
	if searchRulerActionResource.Spec.Webhook.Credentials.SecretRef.Name != "" {
		key := fmt.Sprintf("%s/%s", resource.Namespace, searchRulerActionResource.Name)
		actionCreds, credsExists = SearchRulerActionCredentialsPool.Get(key)
		if !credsExists {
			return fmt.Errorf("credentials not found for %s", key)
		}
	}

	// Check if query is defined
	if resource.Spec.Elasticsearch.Query == nil {
		return fmt.Errorf("query not defined")
	}

	// Get elasticsearch query to execute from resource
	elasticQuery, err := json.Marshal(resource.Spec.Elasticsearch.Query)
	if err != nil {
		return fmt.Errorf("error marshalling query body: %v", err)
	}

	// Make http client for elasticsearch connection
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: searchRulerQueryConnectorResource.Spec.TlsSkipVerify,
			},
		},
	}

	// Generate URL for search to elastic
	searchURL := fmt.Sprintf("%s/%s/_search", searchRulerQueryConnectorResource.Spec.URL, resource.Spec.Elasticsearch.Index)
	req, err := http.NewRequest("POST", searchURL, bytes.NewBuffer(elasticQuery))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	// Add headers and custom headers for elasticsearch queries
	req.Header.Set("Content-Type", "application/json")
	for key, value := range searchRulerQueryConnectorResource.Spec.Headers {
		req.Header.Set(key, value)
	}

	// Add authentication if set for elasticsearch queries
	if searchRulerQueryConnectorResource.Spec.Credentials.SecretRef.Name != "" {
		req.SetBasicAuth(queryConnectorCreds.Username, queryConnectorCreds.Password)
	}

	// Make request to elasticsearch
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error executing request %s: %v", string(elasticQuery), err)
	}
	defer resp.Body.Close()

	// Read response and check if it is ok
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error response from Elasticsearch executing request %s: %s", string(elasticQuery), string(responseBody))
	}

	// Unmarshal elasticsaerch response
	var response map[string]interface{}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return fmt.Errorf("error unmarshalling response: %v", err)
	}

	// Extract response field for elasticsearch and the conditionField defined in the manifest
	conditionValue, err := extractResponseField(response, resource.Spec.Elasticsearch.ConditionField)
	if err != nil {
		return fmt.Errorf("error getting field from response: %v\n", err)
	}

	// Evaluate condition and check if the alert is firing or not
	firing, err := evaluateCondition(conditionValue, resource.Spec.Condition.Operator, resource.Spec.Condition.Threshold)
	if err != nil {
		return fmt.Errorf("error evaluating condition: %v\n", err)
	}

	// Get alertKey for the pool <namespace>/<name> and get it from the pool if exists
	// If not, create a default skeleton alert and save it to the pool
	alertKey := fmt.Sprintf("%s/%s", resource.Namespace, resource.Name)
	alert, alertInPool := SearchRuleAlertPool.Get(alertKey)
	if !alertInPool {
		alert = &Alert{
			fireTime:      time.Time{},
			firing:        false,
			notified:      false,
			resolving:     false,
			resolvingTime: time.Time{},
		}
		SearchRuleAlertPool.Set(alertKey, alert)
	}

	// Get for duration for the alerts firing. When alert is firing during this for time,
	// then the alert is really ocurring
	forDuration, err := time.ParseDuration(resource.Spec.Condition.For)
	if err != nil {
		return fmt.Errorf("error parsing `for` time: %v", err)
	}

	// If alert is firing right now
	if firing {

		// If alert is not set as firing in the pool, set start fireTime and firing as true
		if !alert.firing {
			alert.fireTime = time.Now()
			alert.firing = true
			SearchRuleAlertPool.Set(alertKey, alert)
		}

		// If alert is marked as resolving previously, now it is firing again, so reset fireTime to now and false resolving
		if alert.resolving {
			alert.fireTime = time.Now()
			alert.resolving = false
			SearchRuleAlertPool.Set(alertKey, alert)
		}

		// If alert is firing the For time and it is not notified yet, do it
		if time.Since(alert.fireTime) > forDuration && !alert.notified {
			alert.notified = true
			SearchRuleAlertPool.Set(alertKey, alert)

			log.Println("Firing")
		}

		return nil
	}

	// If alert is not firing right now and it is marked as firing in the pool
	if !firing && alert.firing {

		// If Alert is not marked as resolving in the pool, do it and set start resolvingTime now
		if !alert.resolving {
			alert.resolving = true
			alert.resolvingTime = time.Now()
			SearchRuleAlertPool.Set(alertKey, alert)
		}

		// If Alert stay in resoliving state during the for time or it is not notified, mark as resolved
		if time.Since(alert.resolvingTime) > forDuration || !alert.notified {

			// If the alert was notified as firing, then send resolved message
			if alert.notified {
				log.Println("Resolved")
			}

			// Clean up alert to default values
			alert = &Alert{
				fireTime:      time.Time{},
				firing:        false,
				notified:      false,
				resolving:     false,
				resolvingTime: time.Time{},
			}
			SearchRuleAlertPool.Set(alertKey, alert)
		}
	}

	return nil
}
