/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package searchrule

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"

	corev1 "k8s.io/api/core/v1"
	eventsv1 "k8s.io/api/events/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/tidwall/gjson"

	//
	"prosimcorp.com/SearchRuler/api/v1alpha1"
	"prosimcorp.com/SearchRuler/internal/controller"
	"prosimcorp.com/SearchRuler/internal/globals"
	"prosimcorp.com/SearchRuler/internal/pools"
)

const (

	// Rule states
	RuleNormalState          = "Normal"
	RuleFiringState          = "Firing"
	RulePendingFiringState   = "PendingFiring"
	RulePendingResolvedState = "PendingResolving"

	// Conditions
	conditionGreaterThan        = "greaterThan"
	conditionGreaterThanOrEqual = "greaterThanOrEqual"
	conditionLessThan           = "lessThan"
	conditionLessThanOrEqual    = "lessThanOrEqual"
	conditionEqual              = "equal"

	// kubeEvent
	kubeEventReasonAlertFiring = "AlertFiring"

	// Elasticsearch aggregation field
	elasticAggregationsField = "aggregations"
)

var (
	queryConnectorCreds *pools.Credentials
	credsExists         bool

	// Elasticsearch search path
	ElasticsearchSearchURL = "%s/%s/_search"
)

// Sync execute the query to the elasticsearch and evaluate the condition. Then trigger the action adding the alert to the pool
// and sending an event to the Kubernetes API
func (r *SearchRuleReconciler) Sync(ctx context.Context, eventType watch.EventType, resource *v1alpha1.SearchRule) (err error) {

	logger := log.FromContext(ctx)

	// If the eventType is Deleted, remove the rule from the rules pool and from the alerts pool
	// In other cases, execute Sync logic
	if eventType == watch.Deleted {
		key := fmt.Sprintf("%s_%s", resource.Namespace, resource.Name)
		r.RulesPool.Delete(key)
		r.AlertsPool.Delete(key)
		return nil
	}

	// Get QueryConnector associated to the rule with KubeRawClient
	gvr := schema.GroupVersionResource{
		Group:    v1alpha1.GroupVersion.Group,
		Version:  v1alpha1.GroupVersion.Version,
		Resource: "clusterqueryconnectors",
	}

	queryConnectorWrapper := globals.Application.KubeRawClient.Resource(gvr)
	if resource.Spec.QueryConnectorRef.Namespace != "" {
		gvr.Resource = "queryconnectors"
		queryConnectorWrapper = globals.Application.KubeRawClient.Resource(gvr)
		queryConnectorWrapper.Namespace(resource.Spec.QueryConnectorRef.Namespace)
	}

	QueryConnectorResource, err := queryConnectorWrapper.Get(ctx, resource.Spec.QueryConnectorRef.Name, metav1.GetOptions{})
	if err != nil {
		// TODO: Improve this
		return err
	}

	// If QueryConnector is empty then error
	if reflect.ValueOf(QueryConnectorResource).IsZero() {
		r.UpdateConditionQueryConnectorNotFound(resource)
		return fmt.Errorf(
			controller.QueryConnectorNotFoundMessage,
			resource.Spec.QueryConnectorRef.Name,
			resource.Namespace,
		)
	}

	// Tricky for save queryConnector resource with QueryConnectorSpec type
	QueryConnectorSpec := &v1alpha1.QueryConnectorSpec{}
	QueryConnectorSpecI := QueryConnectorResource.Object["spec"]
	specBytes, err := json.Marshal(QueryConnectorSpecI)
	if err != nil {
		return fmt.Errorf(controller.JSONMarshalErrorMessage, err)
	}
	err = json.Unmarshal(specBytes, QueryConnectorSpec)
	if err != nil {
		return fmt.Errorf(controller.JSONMarshalErrorMessage, err)
	}

	// Get credentials for QueryConnector attached if defined
	if !reflect.ValueOf(QueryConnectorSpec.Credentials).IsZero() {
		key := fmt.Sprintf("%s_%s", QueryConnectorResource.GetNamespace(), QueryConnectorResource.GetName())
		queryConnectorCreds, credsExists = r.QueryConnectorCredentialsPool.Get(key)
		if !credsExists {
			r.UpdateConditionNoCredsFound(resource)
			return fmt.Errorf(controller.MissingCredentialsMessage, key)
		}
	}

	// Get `for` duration for the rules firing. When rule is firing during this for time,
	// then the rule is really ocurring and must be an alert
	forDuration, err := time.ParseDuration(resource.Spec.Condition.For)
	if err != nil {
		return fmt.Errorf(controller.ForValueParseErrorMessage, err)
	}

	// Check if query is defined in the resource
	if resource.Spec.Elasticsearch.Query == nil && resource.Spec.Elasticsearch.QueryJSON == "" {
		r.UpdateConditionNoQueryFound(resource)
		return fmt.Errorf(controller.QueryNotDefinedErrorMessage, resource.Name)
	}

	// Check if both query and queryJson are defined. If true, return error
	if resource.Spec.Elasticsearch.Query != nil && resource.Spec.Elasticsearch.QueryJSON != "" {
		r.UpdateConditionNoQueryFound(resource)
		return fmt.Errorf(controller.QueryDefinedInBothErrorMessage, resource.Name)
	}

	// Select query to use and marshall to JSON
	var elasticQuery []byte
	// If query is defined in the resource, just Marshal it
	if resource.Spec.Elasticsearch.Query != nil {
		elasticQuery, err = json.Marshal(resource.Spec.Elasticsearch.Query)
		if err != nil {
			return fmt.Errorf(controller.JSONMarshalErrorMessage, err)
		}
	}
	// If queryJSON is defined in the resource, it is already a JSON, just convert it to bytes
	if resource.Spec.Elasticsearch.QueryJSON != "" {
		elasticQuery = []byte(resource.Spec.Elasticsearch.QueryJSON)
	}

	// Make http client for elasticsearch connection
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: QueryConnectorSpec.TlsSkipVerify,
			},
		},
	}

	// Generate URL for search to elasticsearch
	searchURL := fmt.Sprintf(
		ElasticsearchSearchURL,
		QueryConnectorSpec.URL,
		resource.Spec.Elasticsearch.Index,
	)
	req, err := http.NewRequest("POST", searchURL, bytes.NewBuffer(elasticQuery))
	if err != nil {
		r.UpdateConditionConnectionError(resource)
		return fmt.Errorf(controller.HttpRequestCreationErrorMessage, err)
	}

	// Add headers and custom headers for elasticsearch queries
	req.Header.Set("Content-Type", "application/json")
	for key, value := range QueryConnectorSpec.Headers {
		req.Header.Set(key, value)
	}

	// Add authentication if set for elasticsearch queries
	if QueryConnectorSpec.Credentials.SecretRef.Name != "" {
		req.SetBasicAuth(queryConnectorCreds.Username, queryConnectorCreds.Password)
	}

	// Make request to elasticsearch
	resp, err := httpClient.Do(req)
	if err != nil {
		r.UpdateConditionConnectionError(resource)
		return fmt.Errorf(controller.ElasticsearchQueryErrorMessage, string(elasticQuery), err)
	}
	defer resp.Body.Close()

	// Read response and check if it is ok
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		r.UpdateConditionQueryError(resource)
		return fmt.Errorf(controller.ResponseBodyReadErrorMessage, err)
	}
	if resp.StatusCode != http.StatusOK {
		r.UpdateConditionQueryError(resource)
		return fmt.Errorf(
			controller.ElasticsearchQueryResponseErrorMessage,
			string(elasticQuery),
			string(responseBody),
		)
	}

	// Extract conditionField from the response field of elasticsearch
	conditionValue := gjson.Get(string(responseBody), resource.Spec.Elasticsearch.ConditionField)
	if !conditionValue.Exists() {
		r.UpdateConditionQueryError(resource)
		return fmt.Errorf(
			controller.ConditionFieldNotFoundMessage,
			resource.Spec.Elasticsearch.ConditionField,
			string(responseBody),
		)
	}

	// Save elastic response if the result has aggregations, this allows user
	// to use the response in the action
	aggregationsResource := interface{}(nil)
	aggregationsResponse := gjson.Get(string(responseBody), elasticAggregationsField)
	if aggregationsResponse.Exists() {
		aggregationsResource = aggregationsResponse.Value()
	}

	// Evaluate condition and check if the alert is firing or not
	firing, err := evaluateCondition(conditionValue.Float(), resource.Spec.Condition.Operator, resource.Spec.Condition.Threshold)
	if err != nil {
		r.UpdateConditionQueryError(resource)
		return fmt.Errorf(
			controller.EvaluatingConditionErrorMessage,
			err,
		)
	}

	// Get ruleKey for the pool <namespace>_<name> and get rule from the pool if exists
	// If not, create a default skeleton rule and save it to the pool
	ruleKey := fmt.Sprintf("%s_%s", resource.Namespace, resource.Name)
	rule, ruleInPool := r.RulesPool.Get(ruleKey)
	if !ruleInPool {
		// Initialize rule with default values
		rule = &pools.Rule{
			SearchRule:    *resource,
			FiringTime:    time.Time{},
			State:         RuleNormalState,
			ResolvingTime: time.Time{},
			Value:         conditionValue.Float(),
			Aggregations:  nil,
		}
		r.RulesPool.Set(ruleKey, rule)
	}

	// Check if resource is sync with the pool
	if !reflect.DeepEqual(rule.SearchRule, *resource) {
		rule.SearchRule = *resource
		r.RulesPool.Set(ruleKey, rule)
	}

	// Set the current value of the condition to the rule
	rule.Value = conditionValue.Float()
	rule.Aggregations = aggregationsResource
	r.RulesPool.Set(ruleKey, rule)

	// If rule is firing right now
	if firing {

		// If rule is not set as firing in the pool, set start fireTime and state PendingFiring
		if rule.State == RuleNormalState || rule.State == RulePendingResolvedState {
			rule.FiringTime = time.Now()
			rule.State = RulePendingFiringState
			r.RulesPool.Set(ruleKey, rule)
		}

		// If rule is firing the For time and it is not notified yet, do it and change state to Firing
		if time.Since(rule.FiringTime) > forDuration {
			rule.State = RuleFiringState
			r.RulesPool.Set(ruleKey, rule)

			// Add alert to the pool with the value, the object and the rulerAction name which will trigger the alert
			alertKey := fmt.Sprintf("%s_%s", resource.Namespace, resource.Name)
			r.AlertsPool.Set(alertKey, &pools.Alert{
				RulerActionName: resource.Spec.ActionRef.Name,
				SearchRule:      *resource,
				Value:           conditionValue.Float(),
				Aggregations:    aggregationsResource,
			})

			// Create an event in Kubernetes of AlertFiring. This event will be readed by the RulerAction controller
			// and will trigger the action inmediately
			err = createKubeEvent(
				ctx,
				*resource,
				kubeEventReasonAlertFiring,
				fmt.Sprintf("Rule is in firing state. Current value is %v", conditionValue),
			)
			if err != nil {
				return fmt.Errorf(controller.KubeEventCreationErrorMessage, err)
			}

			// Log the alert and change the AlertStatus to Firing of the searchRule
			r.UpdateConditionAlertFiring(resource)
			logger.Info(fmt.Sprintf(
				"Rule %s is in firing state. Current value is %v",
				resource.Name,
				conditionValue,
			))
			return nil

		}

		r.UpdateStateAlertPendingFiring(resource)
		return nil

	}

	// If alert is not firing right now and it is not in healthy state
	if !firing && rule.State != RuleNormalState {

		// If rule is not marked as resolving in the pool, change state to PendingResolved and set resolvingTime now
		if rule.State != RulePendingResolvedState {
			rule.State = RulePendingResolvedState
			rule.ResolvingTime = time.Now()
			r.RulesPool.Set(ruleKey, rule)
		}

		// If rule stay in PendingResolved state during the `for` time, mark as resolved
		if time.Since(rule.ResolvingTime) > forDuration {

			// Remove alert from the pool
			alertKey := fmt.Sprintf("%s_%s", resource.Namespace, resource.Name)
			r.AlertsPool.Delete(alertKey)

			// Restore rule to default values
			rule = &pools.Rule{
				FiringTime:    time.Time{},
				State:         RuleNormalState,
				ResolvingTime: time.Time{},
				SearchRule:    *resource,
				Value:         conditionValue.Float(),
				Aggregations:  aggregationsResource,
			}
			r.RulesPool.Set(ruleKey, rule)

			// Log and update the AlertStatus to Resolved
			r.UpdateStateNormal(resource)
			logger.Info(fmt.Sprintf(
				"Rule %s is in normal state. Current value is %v",
				resource.Name,
				conditionValue,
			))
			return nil
		}

		r.UpdateStateAlertPendingResolved(resource)
		return nil
	}

	r.UpdateStateNormal(resource)
	return nil
}

// evaluateCondition evaluates the conditionField with the operator and threshold
func evaluateCondition(value float64, operator string, threshold string) (bool, error) {

	// Parse threshold to float
	floatThreshold, err := strconv.ParseFloat(threshold, 64)
	if err != nil {
		return false, fmt.Errorf("configured threshold is not a valid float: %v", threshold)
	}

	// Evaluate condition
	switch operator {
	case conditionGreaterThan:
		return value > floatThreshold, nil
	case conditionGreaterThanOrEqual:
		return value >= floatThreshold, nil
	case conditionLessThan:
		return value < floatThreshold, nil
	case conditionLessThanOrEqual:
		return value <= floatThreshold, nil
	case conditionEqual:
		return value == floatThreshold, nil
	default:
		return false, fmt.Errorf("unknown configured operator: %q", operator)
	}
}

// createKubeEvent creates a modern event in Kubernetes with data given by params
func createKubeEvent(ctx context.Context, rule v1alpha1.SearchRule, action, message string) (err error) {

	// Define the event object
	eventObj := eventsv1.Event{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "searchruler-alert-",
		},

		EventTime:           metav1.NewMicroTime(time.Now()),
		ReportingController: "searchruler",
		ReportingInstance:   "searchruler-controller",
		Action:              action,
		Reason:              "AlertFiring",

		Regarding: corev1.ObjectReference{
			APIVersion: rule.APIVersion,
			Kind:       rule.Kind,
			Name:       rule.Name,
			Namespace:  rule.Namespace,
		},

		Note: message,
		Type: "Normal",
	}

	// Create the event in Kubernetes using the global client initiated in main.go
	_, err = globals.Application.KubeRawCoreClient.EventsV1().Events(rule.Namespace).
		Create(ctx, &eventObj, metav1.CreateOptions{})

	return err
}
