package controller

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"prosimcorp.com/SearchRuler/api/v1alpha1"
	"prosimcorp.com/SearchRuler/internal/template"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Sync
func (r *RulerActionReconciler) Sync(ctx context.Context, resource *v1alpha1.RulerAction) (err error) {

	logger := log.FromContext(ctx)

	// Get credentials for the Action in the secret associated if defined
	username := ""
	password := ""
	emptyCredentials := v1alpha1.RulerActionCredentials{}
	if resource.Spec.Webhook.Credentials != emptyCredentials {
		// First get secret with the credentials
		RulerActionCredsSecret := &v1.Secret{}
		namespacedName := types.NamespacedName{
			Namespace: resource.Namespace,
			Name:      resource.Spec.Webhook.Credentials.SecretRef.Name,
		}
		err = r.Get(ctx, namespacedName, RulerActionCredsSecret)
		if err != nil {
			return fmt.Errorf("error fetching secret %s: %v", namespacedName, err)
		}

		// Get username and password
		username = string(RulerActionCredsSecret.Data[resource.Spec.Webhook.Credentials.SecretRef.KeyUsername])
		password = string(RulerActionCredsSecret.Data[resource.Spec.Webhook.Credentials.SecretRef.KeyPassword])
		if username == "" || password == "" {
			return fmt.Errorf("missing credentials in secret %s", namespacedName)
		}
	}

	// Check alerts
	logger.Info(fmt.Sprintf("Triggered by: %s", resource.Name))
	alerts := r.AlertsPool.GetByRegex(fmt.Sprintf("%s/%s/*", resource.Namespace, resource.Name))
	for _, alert := range alerts {

		logger.Info(fmt.Sprintf("Alert: %s", alert.SearchRule.Spec.Description))
		httpClient := &http.Client{}

		// Create the request
		httpRequest, err := http.NewRequest(resource.Spec.Webhook.Verb, resource.Spec.Webhook.Url, nil)
		if err != nil {
			return fmt.Errorf("error %v", err)
		}

		// Add headers to the request
		for headerKey, headerValue := range resource.Spec.Webhook.Headers {
			httpRequest.Header.Set(headerKey, headerValue)
		}

		// Add data to the request
		templateInjectedObject := map[string]interface{}{}
		templateInjectedObject["value"] = alert.Value
		templateInjectedObject["object"] = alert.SearchRule

		parsedMessage, err := template.EvaluateTemplate(alert.SearchRule.Spec.ActionRef.Data, templateInjectedObject)
		if err != nil {
			return fmt.Errorf("error evaluating template message: %v", err)
		}
		payload := []byte(parsedMessage)

		httpRequest.Body = io.NopCloser(bytes.NewBuffer(payload))
		httpRequest.Header.Set("Content-Type", "application/json")

		// Add authentication if set for elasticsearch queries
		if username == "" || password == "" {
			httpRequest.SetBasicAuth(username, password)
		}

		// Send HTTP request
		httpResponse, err := httpClient.Do(httpRequest)
		if err != nil {
			return fmt.Errorf("error %v", err)
		}
		defer httpResponse.Body.Close()

		//
		return err
	}

	return nil
}
