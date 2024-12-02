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
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	searchrulerv1alpha1 "prosimcorp.com/SearchRuler/api/v1alpha1"
	"prosimcorp.com/SearchRuler/internal/pools"
)

// QueryConnectorReconciler reconciles a QueryConnector object
type QueryConnectorReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	CredentialsPool *pools.CredentialsStore
}

// +kubebuilder:rbac:groups=searchruler.prosimcorp.com,resources=QueryConnectors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=searchruler.prosimcorp.com,resources=QueryConnectors/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=searchruler.prosimcorp.com,resources=QueryConnectors/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the QueryConnector object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *QueryConnectorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	logger := log.FromContext(ctx)

	// 1. Get the content of the Patch
	QueryConnectorResource := &searchrulerv1alpha1.QueryConnector{}
	err = r.Get(ctx, req.NamespacedName, QueryConnectorResource)

	// 2. Check existence on the cluster
	if err != nil {

		// 2.1 It does NOT exist: manage removal
		if err = client.IgnoreNotFound(err); err == nil {
			logger.Info(fmt.Sprintf(resourceNotFoundError, QueryConnectorResourceType, req.NamespacedName))
			return result, err
		}

		// 2.2 Failed to get the resource, requeue the request
		logger.Info(fmt.Sprintf(resourceSyncTimeRetrievalError, QueryConnectorResourceType, req.NamespacedName, err.Error()))
		return result, err
	}

	// 3. Check if the SearchRule instance is marked to be deleted: indicated by the deletion timestamp being set
	if !QueryConnectorResource.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(QueryConnectorResource, resourceFinalizer) {
			// Remove the finalizers on Patch CR
			controllerutil.RemoveFinalizer(QueryConnectorResource, resourceFinalizer)
			err = r.Update(ctx, QueryConnectorResource)
			if err != nil {
				logger.Info(fmt.Sprintf(resourceFinalizersUpdateError, QueryConnectorResourceType, req.NamespacedName, err.Error()))
			}
		}
		// Delete credentials from pool
		r.CredentialsPool.Delete(fmt.Sprintf("%s/%s", QueryConnectorResource.Namespace, QueryConnectorResource.Name))

		result = ctrl.Result{}
		err = nil
		return result, err
	}

	// 4. Add finalizer to the SearchRule CR
	if !controllerutil.ContainsFinalizer(QueryConnectorResource, resourceFinalizer) {
		controllerutil.AddFinalizer(QueryConnectorResource, resourceFinalizer)
		err = r.Update(ctx, QueryConnectorResource)
		if err != nil {
			return result, err
		}
	}

	// 5. Update the status before the requeue
	defer func() {
		err = r.Status().Update(ctx, QueryConnectorResource)
		if err != nil {
			logger.Info(fmt.Sprintf(resourceConditionUpdateError, QueryConnectorResourceType, req.NamespacedName, err.Error()))
		}
	}()

	// 6. Schedule periodical request
	if QueryConnectorResource.Spec.Credentials.SyncInterval == "" {
		QueryConnectorResource.Spec.Credentials.SyncInterval = defaultSyncInterval
	}
	RequeueTime, err := time.ParseDuration(QueryConnectorResource.Spec.Credentials.SyncInterval)
	if err != nil {
		logger.Info(fmt.Sprintf(resourceSyncTimeRetrievalError, QueryConnectorResourceType, req.NamespacedName, err.Error()))
		return result, err
	}
	result = ctrl.Result{
		RequeueAfter: RequeueTime,
	}

	// 7. Sync credentials if defined
	emptyCredentials := searchrulerv1alpha1.QueryConnectorCredentials{}
	if QueryConnectorResource.Spec.Credentials != emptyCredentials {
		err = r.SyncCredentials(ctx, QueryConnectorResource)
		if err != nil {
			r.UpdateConditionKubernetesApiCallFailure(QueryConnectorResource)
			logger.Info(fmt.Sprintf(syncTargetError, QueryConnectorResourceType, req.NamespacedName, err.Error()))
			return result, err
		}
	}

	// 8. Success, update the status
	r.UpdateConditionSuccess(QueryConnectorResource)

	return result, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *QueryConnectorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&searchrulerv1alpha1.QueryConnector{}).
		Named("QueryConnector").
		Complete(r)
}
