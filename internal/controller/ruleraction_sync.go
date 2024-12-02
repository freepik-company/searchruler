package controller

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"prosimcorp.com/SearchRuler/api/v1alpha1"
)

// SyncCredentials
func (r *RulerActionReconciler) SyncCredentials(ctx context.Context, resource *v1alpha1.RulerAction) (err error) {

	// Get credentials for the Action in the secret associated
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
	username := string(RulerActionCredsSecret.Data[resource.Spec.Webhook.Credentials.SecretRef.KeyUsername])
	password := string(RulerActionCredsSecret.Data[resource.Spec.Webhook.Credentials.SecretRef.KeyPassword])
	if username == "" || password == "" {
		return fmt.Errorf("missing credentials in secret %s", namespacedName)
	}

	// Save in pool
	key := fmt.Sprintf("%s/%s", resource.Namespace, resource.Name)
	RulerActionCredentialsPool.Set(key, &Credentials{
		Username: username,
		Password: password,
	})

	return nil
}

// DeleteCredentials
func (r *RulerActionReconciler) DeleteCredentials(ctx context.Context, resource *v1alpha1.RulerAction) (err error) {

	// Delete from global map
	key := fmt.Sprintf("%s/%s", resource.Namespace, resource.Name)
	RulerActionCredentialsPool.Delete(key)

	return nil
}
