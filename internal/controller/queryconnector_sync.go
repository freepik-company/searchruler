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
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"prosimcorp.com/SearchRuler/api/v1alpha1"
	"prosimcorp.com/SearchRuler/internal/pools"
)

// Sync function is used to synchronize the QueryConnector resource with the credentials. Adds the credentials to the
// credentials pool to be used in SearchRule resources. Just executed when the resource has a secretRef defined.
func (r *QueryConnectorReconciler) Sync(ctx context.Context, resource *v1alpha1.QueryConnector) (err error) {

	// Get credentials for the queryConnector in the secret associated
	// First get secret with the credentials. The secret must be in the same
	// namespace as the QueryConnector resource.
	QueryConnectorCredsSecret := &v1.Secret{}
	namespacedName := types.NamespacedName{
		Namespace: resource.Namespace,
		Name:      resource.Spec.Credentials.SecretRef.Name,
	}
	err = r.Get(ctx, namespacedName, QueryConnectorCredsSecret)
	if err != nil {
		// Updates status to NoCredsFound
		r.UpdateConditionNoCredsFound(resource)
		return fmt.Errorf("error fetching secret %s: %v", namespacedName, err)
	}

	// Get username and password from the secret data
	username := string(QueryConnectorCredsSecret.Data[resource.Spec.Credentials.SecretRef.KeyUsername])
	password := string(QueryConnectorCredsSecret.Data[resource.Spec.Credentials.SecretRef.KeyPassword])

	// If username or password are empty, return an error
	if username == "" || password == "" {
		// Updates status to NoCredsFound
		r.UpdateConditionNoCredsFound(resource)
		return fmt.Errorf("missing credentials in secret %s", namespacedName)
	}

	// Save credentials in the credentials pool
	key := fmt.Sprintf("%s/%s", resource.Namespace, resource.Name)
	r.CredentialsPool.Set(key, &pools.Credentials{
		Username: username,
		Password: password,
	})

	return nil
}
