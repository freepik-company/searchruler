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

package ruleraction

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"net/http"
	"prosimcorp.com/SearchRuler/internal/globals"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"

	//
	"prosimcorp.com/SearchRuler/api/v1alpha1"
	"prosimcorp.com/SearchRuler/internal/controller"
	"prosimcorp.com/SearchRuler/internal/pools"
	"prosimcorp.com/SearchRuler/internal/template"
	"prosimcorp.com/SearchRuler/internal/validators"
)

var (
	// validatorsMap is a map of integration names and their respective validation functions
	validatorsMap = map[string]func(data string) (result bool, hint string, err error){
		"alertmanager": validators.ValidateAlertmanager,
	}
	resourceNamespace string
	resourceName      string
	resourceSpec      v1alpha1.RulerActionSpec
)

// Sync function is used to synchronize the RulerAction resource with the alerts. Executes the webhook defined in the
// resource for each alert found in the AlertsPool.
func (r *RulerActionReconciler) Sync(ctx context.Context, resource *CompoundRulerActionResource, resourceType string) (err error) {

	logger := log.FromContext(ctx)
	// Get the resource values depending on the resourceType
	switch resourceType {
	case controller.ClusterRulerActionResourceType:
		resourceNamespace = ""
		resourceName = resource.ClusterRulerActionResource.Name
		resourceSpec = resource.ClusterRulerActionResource.Spec
	case controller.RulerActionResourceType:
		resourceNamespace = resource.RulerActionResource.Namespace
		resourceName = resource.RulerActionResource.Name
		resourceSpec = resource.RulerActionResource.Spec
	}

	// Get credentials for the Action in the secret associated if defined
	username := ""
	password := ""
	if !reflect.ValueOf(resourceSpec.Webhook.Credentials).IsZero() {
		// First get secret with the credentials
		RulerActionCredsSecret := &corev1.Secret{}
		secretNamespace := resourceSpec.Webhook.Credentials.SecretRef.Namespace
		if secretNamespace == "" {
			secretNamespace = resourceNamespace
		}
		namespacedName := types.NamespacedName{
			Namespace: secretNamespace,
			Name:      resourceSpec.Webhook.Credentials.SecretRef.Name,
		}
		err = r.Get(ctx, namespacedName, RulerActionCredsSecret)
		if err != nil {
			r.UpdateConditionNoCredsFound(resource, resourceType)
			return fmt.Errorf(controller.SecretNotFoundErrorMessage, namespacedName, err)
		}

		// Get username and password
		username = string(RulerActionCredsSecret.Data[resourceSpec.Webhook.Credentials.SecretRef.KeyUsername])
		password = string(RulerActionCredsSecret.Data[resourceSpec.Webhook.Credentials.SecretRef.KeyPassword])
		if username == "" || password == "" {
			r.UpdateConditionNoCredsFound(resource, resourceType)
			return fmt.Errorf(controller.MissingCredentialsMessage, namespacedName)
		}
	}

	// Check alert pool for alerts related to this rulerAction
	// Alerts key pattern: namespace/rulerActionName/searchRuleName
	alerts, err := r.getRulerActionAssociatedAlerts(resourceName)
	if err != nil {
		return fmt.Errorf(controller.AlertsPoolErrorMessage, err)
	}

	// If there are alerts for the rulerAction, initialize the HTTP client
	if len(alerts) > 0 {
		// Create the HTTP client
		httpClient := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: resourceSpec.Webhook.TlsSkipVerify,
				},
			},
		}

		// Create the request with the configured verb and URL
		httpRequest, err := http.NewRequest(resourceSpec.Webhook.Verb, resourceSpec.Webhook.Url, nil)
		if err != nil {
			return fmt.Errorf(controller.HttpRequestCreationErrorMessage, err)
		}

		// Add headers to the request if set
		httpRequest.Header.Set("Content-Type", "application/json")
		for headerKey, headerValue := range resourceSpec.Webhook.Headers {
			httpRequest.Header.Set(headerKey, headerValue)
		}

		// Add authentication if set for the webhook
		if username == "" || password == "" {
			httpRequest.SetBasicAuth(username, password)
		}

		// For every alert found in the pool, execute the
		// webhook configured in the RulerAction resource
		for _, alert := range alerts {

			// Log alert firing
			logger.Info(fmt.Sprintf(
				controller.AlertFiringInfoMessage,
				alert.SearchRule.Namespace,
				alert.SearchRule.Name,
				alert.SearchRule.Spec.Description,
			))

			// Add parsed data to the request
			// object is the SearchRule object and value is the value of the alert
			// to be accessible in the template
			templateInjectedObject := map[string]interface{}{}
			templateInjectedObject["value"] = alert.Value
			templateInjectedObject["object"] = alert.SearchRule
			templateInjectedObject["aggregations"] = alert.Aggregations

			// Evaluate the data template with the injected object
			parsedMessage, err := template.EvaluateTemplate(alert.SearchRule.Spec.ActionRef.Data, templateInjectedObject)
			if err != nil {
				r.UpdateConditionEvaluateTemplateError(resource, resourceType)
				return fmt.Errorf(controller.EvaluateTemplateErrorMessage, err)
			}

			// Check if the webhook has a validator and execute it when available
			if resourceSpec.Webhook.Validator != "" {

				// Check if the validator is available
				_, validatorFound := validatorsMap[resourceSpec.Webhook.Validator]
				if !validatorFound {
					r.UpdateConditionEvaluateTemplateError(resource, resourceType)
					return fmt.Errorf(controller.ValidatorNotFoundErrorMessage, resourceSpec.Webhook.Validator)
				}

				// Execute the validator to the data of the alert
				validatorResult, validatorHint, err := validatorsMap[resourceSpec.Webhook.Validator](parsedMessage)
				if err != nil {
					r.UpdateConditionEvaluateTemplateError(resource, resourceType)
					return fmt.Errorf(controller.ValidationFailedErrorMessage, err.Error())
				}

				// Check the result of the validator
				if !validatorResult {
					r.UpdateConditionEvaluateTemplateError(resource, resourceType)
					return fmt.Errorf(controller.ValidationFailedErrorMessage, validatorHint)
				}
			}

			// Add data to the payload of the request
			payload := []byte(parsedMessage)
			httpRequest.Body = io.NopCloser(bytes.NewBuffer(payload))

			// Send HTTP request to the webhook
			httpResponse, err := httpClient.Do(httpRequest)
			if err != nil {
				r.UpdateConditionConnectionError(resource, resourceType)
				return fmt.Errorf(controller.HttpRequestSendingErrorMessage, err)
			}

			defer httpResponse.Body.Close()
		}
	}

	// Updates status to Success
	r.UpdateStateSuccess(resource, resourceType)
	return nil
}

// GetRuleActionFromEvent returns the RulerAction resource associated with the event that triggered the reconcile
func (r *RulerActionReconciler) GetEventRuleAction(ctx context.Context, ruleAction *CompoundRulerActionResource, namespace, name string) (resourceType string, err error) {

	// Get event resource from the namespace and name of the event that triggered the reconcile
	EventResource := &corev1.Event{}
	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	err = r.Get(ctx, namespacedName, EventResource)
	if err != nil {
		return resourceType, fmt.Errorf(
			"reconcile not triggered by event, triggered by resource %s : %v",
			namespacedName,
			err.Error(),
		)
	}

	// Get SearchRule resource from event resource
	searchRule := &v1alpha1.SearchRule{}
	searchRuleNamespacedName := types.NamespacedName{
		Namespace: EventResource.InvolvedObject.Namespace,
		Name:      EventResource.InvolvedObject.Name,
	}
	err = r.Get(ctx, searchRuleNamespacedName, searchRule)
	if err != nil {
		return resourceType, fmt.Errorf(
			"error fetching SearchRule %s from event %s: %v",
			searchRuleNamespacedName,
			namespacedName,
			err,
		)
	}

	gvr := schema.GroupVersionResource{
		Group:    v1alpha1.GroupVersion.Group,
		Version:  v1alpha1.GroupVersion.Version,
		Resource: "clusterruleractions",
	}

	rulerActionWrapper := globals.Application.KubeRawClient.Resource(gvr)
	if searchRule.Spec.ActionRef.Namespace != "" {
		gvr.Resource = "ruleractions"
		rulerActionWrapper = globals.Application.KubeRawClient.Resource(gvr)
		rulerActionWrapper.Namespace(searchRule.Spec.ActionRef.Namespace)
	}

	rulerActionResource, err := rulerActionWrapper.Get(ctx, searchRule.Spec.ActionRef.Name, metav1.GetOptions{})
	if err != nil {
		// TODO: Improve this
		return resourceType, err
	}

	// If RulerAction is empty then error
	if reflect.ValueOf(rulerActionResource).IsZero() {
		return resourceType, fmt.Errorf(
			"error fetching RulerAction %s from searchRule %s: %v",
			searchRule.Spec.ActionRef.Name,
			searchRuleNamespacedName,
			err,
		)
	}

	// Tricky for save RulerAction resource with RulerAction or ClusterRulerAction type
	specBytes, err := json.Marshal(rulerActionResource.Object)
	if err != nil {
		return resourceType, fmt.Errorf(controller.JSONMarshalErrorMessage, err)
	}
	switch searchRule.Spec.ActionRef.Namespace {
	case "":
		resourceType = controller.ClusterRulerActionResourceType
		err = json.Unmarshal(specBytes, ruleAction.ClusterRulerActionResource)
	default:
		resourceType = controller.RulerActionResourceType
		err = json.Unmarshal(specBytes, ruleAction.RulerActionResource)
	}
	if err != nil {
		return resourceType, fmt.Errorf(controller.JSONMarshalErrorMessage, err)
	}

	return resourceType, nil
}

// getRulerActionAssociatedAlerts returns all alerts associated with the RulerAction
func (r *RulerActionReconciler) getRulerActionAssociatedAlerts(resourceName string) (alerts []*pools.Alert, err error) {

	// Get all alerts from the AlertsPool
	alertsPool := r.AlertsPool.GetAll()

	// Iterate over the alerts in the pool and check if the alert is associated with the RulerAction
	for _, alert := range alertsPool {
		if alert.RulerActionName == resourceName {
			alerts = append(alerts, alert)
		}
	}

	return alerts, nil
}
