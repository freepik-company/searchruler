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

package searchrule

import (
	"context"
	"fmt"
	"time"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	//
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	//
	searchrulerv1alpha1 "freepik.com/searchruler/api/v1alpha1"
	"freepik.com/searchruler/internal/controller"
	"freepik.com/searchruler/internal/pools"
)

// SearchRuleReconciler reconciles a SearchRule object
type SearchRuleReconciler struct {
	client.Client
	Scheme                        *runtime.Scheme
	QueryConnectorCredentialsPool *pools.CredentialsStore
	RulesPool                     *pools.RulesStore
	AlertsPool                    *pools.AlertsStore

	// PrometheusRuleSupported indicates whether the cluster has the
	// monitoring.coreos.com/v1 PrometheusRule CRD installed. Detected once at
	// boot time. When false, SearchRules that opt into spec.prometheusRule are
	// reconciled but the reconciler reports the feature as unsupported instead
	// of trying to create the resource.
	PrometheusRuleSupported bool

	// MetricsExposed indicates whether the operator was started with the
	// custom-metrics endpoint enabled (--rules-metrics-bind-address != "0").
	// When false, generated PrometheusRules will alert on a metric that is
	// not exposed; we still create them but flag the SearchRule status with a
	// MetricsNotExposed condition so the user can spot the misconfiguration.
	MetricsExposed bool
}

// +kubebuilder:rbac:groups=searchruler.freepik.com,resources=searchrules,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=searchruler.freepik.com,resources=searchrules/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=searchruler.freepik.com,resources=searchrules/finalizers,verbs=update

// +kubebuilder:rbac:groups="events.k8s.io",resources=events,verbs=get;list;watch;create;update;patch

// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=prometheusrules,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *SearchRuleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	logger := log.FromContext(ctx)

	// 1. Get the content of the Patch
	searchRuleResource := &searchrulerv1alpha1.SearchRule{}
	err = r.Get(ctx, req.NamespacedName, searchRuleResource)

	// 2. Check existence on the cluster
	if err != nil {

		// 2.1 It does NOT exist: manage removal
		if err = client.IgnoreNotFound(err); err == nil {
			logger.Info(fmt.Sprintf(controller.ResourceNotFoundError, controller.SearchRuleResourceType, req.NamespacedName))
			return result, err
		}

		// 2.2 Failed to get the resource, requeue the request
		logger.Info(fmt.Sprintf(controller.ResourceSyncTimeRetrievalError, controller.SearchRuleResourceType, req.NamespacedName, err.Error()))
		return result, err
	}

	// 3. Check if the SearchRule instance is marked to be deleted: indicated by the deletion timestamp being set
	if !searchRuleResource.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(searchRuleResource, controller.ResourceFinalizer) {

			// 3.1 Delete the resources associated with the SearchRule
			err = r.Sync(ctx, watch.Deleted, searchRuleResource)

			// Remove the finalizers on Patch CR
			controllerutil.RemoveFinalizer(searchRuleResource, controller.ResourceFinalizer)
			err = r.Update(ctx, searchRuleResource)
			if err != nil {
				logger.Info(fmt.Sprintf(controller.ResourceFinalizersUpdateError, controller.SearchRuleResourceType, req.NamespacedName, err.Error()))
			}
		}

		result = ctrl.Result{}
		err = nil
		return result, err
	}

	// 4. Add finalizer to the SearchRule CR
	if !controllerutil.ContainsFinalizer(searchRuleResource, controller.ResourceFinalizer) {
		controllerutil.AddFinalizer(searchRuleResource, controller.ResourceFinalizer)
		err = r.Update(ctx, searchRuleResource)
		if err != nil {
			return result, err
		}
	}

	// 5. Update the status before the requeue
	defer func() {
		err = r.Status().Update(ctx, searchRuleResource)
		if err != nil {
			logger.Info(fmt.Sprintf(controller.ResourceConditionUpdateError, controller.SearchRuleResourceType, req.NamespacedName, err.Error()))
		}
	}()

	// 6. Validate that at least one output is defined. A SearchRule whose only
	// purpose is to update its own status (without actionRef and without
	// prometheusRule) silently produces nothing useful, so flag it.
	if searchRuleResource.Spec.ActionRef == nil && searchRuleResource.Spec.PrometheusRule == nil {
		r.UpdateConditionMissingOutput(searchRuleResource)
		return ctrl.Result{}, fmt.Errorf("searchrule %s/%s has no actionRef nor prometheusRule defined",
			searchRuleResource.Namespace, searchRuleResource.Name)
	}

	// 7. Reconcile the auto-generated PrometheusRule (no-op when not enabled).
	if err := r.reconcilePrometheusRule(ctx, searchRuleResource); err != nil {
		logger.Info(fmt.Sprintf("failed to reconcile PrometheusRule for %s: %s",
			req.NamespacedName, err.Error()))
		return result, err
	}

	// 8. Schedule periodical request
	RequeueTime, err := time.ParseDuration(searchRuleResource.Spec.CheckInterval)
	if err != nil {
		logger.Info(fmt.Sprintf(controller.ResourceSyncTimeRetrievalError, controller.SearchRuleResourceType, req.NamespacedName, err.Error()))
		return result, err
	}
	result = ctrl.Result{
		RequeueAfter: RequeueTime,
	}

	// 9. Check the rule
	err = r.Sync(ctx, watch.Modified, searchRuleResource)
	if err != nil {
		r.UpdateConditionKubernetesApiCallFailure(searchRuleResource)
		logger.Info(fmt.Sprintf(controller.SyncTargetError, controller.SearchRuleResourceType, req.NamespacedName, err.Error()))
		return result, err
	}

	// 10. Success, update the status
	r.UpdateConditionSuccess(searchRuleResource)

	return result, err

}

// SetupWithManager sets up the controller with the Manager.
func (r *SearchRuleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	b := ctrl.NewControllerManagedBy(mgr).
		For(&searchrulerv1alpha1.SearchRule{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Named("searchrule")
	// Only watch PrometheusRule when the CRD exists, otherwise controller-runtime
	// will fail to set up the informer with a NoMatch error.
	if r.PrometheusRuleSupported {
		b = b.Owns(&monitoringv1.PrometheusRule{}, builder.MatchEveryOwner)
	}
	return b.Complete(r)
}
