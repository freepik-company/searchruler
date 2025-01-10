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

package queryconnector

import (
	"context"
	"fmt"
	"reflect"
	"time"

	//
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	//
	searchrulerv1alpha1 "prosimcorp.com/SearchRuler/api/v1alpha1"
	"prosimcorp.com/SearchRuler/internal/controller"
	"prosimcorp.com/SearchRuler/internal/pools"
)

// QueryConnectorReconciler reconciles a QueryConnector object
type QueryConnectorReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	CredentialsPool *pools.CredentialsStore
}

type CompoundQueryConnectorResource struct {
	QueryConnectorResource        *searchrulerv1alpha1.QueryConnector
	ClusterQueryConnectorResource *searchrulerv1alpha1.ClusterQueryConnector
}

var (
	resourceType      string
	containsFinalizer bool
)

// +kubebuilder:rbac:groups=searchruler.prosimcorp.com,resources=queryconnectors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=searchruler.prosimcorp.com,resources=queryconnectors/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=searchruler.prosimcorp.com,resources=queryconnectors/finalizers,verbs=update

// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *QueryConnectorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	logger := log.FromContext(ctx)

	// 1. Get the content of the Patch
	CompoundQueryConnectorResource := &CompoundQueryConnectorResource{
		QueryConnectorResource:        &searchrulerv1alpha1.QueryConnector{},
		ClusterQueryConnectorResource: &searchrulerv1alpha1.ClusterQueryConnector{},
	}
	switch req.Namespace {
	case "":
		resourceType = controller.ClusterQueryConnectorResourceType
		err = r.Get(ctx, req.NamespacedName, CompoundQueryConnectorResource.ClusterQueryConnectorResource)
	default:
		resourceType = controller.QueryConnectorResourceType
		err = r.Get(ctx, req.NamespacedName, CompoundQueryConnectorResource.QueryConnectorResource)
	}

	// 2. Check existence on the cluster
	if err != nil {

		// 2.1 It does NOT exist: manage removal
		if err = client.IgnoreNotFound(err); err == nil {
			logger.Info(fmt.Sprintf(controller.ResourceNotFoundError, resourceType, req.NamespacedName))
			return result, err
		}

		// 2.2 Failed to get the resource, requeue the request
		logger.Info(fmt.Sprintf(controller.ResourceSyncTimeRetrievalError, resourceType, req.NamespacedName, err.Error()))
		return result, err
	}

	// 3. Check if the SearchRule instance is marked to be deleted: indicated by the deletion timestamp being set
	deletionTimestamp := &v1.Time{}
	switch resourceType {
	case controller.ClusterQueryConnectorResourceType:
		deletionTimestamp = CompoundQueryConnectorResource.ClusterQueryConnectorResource.DeletionTimestamp
		containsFinalizer = controllerutil.ContainsFinalizer(CompoundQueryConnectorResource.ClusterQueryConnectorResource, controller.ResourceFinalizer)
	default:
		deletionTimestamp = CompoundQueryConnectorResource.QueryConnectorResource.DeletionTimestamp
		containsFinalizer = controllerutil.ContainsFinalizer(CompoundQueryConnectorResource.QueryConnectorResource, controller.ResourceFinalizer)
	}
	if !deletionTimestamp.IsZero() {
		if containsFinalizer {

			// 3.1 Delete the resources associated with the QueryConnector
			err = r.Sync(ctx, watch.Deleted, CompoundQueryConnectorResource, resourceType)

			// Remove the finalizers on Patch CR
			switch resourceType {
			case controller.ClusterQueryConnectorResourceType:
				controllerutil.RemoveFinalizer(CompoundQueryConnectorResource.ClusterQueryConnectorResource, controller.ResourceFinalizer)
				err = r.Update(ctx, CompoundQueryConnectorResource.ClusterQueryConnectorResource)
			default:
				controllerutil.RemoveFinalizer(CompoundQueryConnectorResource.QueryConnectorResource, controller.ResourceFinalizer)
				err = r.Update(ctx, CompoundQueryConnectorResource.QueryConnectorResource)
			}
			if err != nil {
				logger.Info(fmt.Sprintf(controller.ResourceFinalizersUpdateError, resourceType, req.NamespacedName, err.Error()))
			}
		}

		result = ctrl.Result{}
		err = nil
		return result, err
	}

	// 4. Add finalizer to the SearchRule CR
	if !containsFinalizer {
		switch resourceType {
		case controller.ClusterQueryConnectorResourceType:
			controllerutil.AddFinalizer(CompoundQueryConnectorResource.ClusterQueryConnectorResource, controller.ResourceFinalizer)
			err = r.Update(ctx, CompoundQueryConnectorResource.ClusterQueryConnectorResource)
		default:
			controllerutil.AddFinalizer(CompoundQueryConnectorResource.QueryConnectorResource, controller.ResourceFinalizer)
			err = r.Update(ctx, CompoundQueryConnectorResource.QueryConnectorResource)
		}
		if err != nil {
			return result, err
		}
	}

	// 5. Update the status before the requeue
	defer func() {
		switch resourceType {
		case controller.ClusterQueryConnectorResourceType:
			err = r.Status().Update(ctx, CompoundQueryConnectorResource.ClusterQueryConnectorResource)
		default:
			err = r.Status().Update(ctx, CompoundQueryConnectorResource.QueryConnectorResource)
		}
		if err != nil {
			logger.Info(fmt.Sprintf(controller.ResourceConditionUpdateError, resourceType, req.NamespacedName, err.Error()))
		}
	}()

	// 6. Schedule periodical request
	syncInterval := controller.DefaultSyncInterval
	switch resourceType {
	case controller.ClusterQueryConnectorResourceType:
		if !reflect.ValueOf(CompoundQueryConnectorResource.ClusterQueryConnectorResource.Spec.Credentials.SyncInterval).IsZero() {
			syncInterval = CompoundQueryConnectorResource.ClusterQueryConnectorResource.Spec.Credentials.SyncInterval
		}
	default:
		if !reflect.ValueOf(CompoundQueryConnectorResource.QueryConnectorResource.Spec.Credentials.SyncInterval).IsZero() {
			syncInterval = CompoundQueryConnectorResource.QueryConnectorResource.Spec.Credentials.SyncInterval
		}
	}

	RequeueTime, err := time.ParseDuration(syncInterval)
	if err != nil {
		logger.Info(fmt.Sprintf(controller.ResourceSyncTimeRetrievalError, resourceType, req.NamespacedName, err.Error()))
		return result, err
	}
	result = ctrl.Result{
		RequeueAfter: RequeueTime,
	}

	// 7. Sync credentials if defined
	credentials := CompoundQueryConnectorResource.QueryConnectorResource.Spec.Credentials
	if resourceType == controller.ClusterQueryConnectorResourceType {
		credentials = CompoundQueryConnectorResource.ClusterQueryConnectorResource.Spec.Credentials
	}

	if !reflect.ValueOf(credentials).IsZero() {
		err = r.Sync(ctx, watch.Modified, CompoundQueryConnectorResource, resourceType)
		if err != nil {
			r.UpdateConditionKubernetesApiCallFailure(CompoundQueryConnectorResource, resourceType)
			logger.Info(fmt.Sprintf(controller.SyncTargetError, resourceType, req.NamespacedName, err.Error()))
			return result, err
		}
	}

	// 8. Success, update the status
	r.UpdateConditionSuccess(CompoundQueryConnectorResource, resourceType)

	return result, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *QueryConnectorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&searchrulerv1alpha1.QueryConnector{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Named("QueryConnector").
		Watches(&searchrulerv1alpha1.ClusterQueryConnector{}, &handler.EnqueueRequestForObject{}).
		Complete(r)
}
