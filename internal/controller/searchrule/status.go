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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	//
	"freepik.com/searchruler/api/v1alpha1"
	"freepik.com/searchruler/internal/globals"
)

// UpdateConditionSuccess updates the status of the SearchRule resource with a success condition
func (r *SearchRuleReconciler) UpdateConditionSuccess(SearchRule *v1alpha1.SearchRule) {

	// Create the new condition with the success status
	condition := globals.NewCondition(globals.ConditionTypeResourceSynced, metav1.ConditionTrue,
		globals.ConditionReasonTargetSynced, globals.ConditionReasonTargetSyncedMessage)

	// Update the status of the SearchRule resource
	globals.UpdateCondition(&SearchRule.Status.Conditions, condition)
}

// UpdateConditionKubernetesApiCallFailure updates the status of the SearchRule resource with a failure condition
func (r *SearchRuleReconciler) UpdateConditionKubernetesApiCallFailure(SearchRule *v1alpha1.SearchRule) {

	// Create the new condition with the failure status
	condition := globals.NewCondition(globals.ConditionTypeResourceSynced, metav1.ConditionTrue,
		globals.ConditionReasonKubernetesApiCallErrorType, globals.ConditionReasonKubernetesApiCallErrorMessage)

	// Update the status of the SearchRule resource
	globals.UpdateCondition(&SearchRule.Status.Conditions, condition)
}

// UpdateStateNormal updates the status of the SearchRule resource with a Normal condition
func (r *SearchRuleReconciler) UpdateStateNormal(SearchRule *v1alpha1.SearchRule) {

	// Create the new condition with the Normal status
	condition := globals.NewCondition(globals.ConditionTypeState, metav1.ConditionTrue,
		globals.ConditionReasonStateNormalType, globals.ConditionReasonStateNormalMessage)

	// Update the status of the SearchRule resource
	globals.UpdateCondition(&SearchRule.Status.Conditions, condition)
}

// UpdateConditionNoCredsFound updates the status of the SearchRule resource with alert firing condition
func (r *SearchRuleReconciler) UpdateConditionAlertFiring(searchRule *v1alpha1.SearchRule) {

	// Create the new condition with the alert firing status
	condition := globals.NewCondition(globals.ConditionTypeState, metav1.ConditionTrue,
		globals.ConditionReasonAlertFiring, globals.ConditionReasonAlertFiringMessage)

	// Update the status of the SearchRule resource
	globals.UpdateCondition(&searchRule.Status.Conditions, condition)
}

// UpdateStateAlertPendingFiring updates the status of the SearchRule resource with alert pending firing condition
func (r *SearchRuleReconciler) UpdateStateAlertPendingFiring(searchRule *v1alpha1.SearchRule) {

	// Create the new condition with the alert resolved status
	condition := globals.NewCondition(globals.ConditionTypeState, metav1.ConditionTrue,
		globals.ConditionReasonPendingAlertFiring, globals.ConditionReasonPendingAlertFiringMessage)

	// Update the status of the SearchRule resource
	globals.UpdateCondition(&searchRule.Status.Conditions, condition)
}

// UpdateStateAlertPendingResolved updates the status of the SearchRule resource with alert pending resolved condition
func (r *SearchRuleReconciler) UpdateStateAlertPendingResolved(searchRule *v1alpha1.SearchRule) {

	// Create the new condition with the alert resolved status
	condition := globals.NewCondition(globals.ConditionTypeState, metav1.ConditionTrue,
		globals.ConditionReasonPendingAlertResolved, globals.ConditionReasonPendingAlertResolvedMessage)

	// Update the status of the SearchRule resource
	globals.UpdateCondition(&searchRule.Status.Conditions, condition)
}

// UpdateConditionConnectionError updates the status of the SearchRule resource with a QueryConnector not found condition
func (r *SearchRuleReconciler) UpdateConditionQueryConnectorNotFound(searchRule *v1alpha1.SearchRule) {

	// Create the new condition with the alert firing status
	condition := globals.NewCondition(globals.ConditionTypeState, metav1.ConditionTrue,
		globals.ConditionReasonQueryConnectorNotFoundType, globals.ConditionReasonQueryConnectorNotFoundMessage)

	// Update the status of the SearchRule resource
	globals.UpdateCondition(&searchRule.Status.Conditions, condition)
}

// UpdateConditionNoCredsFound updates the status of the SearchRule resource with a NoCreds condition
func (r *SearchRuleReconciler) UpdateConditionNoCredsFound(SearchRule *v1alpha1.SearchRule) {

	// Create the new condition with the success status
	condition := globals.NewCondition(globals.ConditionTypeState, metav1.ConditionTrue,
		globals.ConditionReasonNoCredsFoundType, globals.ConditionReasonNoCredsFoundMessage)

	// Update the status of the SearchRule resource
	globals.UpdateCondition(&SearchRule.Status.Conditions, condition)
}

func (r *SearchRuleReconciler) UpdateConditionNoQueryFound(SearchRule *v1alpha1.SearchRule) {

	// Create the new condition with the success status
	condition := globals.NewCondition(globals.ConditionTypeState, metav1.ConditionTrue,
		globals.ConditionReasonNoQueryFoundType, globals.ConditionReasonNoQueryFoundMessage)

	// Update the status of the SearchRule resource
	globals.UpdateCondition(&SearchRule.Status.Conditions, condition)
}

// UpdateConditionConnectionError updates the status of the SearchRule resource with a ConnectionError condition
func (r *SearchRuleReconciler) UpdateConditionConnectionError(SearchRule *v1alpha1.SearchRule) {

	// Create the new condition with the failure status
	condition := globals.NewCondition(globals.ConditionTypeState, metav1.ConditionTrue,
		globals.ConditionReasonConnectionErrorType, globals.ConditionReasonConnectionErrorMessage)

	// Update the status of the SearchRule resource
	globals.UpdateCondition(&SearchRule.Status.Conditions, condition)
}

// UpdateConditionEvaluateTemplateError updates the status of the SearchRule resource with a QueryError condition
func (r *SearchRuleReconciler) UpdateConditionQueryError(SearchRule *v1alpha1.SearchRule) {

	// Create the new condition with the failure status
	condition := globals.NewCondition(globals.ConditionTypeState, metav1.ConditionTrue,
		globals.ConditionReasonQueryErrorType, globals.ConditionReasonQueryErrorMessage)

	// Update the status of the SearchRule resource
	globals.UpdateCondition(&SearchRule.Status.Conditions, condition)
}

// UpdateConditionPrometheusRuleSynced reports a successful reconcile of the
// auto-generated PrometheusRule resource.
func (r *SearchRuleReconciler) UpdateConditionPrometheusRuleSynced(searchRule *v1alpha1.SearchRule) {
	condition := globals.NewCondition(globals.ConditionTypePrometheusRule, metav1.ConditionTrue,
		globals.ConditionReasonPrometheusRuleSyncedType, globals.ConditionReasonPrometheusRuleSyncedMessage)
	globals.UpdateCondition(&searchRule.Status.Conditions, condition)
}

// UpdateConditionPrometheusRuleUnsupported reports that the cluster does not
// have the monitoring.coreos.com/v1 PrometheusRule CRD installed.
func (r *SearchRuleReconciler) UpdateConditionPrometheusRuleUnsupported(searchRule *v1alpha1.SearchRule) {
	condition := globals.NewCondition(globals.ConditionTypePrometheusRule, metav1.ConditionFalse,
		globals.ConditionReasonPrometheusRuleUnsupportedType, globals.ConditionReasonPrometheusRuleUnsupportedMessage)
	globals.UpdateCondition(&searchRule.Status.Conditions, condition)
}

// UpdateConditionPrometheusRuleMetricsNotExposed warns the user that the
// PrometheusRule was created but the underlying metric is not being served by
// the operator (--rules-metrics-bind-address=0).
func (r *SearchRuleReconciler) UpdateConditionPrometheusRuleMetricsNotExposed(searchRule *v1alpha1.SearchRule) {
	condition := globals.NewCondition(globals.ConditionTypePrometheusRule, metav1.ConditionTrue,
		globals.ConditionReasonPrometheusRuleMetricsNotExposedType, globals.ConditionReasonPrometheusRuleMetricsNotExposedMessage)
	globals.UpdateCondition(&searchRule.Status.Conditions, condition)
}

// UpdateConditionPrometheusRuleError reports a failure reconciling the
// PrometheusRule. The detailed message comes from the underlying error so the
// user can see why the API call or rendering failed.
func (r *SearchRuleReconciler) UpdateConditionPrometheusRuleError(searchRule *v1alpha1.SearchRule, message string) {
	condition := globals.NewCondition(globals.ConditionTypePrometheusRule, metav1.ConditionFalse,
		globals.ConditionReasonPrometheusRuleErrorType, message)
	globals.UpdateCondition(&searchRule.Status.Conditions, condition)
}

// UpdateConditionPrometheusRuleMetricNameMismatch warns the user that
// `spec.prometheusRule.metricName` does not match any of the declared
// `spec.customMetrics[*].name`. The alert is generated against the fallback
// metric (the first customMetrics entry) so behaviour is deterministic, but
// the user almost certainly intended a different gauge.
func (r *SearchRuleReconciler) UpdateConditionPrometheusRuleMetricNameMismatch(searchRule *v1alpha1.SearchRule, message string) {
	condition := globals.NewCondition(globals.ConditionTypePrometheusRule, metav1.ConditionTrue,
		globals.ConditionReasonPrometheusRuleMetricNameMismatchType, message)
	globals.UpdateCondition(&searchRule.Status.Conditions, condition)
}

// UpdateConditionMissingOutput reports that the SearchRule defines neither
// actionRef nor prometheusRule and thus cannot produce any output.
func (r *SearchRuleReconciler) UpdateConditionMissingOutput(searchRule *v1alpha1.SearchRule) {
	condition := globals.NewCondition(globals.ConditionTypeResourceSynced, metav1.ConditionFalse,
		globals.ConditionReasonMissingOutputType, globals.ConditionReasonMissingOutputMessage)
	globals.UpdateCondition(&searchRule.Status.Conditions, condition)
}
