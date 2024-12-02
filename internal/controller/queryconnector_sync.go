package controller

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"prosimcorp.com/SearchRuler/api/v1alpha1"
	"prosimcorp.com/SearchRuler/internal/pools"
)

// SyncCredentials
func (r *QueryConnectorReconciler) SyncCredentials(ctx context.Context, resource *v1alpha1.QueryConnector) (err error) {

	// Get credentials for the queryConnector in the secret associated
	// First get secret with the credentials
	QueryConnectorCredsSecret := &v1.Secret{}
	namespacedName := types.NamespacedName{
		Namespace: resource.Namespace,
		Name:      resource.Spec.Credentials.SecretRef.Name,
	}
	err = r.Get(ctx, namespacedName, QueryConnectorCredsSecret)
	if err != nil {
		return fmt.Errorf("error fetching secret %s: %v", namespacedName, err)
	}

	// Get username and password
	username := string(QueryConnectorCredsSecret.Data[resource.Spec.Credentials.SecretRef.KeyUsername])
	password := string(QueryConnectorCredsSecret.Data[resource.Spec.Credentials.SecretRef.KeyPassword])
	if username == "" || password == "" {
		return fmt.Errorf("missing credentials in secret %s", namespacedName)
	}

	// Save in pool
	key := fmt.Sprintf("%s/%s", resource.Namespace, resource.Name)
	r.CredentialsPool.Set(key, &pools.Credentials{
		Username: username,
		Password: password,
	})

	return nil
}
