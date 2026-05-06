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
	"strconv"
	"time"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"freepik.com/searchruler/api/v1alpha1"
	"freepik.com/searchruler/internal/globals"
)

const (
	// promRuleGroupName is the group name used inside the generated
	// PrometheusRule. It is fixed because each SearchRule materializes a
	// single rule, in a single group.
	promRuleGroupName = "searchruler"

	// promRuleManagedLabel marks every PrometheusRule rendered by this
	// operator so users can distinguish them from manually-managed ones.
	promRuleManagedLabel = "searchruler.freepik.com/managed-by"
	promRuleManagedValue = "searchruler"

	// promRuleSearchRuleLabel carries the SearchRule name on the alert
	// labels, mirroring the `rule` label exported by the searchrule_value
	// metric.
	promRuleSearchRuleLabel = "searchrule"
)

// reconcilePrometheusRule reconciles the PrometheusRule resource that mirrors
// a SearchRule's condition, so users with a Prometheus + Alertmanager stack
// can route alerts without configuring a separate actionRef.
//
// Behaviour:
//   - If spec.prometheusRule is nil or disabled, any previously-created
//     PrometheusRule is deleted (idempotently). No condition is set.
//   - If the cluster does not have the PrometheusRule CRD, the SearchRule is
//     marked with PrometheusRuleUnsupported and reconcile returns nil so the
//     normal sync loop continues.
//   - Otherwise, a PrometheusRule named after the SearchRule is created or
//     updated, owned by the SearchRule for automatic GC.
func (r *SearchRuleReconciler) reconcilePrometheusRule(ctx context.Context, rule *v1alpha1.SearchRule) error {
	logger := log.FromContext(ctx)

	desired := rule.Spec.PrometheusRule

	// Disabled or absent: ensure any previously-created PrometheusRule is gone
	// and clear any stale status condition. Otherwise a SearchRule that used
	// to report PrometheusRule.Synced would keep showing that state long
	// after the feature was turned off.
	if desired == nil || !desired.Enabled {
		globals.RemoveCondition(&rule.Status.Conditions, globals.ConditionTypePrometheusRule)
		if !r.PrometheusRuleSupported {
			return nil
		}
		return r.deletePrometheusRuleIfExists(ctx, rule)
	}

	// Requested but the CRD is not installed: surface a clear condition and
	// return without erroring out the entire reconcile loop.
	if !r.PrometheusRuleSupported {
		r.UpdateConditionPrometheusRuleUnsupported(rule)
		logger.Info("spec.prometheusRule is enabled but PrometheusRule CRD is not installed; skipping",
			"searchrule", rule.Name, "namespace", rule.Namespace)
		return nil
	}

	expr, err := buildPromQLExpr(rule)
	if err != nil {
		r.UpdateConditionPrometheusRuleError(rule, err.Error())
		return fmt.Errorf("failed to build PromQL expression: %w", err)
	}

	forDuration, err := parsePromDuration(rule.Spec.Condition.For)
	if err != nil {
		r.UpdateConditionPrometheusRuleError(rule, err.Error())
		return fmt.Errorf("failed to parse condition.for: %w", err)
	}

	// Refuse to adopt a PrometheusRule that already exists with a different
	// (or missing) owner. We never want to silently take ownership of a
	// resource a human created by hand and then later delete it.
	pr := &monitoringv1.PrometheusRule{}
	getErr := r.Get(ctx, types.NamespacedName{Namespace: rule.Namespace, Name: rule.Name}, pr)
	switch {
	case apierrors.IsNotFound(getErr):
		pr = &monitoringv1.PrometheusRule{ObjectMeta: metav1.ObjectMeta{
			Name:      rule.Name,
			Namespace: rule.Namespace,
		}}
	case getErr != nil:
		r.UpdateConditionPrometheusRuleError(rule, getErr.Error())
		return fmt.Errorf("failed to fetch PrometheusRule: %w", getErr)
	default:
		if !isOwnedByThisSearchRule(pr, rule) {
			msg := fmt.Sprintf("a PrometheusRule named %q already exists and is not managed by this SearchRule; refusing to overwrite it", pr.Name)
			r.UpdateConditionPrometheusRuleError(rule, msg)
			return fmt.Errorf("%s", msg)
		}
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, pr, func() error {
		if err := controllerutil.SetControllerReference(rule, pr, r.Scheme); err != nil {
			return err
		}
		pr.Labels = mergeLabels(pr.Labels, map[string]string{
			promRuleManagedLabel:           promRuleManagedValue,
			"app.kubernetes.io/managed-by": "searchruler",
		})
		pr.Spec = monitoringv1.PrometheusRuleSpec{
			Groups: []monitoringv1.RuleGroup{{
				Name:  promRuleGroupName,
				Rules: []monitoringv1.Rule{buildAlertingRule(rule, expr, forDuration)},
			}},
		}
		return nil
	})
	if err != nil {
		r.UpdateConditionPrometheusRuleError(rule, err.Error())
		return fmt.Errorf("failed to create or update PrometheusRule: %w", err)
	}
	logger.V(1).Info("PrometheusRule reconciled",
		"operation", op, "name", pr.Name, "namespace", pr.Namespace)

	if !r.MetricsExposed {
		r.UpdateConditionPrometheusRuleMetricsNotExposed(rule)
		return nil
	}
	r.UpdateConditionPrometheusRuleSynced(rule)
	return nil
}

// buildAlertingRule renders the single alerting rule embedded in the
// PrometheusRule for this SearchRule.
func buildAlertingRule(rule *v1alpha1.SearchRule, expr string, forDuration monitoringv1.Duration) monitoringv1.Rule {
	alertName := rule.Name
	if rule.Spec.PrometheusRule != nil && rule.Spec.PrometheusRule.AlertName != "" {
		alertName = rule.Spec.PrometheusRule.AlertName
	}

	// We expose the SearchRule's own namespace under `searchrule_namespace`
	// rather than `namespace` for the same reason as the underlying metric:
	// avoid the ServiceMonitor target-label collision that turns `namespace`
	// into `exported_namespace` on the Prometheus side.
	labels := map[string]string{
		promRuleSearchRuleLabel: rule.Name,
		"searchrule_namespace":  rule.Namespace,
	}
	if rule.Spec.PrometheusRule != nil {
		labels = mergeLabels(labels, rule.Spec.PrometheusRule.Labels)
	}

	annotations := map[string]string{
		"description": fmt.Sprintf(
			"SearchRule %s/%s condition (%s %s %s) is firing on metric searchrule_value.",
			rule.Namespace, rule.Name,
			rule.Spec.Condition.Operator, "threshold", rule.Spec.Condition.Threshold,
		),
	}
	if rule.Spec.PrometheusRule != nil {
		annotations = mergeLabels(annotations, rule.Spec.PrometheusRule.Annotations)
	}

	return monitoringv1.Rule{
		Alert:       alertName,
		Expr:        intstr.FromString(expr),
		For:         &forDuration,
		Labels:      labels,
		Annotations: annotations,
	}
}

// buildPromQLExpr translates a SearchRule's condition into a PromQL
// expression on the searchrule_value metric exposed by the operator.
// The selector includes both searchrule_namespace and rule labels because
// SearchRule names are unique only within a namespace, so two rules with the
// same name in different namespaces would otherwise share the same time
// series. The label is `searchrule_namespace` (not `namespace`) to avoid the
// ServiceMonitor target-label collision described in metrics.go.
//
// The threshold is parsed as a float and re-rendered before being inlined.
// We never embed the user-provided string directly: it would let a SearchRule
// author smuggle arbitrary PromQL into the alert expression
// (e.g. "100 or vector(0)"), bypassing the intended scalar comparison.
func buildPromQLExpr(rule *v1alpha1.SearchRule) (string, error) {
	op, err := promqlOperator(rule.Spec.Condition.Operator)
	if err != nil {
		return "", err
	}
	rawThreshold := rule.Spec.Condition.Threshold
	if rawThreshold == "" {
		return "", fmt.Errorf("condition.threshold is empty")
	}
	threshold, err := strconv.ParseFloat(rawThreshold, 64)
	if err != nil {
		return "", fmt.Errorf("condition.threshold %q is not a valid float: %w", rawThreshold, err)
	}
	// strconv.FormatFloat with -1 precision keeps the shortest representation
	// that round-trips, so "100" stays "100" and "0.5" stays "0.5".
	return fmt.Sprintf(`searchrule_value{searchrule_namespace=%q,rule=%q} %s %s`,
		rule.Namespace, rule.Name, op, strconv.FormatFloat(threshold, 'f', -1, 64)), nil
}

// promqlOperator maps a SearchRule condition operator to its PromQL syntax.
// The set is fixed and matches the operators understood by evaluateCondition.
func promqlOperator(op string) (string, error) {
	switch op {
	case conditionGreaterThan:
		return ">", nil
	case conditionGreaterThanOrEqual:
		return ">=", nil
	case conditionLessThan:
		return "<", nil
	case conditionLessThanOrEqual:
		return "<=", nil
	case conditionEqual:
		return "==", nil
	default:
		return "", fmt.Errorf("unsupported condition operator %q", op)
	}
}

// parsePromDuration validates that a duration string is parseable as Go time
// (the same format the rest of the operator already accepts) and returns it
// in the monitoringv1.Duration type expected by the prometheus-operator API.
// monitoringv1.Duration is a typed string, so we keep the original spelling
// (e.g. "5m") rather than re-formatting it.
func parsePromDuration(s string) (monitoringv1.Duration, error) {
	if _, err := time.ParseDuration(s); err != nil {
		return "", fmt.Errorf("invalid duration %q: %w", s, err)
	}
	return monitoringv1.Duration(s), nil
}

// deletePrometheusRuleIfExists removes the PrometheusRule that mirrors this
// SearchRule, ignoring NotFound errors. Only deletes resources owned by the
// given SearchRule so a hand-managed PrometheusRule that happens to share the
// name is never destroyed.
func (r *SearchRuleReconciler) deletePrometheusRuleIfExists(ctx context.Context, rule *v1alpha1.SearchRule) error {
	pr := &monitoringv1.PrometheusRule{}
	key := types.NamespacedName{Namespace: rule.Namespace, Name: rule.Name}
	if err := r.Get(ctx, key, pr); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to fetch existing PrometheusRule for cleanup: %w", err)
	}
	if !isOwnedByThisSearchRule(pr, rule) {
		// Not ours: leave it alone.
		return nil
	}
	if err := r.Delete(ctx, pr); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete PrometheusRule: %w", err)
	}
	return nil
}

// isOwnedByThisSearchRule returns true when the PrometheusRule has a
// controller OwnerReference pointing at the given SearchRule. This is the
// strict ownership check used to decide whether the operator may mutate or
// delete the resource.
func isOwnedByThisSearchRule(pr *monitoringv1.PrometheusRule, rule *v1alpha1.SearchRule) bool {
	for _, ref := range pr.OwnerReferences {
		if ref.Controller != nil && *ref.Controller && ref.UID == rule.UID {
			return true
		}
	}
	return false
}

// mergeLabels returns a new map containing all keys from base, with values
// from overlay overriding when both define the same key. Either argument may
// be nil.
func mergeLabels(base, overlay map[string]string) map[string]string {
	if len(base) == 0 && len(overlay) == 0 {
		return nil
	}
	out := make(map[string]string, len(base)+len(overlay))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range overlay {
		out[k] = v
	}
	return out
}

