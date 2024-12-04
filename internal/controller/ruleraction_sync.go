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

package controller

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"prosimcorp.com/SearchRuler/api/v1alpha1"
	"prosimcorp.com/SearchRuler/internal/pools"
	"prosimcorp.com/SearchRuler/internal/template"
	"prosimcorp.com/SearchRuler/internal/validators"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	// validatorsMap is a map of integration names and their respective validation functions
	validatorsMap = map[string]func(data string) (result bool, hint string, err error){
		"alertmanager": validators.ValidateAlertmanager,
	}
)

// Sync function is used to synchronize the RulerAction resource with the alerts. Executes the webhook defined in the
// resource for each alert found in the AlertsPool.
func (r *RulerActionReconciler) Sync(ctx context.Context, resource *v1alpha1.RulerAction) (err error) {

	logger := log.FromContext(ctx)

	// Get credentials for the Action in the secret associated if defined
	username := ""
	password := ""
	if !reflect.ValueOf(resource.Spec.Webhook.Credentials).IsZero() {
		// First get secret with the credentials
		RulerActionCredsSecret := &corev1.Secret{}
		namespacedName := types.NamespacedName{
			Namespace: resource.Namespace,
			Name:      resource.Spec.Webhook.Credentials.SecretRef.Name,
		}
		err = r.Get(ctx, namespacedName, RulerActionCredsSecret)
		if err != nil {
			r.UpdateConditionNoCredsFound(resource)
			return fmt.Errorf(SecretNotFoundErrorMessage, namespacedName, err)
		}

		// Get username and password
		username = string(RulerActionCredsSecret.Data[resource.Spec.Webhook.Credentials.SecretRef.KeyUsername])
		password = string(RulerActionCredsSecret.Data[resource.Spec.Webhook.Credentials.SecretRef.KeyPassword])
		if username == "" || password == "" {
			r.UpdateConditionNoCredsFound(resource)
			return fmt.Errorf(MissingCredentialsMessage, namespacedName)
		}
	}

	// Check alert pool for alerts related to this rulerAction
	// Alerts key pattern: namespace/rulerActionName/searchRuleName
	alerts, err := r.getRulerActionAssociatedAlerts(resource)
	if err != nil {
		return fmt.Errorf(AlertsPoolErrorMessage, err)
	}

	// If there are alerts for the rulerAction, initialice the HTTP client
	if len(alerts) > 0 {
		// Create the HTTP client
		httpClient := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: resource.Spec.Webhook.TlsSkipVerify,
				},
			},
		}

		// Create the request with the configured verb and URL
		httpRequest, err := http.NewRequest(resource.Spec.Webhook.Verb, resource.Spec.Webhook.Url, nil)
		if err != nil {
			return fmt.Errorf(HttpRequestCreationErrorMessage, err)
		}

		// Add headers to the request if set
		httpRequest.Header.Set("Content-Type", "application/json")
		for headerKey, headerValue := range resource.Spec.Webhook.Headers {
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
				AlertFiringInfoMessage,
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

			// Evaluate the data template with the injected object
			parsedMessage, err := template.EvaluateTemplate(alert.SearchRule.Spec.ActionRef.Data, templateInjectedObject)
			if err != nil {
				r.UpdateConditionEvaluateTemplateError(resource)
				return fmt.Errorf(EvaluateTemplateErrorMessage, err)
			}

			// Check if the webhook has a validator and execute it when available
			if resource.Spec.Webhook.Validator != "" {

				// Check if the validator is available
				_, validatorFound := validatorsMap[resource.Spec.Webhook.Validator]
				if !validatorFound {
					r.UpdateConditionEvaluateTemplateError(resource)
					return fmt.Errorf(ValidatorNotFoundErrorMessage, resource.Spec.Webhook.Validator)
				}

				// Execute the validator to the data of the alert
				validatorResult, validatorHint, err := validatorsMap[resource.Spec.Webhook.Validator](parsedMessage)
				if err != nil {
					r.UpdateConditionEvaluateTemplateError(resource)
					return fmt.Errorf(ValidationFailedErrorMessage, err.Error())
				}

				// Check the result of the validator
				if !validatorResult {
					r.UpdateConditionEvaluateTemplateError(resource)
					return fmt.Errorf(ValidationFailedErrorMessage, validatorHint)
				}
			}

			// Add data to the payload of the request
			payload := []byte(parsedMessage)
			httpRequest.Body = io.NopCloser(bytes.NewBuffer(payload))

			// Send HTTP request to the webhook
			httpResponse, err := httpClient.Do(httpRequest)
			if err != nil {
				r.UpdateConditionConnectionError(resource)
				return fmt.Errorf(HttpRequestSendingErrorMessage, err)
			}

			defer httpResponse.Body.Close()
		}
	}

	// Updates status to Success
	r.UpdateStateSuccess(resource)
	return nil
}

// GetRuleActionFromEvent returns the RulerAction resource associated with the event that triggered the reconcile
func (r *RulerActionReconciler) GetEventRuleAction(ctx context.Context, namespace, name string) (ruleAction v1alpha1.RulerAction, err error) {

	// Get event resource from the namespace and name of the event that triggered the reconcile
	EventResource := &corev1.Event{}
	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	err = r.Get(ctx, namespacedName, EventResource)
	if err != nil {
		return ruleAction, fmt.Errorf(
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
		return ruleAction, fmt.Errorf(
			"error fetching SearchRule %s from event %s: %v",
			searchRuleNamespacedName,
			namespacedName,
			err,
		)
	}

	// Get RulerAction resource from searchRule resource
	ruleAction = v1alpha1.RulerAction{}
	ruleActionNamespacedName := types.NamespacedName{
		Namespace: searchRule.Namespace,
		Name:      searchRule.Spec.ActionRef.Name,
	}
	err = r.Get(ctx, ruleActionNamespacedName, &ruleAction)
	if err != nil {
		return ruleAction, fmt.Errorf(
			"error fetching RulerAction %s from searchRule %s: %v",
			ruleActionNamespacedName,
			searchRuleNamespacedName,
			err,
		)
	}

	return ruleAction, nil
}

// getRulerActionAssociatedAlerts returns all alerts associated with the RulerAction
func (r *RulerActionReconciler) getRulerActionAssociatedAlerts(resource *v1alpha1.RulerAction) (alerts []*pools.Alert, err error) {

	// Get all alerts from the AlertsPool
	alertsPool := r.AlertsPool.GetAll()

	// Iterate over the alerts in the pool and check if the alert is associated with the RulerAction
	for _, alert := range alertsPool {
		if alert.RulerActionName == resource.Name {
			alerts = append(alerts, alert)
		}
	}

	return alerts, nil
}
