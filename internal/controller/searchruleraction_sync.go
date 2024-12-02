package controller

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"prosimcorp.com/SearchRuler/api/v1alpha1"
)

// SyncCredentials
func (r *SearchRulerActionReconciler) SyncCredentials(ctx context.Context, resource *v1alpha1.SearchRulerAction) (err error) {

	// Get credentials for the Action in the secret associated
	// First get secret with the credentials
	searchRulerActionCredsSecret := &v1.Secret{}
	namespacedName := types.NamespacedName{
		Namespace: resource.Namespace,
		Name:      resource.Spec.Webhook.Credentials.SecretRef.Name,
	}
	err = r.Get(ctx, namespacedName, searchRulerActionCredsSecret)
	if err != nil {
		return fmt.Errorf("error fetching secret %s: %v", namespacedName, err)
	}

	// Get username and password
	username := string(searchRulerActionCredsSecret.Data[resource.Spec.Webhook.Credentials.SecretRef.KeyUsername])
	password := string(searchRulerActionCredsSecret.Data[resource.Spec.Webhook.Credentials.SecretRef.KeyPassword])
	if username == "" || password == "" {
		return fmt.Errorf("missing credentials in secret %s", namespacedName)
	}

	// Save in pool
	key := fmt.Sprintf("%s/%s", resource.Namespace, resource.Name)
	SearchRulerActionCredentialsPool.Set(key, &Credentials{
		Username: username,
		Password: password,
	})

	return nil
}

// SyncCredentials
func (r *SearchRulerActionReconciler) DeleteCredentials(ctx context.Context, resource *v1alpha1.SearchRulerAction) (err error) {

	// Delete from global map
	key := fmt.Sprintf("%s/%s", resource.Namespace, resource.Name)
	SearchRulerActionCredentialsPool.Delete(key)

	return nil
}
