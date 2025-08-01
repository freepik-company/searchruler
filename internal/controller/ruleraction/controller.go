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
	"context"
	"fmt"
	"reflect"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	//
	searchrulerv1alpha1 "freepik.com/searchruler/api/v1alpha1"
	"freepik.com/searchruler/internal/controller"
	"freepik.com/searchruler/internal/pools"
)

// RulerActionReconciler reconciles a RulerAction object
type RulerActionReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	AlertsPool *pools.AlertsStore
}

type CompoundRulerActionResource struct {
	RulerActionResource        *searchrulerv1alpha1.RulerAction
	ClusterRulerActionResource *searchrulerv1alpha1.ClusterRulerAction
}

var (
	resourceType      string
	containsFinalizer bool
	deletionTimestamp *v1.Time
)

// +kubebuilder:rbac:groups=searchruler.freepik.com,resources=ruleractions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=searchruler.freepik.com,resources=ruleractions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=searchruler.freepik.com,resources=ruleractions/finalizers,verbs=update

// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *RulerActionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {

	logger := log.FromContext(ctx)

	// 1. Get the content of the Patch
	CompoundRulerActionResource := &CompoundRulerActionResource{
		RulerActionResource:        &searchrulerv1alpha1.RulerAction{},
		ClusterRulerActionResource: &searchrulerv1alpha1.ClusterRulerAction{},
	}

	// 1 Get the RulerAction or ClusterRulerAction resource
	switch req.Namespace {
	case "":
		resourceType = controller.ClusterRulerActionResourceType
		err = r.Get(ctx, req.NamespacedName, CompoundRulerActionResource.ClusterRulerActionResource)
	default:
		resourceType = controller.RulerActionResourceType
		err = r.Get(ctx, req.NamespacedName, CompoundRulerActionResource.RulerActionResource)
	}

	// 2. Check existence on the cluster
	if err != nil {

		// 2.1 It does NOT exist: manage removal
		if err = client.IgnoreNotFound(err); err == nil {
			logger.Info(fmt.Sprintf(controller.ResourceNotFoundError, controller.RulerActionResourceType, req.NamespacedName))
			return result, err
		}

		// 2.2 Failed to get the resource, requeue the request
		logger.Info(fmt.Sprintf(controller.CanNotGetResourceError, controller.RulerActionResourceType, req.NamespacedName, err.Error()))
		return result, err
	}

	// 3. Check if the RulerAction instance is marked to be deleted: indicated by the deletion timestamp being set
	switch resourceType {
	case controller.ClusterRulerActionResourceType:
		deletionTimestamp = CompoundRulerActionResource.ClusterRulerActionResource.DeletionTimestamp
		containsFinalizer = controllerutil.ContainsFinalizer(CompoundRulerActionResource.ClusterRulerActionResource, controller.ResourceFinalizer)
	default:
		deletionTimestamp = CompoundRulerActionResource.RulerActionResource.DeletionTimestamp
		containsFinalizer = controllerutil.ContainsFinalizer(CompoundRulerActionResource.RulerActionResource, controller.ResourceFinalizer)
	}
	if !deletionTimestamp.IsZero() {
		if containsFinalizer {

			// Remove the finalizers on Patch CR
			switch resourceType {
			case controller.ClusterRulerActionResourceType:
				controllerutil.RemoveFinalizer(CompoundRulerActionResource.ClusterRulerActionResource, controller.ResourceFinalizer)
				err = r.Update(ctx, CompoundRulerActionResource.ClusterRulerActionResource)
			default:
				controllerutil.RemoveFinalizer(CompoundRulerActionResource.RulerActionResource, controller.ResourceFinalizer)
				err = r.Update(ctx, CompoundRulerActionResource.RulerActionResource)
			}
			if err != nil {
				logger.Info(fmt.Sprintf(controller.ResourceFinalizersUpdateError, resourceType, req.NamespacedName, err.Error()))
			}
		}

		result = ctrl.Result{}
		err = nil
		return result, err
	}

	// 4. Add finalizer to the RulerAction CR
	if !containsFinalizer {
		switch resourceType {
		case controller.ClusterRulerActionResourceType:
			controllerutil.AddFinalizer(CompoundRulerActionResource.ClusterRulerActionResource, controller.ResourceFinalizer)
			err = r.Update(ctx, CompoundRulerActionResource.ClusterRulerActionResource)
		default:
			controllerutil.AddFinalizer(CompoundRulerActionResource.RulerActionResource, controller.ResourceFinalizer)
			err = r.Update(ctx, CompoundRulerActionResource.RulerActionResource)
		}
		if err != nil {
			return result, err
		}
	}

	// 5. Update the status before the requeue
	defer func() {
		switch resourceType {
		case controller.ClusterRulerActionResourceType:
			err = r.Status().Update(ctx, CompoundRulerActionResource.ClusterRulerActionResource)
		default:
			err = r.Status().Update(ctx, CompoundRulerActionResource.RulerActionResource)
		}
		if err != nil {
			logger.Info(fmt.Sprintf(controller.ResourceConditionUpdateError, resourceType, req.NamespacedName, err.Error()))
		}
	}()

	// 6. Schedule periodical request
	syncInterval := controller.DefaultSyncIntervalRulerAction
	switch resourceType {
	case controller.ClusterQueryConnectorResourceType:
		if !reflect.ValueOf(CompoundRulerActionResource.ClusterRulerActionResource.Spec.SyncInterval).IsZero() {
			syncInterval = CompoundRulerActionResource.ClusterRulerActionResource.Spec.SyncInterval
		}
	default:
		if !reflect.ValueOf(CompoundRulerActionResource.RulerActionResource.Spec.SyncInterval).IsZero() {
			syncInterval = CompoundRulerActionResource.RulerActionResource.Spec.SyncInterval
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
	err = r.Sync(ctx, CompoundRulerActionResource, resourceType)
	if err != nil {
		r.UpdateConditionKubernetesApiCallFailure(CompoundRulerActionResource, resourceType)
		logger.Info(fmt.Sprintf(controller.SyncTargetError, controller.RulerActionResourceType, req.NamespacedName, err.Error()))
		return result, err
	}

	// 8. Success, update the status
	r.UpdateConditionSuccess(CompoundRulerActionResource, resourceType)

	return result, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *RulerActionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&searchrulerv1alpha1.RulerAction{}).
		Named("RulerAction").
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Watches(&searchrulerv1alpha1.ClusterRulerAction{}, &handler.EnqueueRequestForObject{}).
		Complete(r)
}
