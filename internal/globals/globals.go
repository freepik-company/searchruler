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

package globals

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (

	// Constants for the sync conditions
	// Condition type for sync resource
	ConditionTypeResourceSynced = "ResourceSynced"

	// Success
	ConditionReasonTargetSynced        = "TargetSynced"
	ConditionReasonTargetSyncedMessage = "Target was successfully synced"

	// Kubernetes error type
	ConditionReasonKubernetesApiCallErrorType    = "KubernetesApiCallError"
	ConditionReasonKubernetesApiCallErrorMessage = "Call to Kubernetes API failed. More info in logs."

	// Constants for the state conditions
	// Condition type for state
	ConditionTypeState = "State"

	// State success type
	ConditionReasonStateSuccessType    = "Success"
	ConditionReasonStateSuccessMessage = "Success executing tasks"

	// Alert firing and resolved status messages
	ConditionReasonAlertFiring                 = "AlertFiring"
	ConditionReasonAlertFiringMessage          = "Alert is firing"
	ConditionReasonPendingAlertFiringMessage   = "Pending alert firing"
	ConditionReasonPendingAlertFiring          = "PendingAlertFiring"
	ConditionReasonPendingAlertResolvedMessage = "Pending alert resolved"
	ConditionReasonPendingAlertResolved        = "PendingAlertResolved"
	ConditionReasonStateNormalType             = "Normal"
	ConditionReasonStateNormalMessage          = "Rule is normal"

	// No credentials found
	ConditionReasonNoCredsFoundType    = "NoCredsFound"
	ConditionReasonNoCredsFoundMessage = "No credentials found in secret"

	// Connection error
	ConditionReasonConnectionErrorType    = "ConnectionError"
	ConditionReasonConnectionErrorMessage = "Connection error to the webhook target to send the alert"

	// Evaluate template error
	ConditionReasonEvaluateTemplateErrorType    = "EvaluateTemplateError"
	ConditionReasonEvaluateTemplateErrorMessage = "Error evaluating the template for the alert"

	// QueryConnector not found
	ConditionReasonQueryConnectorNotFoundType    = "QueryConnectorNotFound"
	ConditionReasonQueryConnectorNotFoundMessage = "QueryConnector not found"

	// No query found in the SearchRule
	ConditionReasonNoQueryFoundMessage = "No query found in the SearchRule"
	ConditionReasonNoQueryFoundType    = "NoQueryFound"

	// Query error
	ConditionReasonQueryErrorMessage = "Error executing the query"
	ConditionReasonQueryErrorType    = "QueryError"

	// No certificates found
	ConditionReasonNoCertsFoundType    = "NoCertsFound"
	ConditionReasonNoCertsFoundMessage = "No certificates found in secret"

	// PrometheusRule output condition
	ConditionTypePrometheusRule = "PrometheusRule"

	ConditionReasonPrometheusRuleSyncedType    = "Synced"
	ConditionReasonPrometheusRuleSyncedMessage = "PrometheusRule resource is in sync with the SearchRule"

	ConditionReasonPrometheusRuleUnsupportedType    = "Unsupported"
	ConditionReasonPrometheusRuleUnsupportedMessage = "monitoring.coreos.com/v1 PrometheusRule CRD was not installed in the cluster when the operator started; install the CRD and restart the operator to pick up the change"

	ConditionReasonPrometheusRuleMetricsNotExposedType    = "MetricsNotExposed"
	ConditionReasonPrometheusRuleMetricsNotExposedMessage = "PrometheusRule was created but the searchrule_value metric is not exposed (--rules-metrics-bind-address=0); alerts will not fire until the operator exposes its custom metrics"

	ConditionReasonPrometheusRuleErrorType    = "PrometheusRuleError"
	ConditionReasonPrometheusRuleErrorMessage = "Failed to reconcile the PrometheusRule resource"

	ConditionReasonPrometheusRuleMetricNameMismatchType = "MetricNameMismatch"

	// At least one of actionRef or prometheusRule must be defined
	ConditionReasonMissingOutputType    = "MissingOutput"
	ConditionReasonMissingOutputMessage = "SearchRule has neither actionRef nor prometheusRule defined; at least one is required"

	// CustomMetrics — bucket-aware Prometheus gauges
	ConditionTypeCustomMetrics = "CustomMetrics"

	ConditionReasonCustomMetricsSyncedType    = "Synced"
	ConditionReasonCustomMetricsSyncedMessage = "Custom metrics derived from the Elasticsearch aggregations are being exposed"

	ConditionReasonCustomMetricsNoAggregationsType    = "NoAggregations"
	ConditionReasonCustomMetricsNoAggregationsMessage = "spec.customMetrics is set but the last Elasticsearch response did not include any aggregations; nothing to expose"

	ConditionReasonCustomMetricsPathNotFoundType = "PathNotFound"

	ConditionReasonCustomMetricsNameReservedType = "NameReserved"

	ConditionReasonCustomMetricsTruncatedType = "Truncated"

	ConditionReasonCustomMetricsLabelMismatchType = "LabelMismatch"
)

var (
	Application = applicationT{
		Context: context.Background(),
	}
)

// NewCondition a set of default options for creating a Condition.
func NewCondition(condType string, status metav1.ConditionStatus, reason, message string) metav1.Condition {
	return metav1.Condition{
		Type:               condType,
		Status:             status,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

func getCondition(conditions *[]metav1.Condition, condType string) *metav1.Condition {
	for i, v := range *conditions {
		if v.Type == condType {
			return &(*conditions)[i]
		}
	}
	return nil
}

func UpdateCondition(conditions *[]metav1.Condition, condition metav1.Condition) {

	// Get the condition
	currentCondition := getCondition(conditions, condition.Type)

	if currentCondition == nil {
		// Create the condition when not existent
		*conditions = append(*conditions, condition)
	} else {
		// Update the condition when existent.
		currentCondition.Status = condition.Status
		currentCondition.Reason = condition.Reason
		currentCondition.Message = condition.Message
		currentCondition.LastTransitionTime = metav1.Now()
	}
}

// RemoveCondition drops any condition with the given type from the slice.
// Useful when a feature toggle flips off and we want to keep the status free
// of stale information.
func RemoveCondition(conditions *[]metav1.Condition, condType string) {
	if conditions == nil {
		return
	}
	out := (*conditions)[:0]
	for _, c := range *conditions {
		if c.Type != condType {
			out = append(out, c)
		}
	}
	*conditions = out
}
