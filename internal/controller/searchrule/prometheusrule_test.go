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
	"testing"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	searchrulerv1alpha1 "freepik.com/searchruler/api/v1alpha1"
)

func newSearchRule(modify func(*searchrulerv1alpha1.SearchRule)) *searchrulerv1alpha1.SearchRule {
	rule := &searchrulerv1alpha1.SearchRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "default",
			UID:       "uid-demo",
		},
		Spec: searchrulerv1alpha1.SearchRuleSpec{
			CheckInterval: "30s",
			Condition: searchrulerv1alpha1.Condition{
				Operator:  conditionGreaterThan,
				Threshold: "100",
				For:       "1m",
			},
		},
	}
	if modify != nil {
		modify(rule)
	}
	return rule
}

func newScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(s); err != nil {
		t.Fatalf("clientgoscheme: %v", err)
	}
	if err := searchrulerv1alpha1.AddToScheme(s); err != nil {
		t.Fatalf("searchrulerv1alpha1: %v", err)
	}
	if err := monitoringv1.AddToScheme(s); err != nil {
		t.Fatalf("monitoringv1: %v", err)
	}
	return s
}

func TestPromqlOperator(t *testing.T) {
	t.Parallel()
	cases := []struct {
		op      string
		want    string
		wantErr bool
	}{
		{conditionGreaterThan, ">", false},
		{conditionGreaterThanOrEqual, ">=", false},
		{conditionLessThan, "<", false},
		{conditionLessThanOrEqual, "<=", false},
		{conditionEqual, "==", false},
		{"unknown", "", true},
		{"", "", true},
	}
	for _, c := range cases {
		c := c
		t.Run(c.op, func(t *testing.T) {
			t.Parallel()
			got, err := promqlOperator(c.op)
			if (err != nil) != c.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, c.wantErr)
			}
			if got != c.want {
				t.Fatalf("got=%q want=%q", got, c.want)
			}
		})
	}
}

func TestChooseAlertMetric(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name         string
		rule         *searchrulerv1alpha1.SearchRule
		want         string
		wantFallback bool
	}{
		{
			name: "no customMetrics falls back to legacy",
			rule: newSearchRule(nil),
			want: "searchrule_value",
		},
		{
			name: "single custom metric is selected",
			rule: newSearchRule(func(r *searchrulerv1alpha1.SearchRule) {
				r.Spec.CustomMetrics = []searchrulerv1alpha1.CustomMetric{{Name: "akamai_5xx_by_host"}}
			}),
			want: "searchrule_akamai_5xx_by_host",
		},
		{
			name: "first custom metric wins when no MetricName selector",
			rule: newSearchRule(func(r *searchrulerv1alpha1.SearchRule) {
				r.Spec.CustomMetrics = []searchrulerv1alpha1.CustomMetric{
					{Name: "primary"},
					{Name: "secondary"},
				}
				r.Spec.PrometheusRule = &searchrulerv1alpha1.PrometheusRuleSpec{Enabled: true}
			}),
			want: "searchrule_primary",
		},
		{
			name: "MetricName selector picks the matching metric",
			rule: newSearchRule(func(r *searchrulerv1alpha1.SearchRule) {
				r.Spec.CustomMetrics = []searchrulerv1alpha1.CustomMetric{
					{Name: "primary"},
					{Name: "secondary"},
				}
				r.Spec.PrometheusRule = &searchrulerv1alpha1.PrometheusRuleSpec{
					Enabled:    true,
					MetricName: "secondary",
				}
			}),
			want: "searchrule_secondary",
		},
		{
			name: "unknown MetricName falls back to first entry and surfaces a reason",
			rule: newSearchRule(func(r *searchrulerv1alpha1.SearchRule) {
				r.Spec.CustomMetrics = []searchrulerv1alpha1.CustomMetric{
					{Name: "primary"},
					{Name: "secondary"},
				}
				r.Spec.PrometheusRule = &searchrulerv1alpha1.PrometheusRuleSpec{
					Enabled:    true,
					MetricName: "ghost",
				}
			}),
			want:         "searchrule_primary",
			wantFallback: true,
		},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, fallback := chooseAlertMetric(c.rule)
			if got != c.want {
				t.Fatalf("got=%q want=%q", got, c.want)
			}
			if (fallback != "") != c.wantFallback {
				t.Fatalf("fallback=%q wantFallback=%v", fallback, c.wantFallback)
			}
		})
	}
}

func TestBuildPromQLExpr_UsesCustomMetric(t *testing.T) {
	t.Parallel()
	rule := newSearchRule(func(r *searchrulerv1alpha1.SearchRule) {
		r.Spec.CustomMetrics = []searchrulerv1alpha1.CustomMetric{{Name: "akamai_5xx_by_host"}}
	})
	got, err := buildPromQLExpr(rule)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	want := `searchrule_akamai_5xx_by_host{searchrule_namespace="default",rule="demo"} > 100`
	if got != want {
		t.Fatalf("got=%q\nwant=%q", got, want)
	}
}

func TestBuildPromQLExpr(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		condition searchrulerv1alpha1.Condition
		want      string
		wantErr   bool
	}{
		{
			name:      "greaterThan",
			condition: searchrulerv1alpha1.Condition{Operator: conditionGreaterThan, Threshold: "100", For: "1m"},
			want:      `searchrule_value{searchrule_namespace="default",rule="demo"} > 100`,
		},
		{
			name:      "lessThanOrEqual with decimal threshold",
			condition: searchrulerv1alpha1.Condition{Operator: conditionLessThanOrEqual, Threshold: "0.5", For: "30s"},
			want:      `searchrule_value{searchrule_namespace="default",rule="demo"} <= 0.5`,
		},
		{
			name:      "empty threshold",
			condition: searchrulerv1alpha1.Condition{Operator: conditionGreaterThan, Threshold: "", For: "1m"},
			wantErr:   true,
		},
		{
			name:      "unknown operator",
			condition: searchrulerv1alpha1.Condition{Operator: "neverHeardOf", Threshold: "10", For: "1m"},
			wantErr:   true,
		},
		{
			name:      "non-numeric threshold rejected",
			condition: searchrulerv1alpha1.Condition{Operator: conditionGreaterThan, Threshold: "not_a_number", For: "1m"},
			wantErr:   true,
		},
		{
			name:      "promql injection attempt rejected",
			condition: searchrulerv1alpha1.Condition{Operator: conditionGreaterThan, Threshold: "100 or vector(0)", For: "1m"},
			wantErr:   true,
		},
		{
			name:      "scientific notation normalized",
			condition: searchrulerv1alpha1.Condition{Operator: conditionGreaterThan, Threshold: "1e3", For: "1m"},
			want:      `searchrule_value{searchrule_namespace="default",rule="demo"} > 1000`,
		},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rule := newSearchRule(func(r *searchrulerv1alpha1.SearchRule) {
				r.Spec.Condition = c.condition
			})
			got, err := buildPromQLExpr(rule)
			if (err != nil) != c.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, c.wantErr)
			}
			if !c.wantErr && got != c.want {
				t.Fatalf("got=%q want=%q", got, c.want)
			}
		})
	}
}

func TestParsePromDuration(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in      string
		want    monitoringv1.Duration
		wantErr bool
	}{
		{"1m", monitoringv1.Duration("1m"), false},
		{"500ms", monitoringv1.Duration("500ms"), false},
		{"2h30m", monitoringv1.Duration("2h30m"), false},
		{"", "", true},
		{"abc", "", true},
		{"5", "", true}, // no unit
	}
	for _, c := range cases {
		c := c
		t.Run(c.in, func(t *testing.T) {
			t.Parallel()
			got, err := parsePromDuration(c.in)
			if (err != nil) != c.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, c.wantErr)
			}
			if !c.wantErr && got != c.want {
				t.Fatalf("got=%q want=%q", got, c.want)
			}
		})
	}
}

func TestMergeLabels(t *testing.T) {
	t.Parallel()
	t.Run("both nil returns nil", func(t *testing.T) {
		t.Parallel()
		got := mergeLabels(nil, nil)
		if got != nil {
			t.Fatalf("expected nil, got %v", got)
		}
	})
	t.Run("overlay overrides base", func(t *testing.T) {
		t.Parallel()
		got := mergeLabels(map[string]string{"a": "1", "b": "2"}, map[string]string{"b": "OVERRIDE", "c": "3"})
		want := map[string]string{"a": "1", "b": "OVERRIDE", "c": "3"}
		if len(got) != len(want) {
			t.Fatalf("size mismatch got=%v want=%v", got, want)
		}
		for k, v := range want {
			if got[k] != v {
				t.Fatalf("key %q: got=%q want=%q", k, got[k], v)
			}
		}
	})
}

func TestBuildAlertingRule(t *testing.T) {
	t.Parallel()
	rule := newSearchRule(func(r *searchrulerv1alpha1.SearchRule) {
		r.Spec.PrometheusRule = &searchrulerv1alpha1.PrometheusRuleSpec{
			Enabled:     true,
			AlertName:   "CustomAlert",
			Labels:      map[string]string{"severity": "warning"},
			Annotations: map[string]string{"summary": "test"},
		}
	})
	dur, err := parsePromDuration(rule.Spec.Condition.For)
	if err != nil {
		t.Fatalf("parsePromDuration: %v", err)
	}
	got := buildAlertingRule(rule, "expr", dur)
	if got.Alert != "CustomAlert" {
		t.Fatalf("Alert=%q want=CustomAlert", got.Alert)
	}
	if got.Labels["searchrule"] != rule.Name {
		t.Fatalf("missing automatic searchrule label, got=%v", got.Labels)
	}
	if got.Labels["severity"] != "warning" {
		t.Fatalf("user severity label missing, got=%v", got.Labels)
	}
	if got.Annotations["summary"] != "test" {
		t.Fatalf("user summary annotation missing, got=%v", got.Annotations)
	}
	if got.Annotations["description"] == "" {
		t.Fatalf("default description annotation missing")
	}
	if got.For == nil || *got.For != dur {
		t.Fatalf("For mismatch: got=%v want=%v", got.For, dur)
	}
}

func TestBuildAlertingRule_DefaultAlertName(t *testing.T) {
	t.Parallel()
	rule := newSearchRule(func(r *searchrulerv1alpha1.SearchRule) {
		r.Spec.PrometheusRule = &searchrulerv1alpha1.PrometheusRuleSpec{Enabled: true}
	})
	got := buildAlertingRule(rule, "expr", monitoringv1.Duration("1m"))
	// Default alert name equals the SearchRule name (no hyphens to swap).
	if got.Alert != rule.Name {
		t.Fatalf("Alert=%q want=%q (SearchRule name)", got.Alert, rule.Name)
	}
}

func TestBuildAlertingRule_DefaultAlertNameSanitizesHyphens(t *testing.T) {
	t.Parallel()
	rule := newSearchRule(func(r *searchrulerv1alpha1.SearchRule) {
		r.Name = "high-error-rate"
		r.Spec.PrometheusRule = &searchrulerv1alpha1.PrometheusRuleSpec{Enabled: true}
	})
	got := buildAlertingRule(rule, "expr", monitoringv1.Duration("1m"))
	want := "high_error_rate"
	if got.Alert != want {
		t.Fatalf("Alert=%q want=%q (sanitized)", got.Alert, want)
	}
}

func TestDefaultAlertName(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"plain":            "plain",
		"with-hyphens":     "with_hyphens",
		"multi-word-name":  "multi_word_name",
		"already_safe":     "already_safe",
		"123-leading-num":  "123_leading_num", // K8s allows but Prometheus rejects leading digit; user must set alertName for that edge
	}
	for in, want := range cases {
		in, want := in, want
		t.Run(in, func(t *testing.T) {
			t.Parallel()
			if got := defaultAlertName(in); got != want {
				t.Fatalf("got=%q want=%q", got, want)
			}
		})
	}
}

func TestReconcilePrometheusRule_Disabled_NoOp(t *testing.T) {
	t.Parallel()
	scheme := newScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	r := &SearchRuleReconciler{Client: c, Scheme: scheme, PrometheusRuleSupported: true, MetricsExposed: true}

	rule := newSearchRule(nil) // PrometheusRule is nil
	if err := r.reconcilePrometheusRule(context.Background(), rule, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pr := &monitoringv1.PrometheusRule{}
	err := c.Get(context.Background(), types.NamespacedName{Namespace: rule.Namespace, Name: rule.Name}, pr)
	if !apierrors.IsNotFound(err) {
		t.Fatalf("expected NotFound, got err=%v", err)
	}
}

func TestReconcilePrometheusRule_Unsupported_SetsCondition(t *testing.T) {
	t.Parallel()
	scheme := newScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	r := &SearchRuleReconciler{Client: c, Scheme: scheme, PrometheusRuleSupported: false, MetricsExposed: true}

	rule := newSearchRule(func(sr *searchrulerv1alpha1.SearchRule) {
		sr.Spec.PrometheusRule = &searchrulerv1alpha1.PrometheusRuleSpec{Enabled: true}
	})
	if err := r.reconcilePrometheusRule(context.Background(), rule, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// No PrometheusRule should have been created
	pr := &monitoringv1.PrometheusRule{}
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: rule.Namespace, Name: rule.Name}, pr); !apierrors.IsNotFound(err) {
		t.Fatalf("expected NotFound, got err=%v", err)
	}

	if !hasCondition(rule.Status.Conditions, "PrometheusRule", "Unsupported") {
		t.Fatalf("expected PrometheusRule/Unsupported condition, got=%v", rule.Status.Conditions)
	}
}

func TestReconcilePrometheusRule_Enabled_CreatesResourceWithOwnerRef(t *testing.T) {
	t.Parallel()
	scheme := newScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	r := &SearchRuleReconciler{Client: c, Scheme: scheme, PrometheusRuleSupported: true, MetricsExposed: true}

	rule := newSearchRule(func(sr *searchrulerv1alpha1.SearchRule) {
		sr.Spec.PrometheusRule = &searchrulerv1alpha1.PrometheusRuleSpec{
			Enabled:     true,
			AlertName:   "MyAlert",
			Labels:      map[string]string{"severity": "critical"},
			Annotations: map[string]string{"runbook_url": "https://runbooks/example"},
		}
	})

	if err := r.reconcilePrometheusRule(context.Background(), rule, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pr := &monitoringv1.PrometheusRule{}
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: rule.Namespace, Name: rule.Name}, pr); err != nil {
		t.Fatalf("expected PrometheusRule to exist: %v", err)
	}

	if len(pr.OwnerReferences) != 1 {
		t.Fatalf("expected 1 owner reference, got=%d", len(pr.OwnerReferences))
	}
	if pr.OwnerReferences[0].UID != rule.UID {
		t.Fatalf("owner UID mismatch: got=%q want=%q", pr.OwnerReferences[0].UID, rule.UID)
	}
	if len(pr.Spec.Groups) != 1 || len(pr.Spec.Groups[0].Rules) != 1 {
		t.Fatalf("expected exactly one group with one rule, got=%v", pr.Spec.Groups)
	}
	r0 := pr.Spec.Groups[0].Rules[0]
	if r0.Alert != "MyAlert" {
		t.Fatalf("alert name mismatch: got=%q", r0.Alert)
	}
	wantExpr := `searchrule_value{searchrule_namespace="default",rule="demo"} > 100`
	if r0.Expr.String() != wantExpr {
		t.Fatalf("expr mismatch: got=%q want=%q", r0.Expr.String(), wantExpr)
	}
	if !hasCondition(rule.Status.Conditions, "PrometheusRule", "Synced") {
		t.Fatalf("expected PrometheusRule/Synced condition, got=%v", rule.Status.Conditions)
	}
}

func TestReconcilePrometheusRule_MetricsNotExposed_SetsWarningCondition(t *testing.T) {
	t.Parallel()
	scheme := newScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	r := &SearchRuleReconciler{Client: c, Scheme: scheme, PrometheusRuleSupported: true, MetricsExposed: false}

	rule := newSearchRule(func(sr *searchrulerv1alpha1.SearchRule) {
		sr.Spec.PrometheusRule = &searchrulerv1alpha1.PrometheusRuleSpec{Enabled: true}
	})
	if err := r.reconcilePrometheusRule(context.Background(), rule, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// PrometheusRule should still be created (not blocked).
	pr := &monitoringv1.PrometheusRule{}
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: rule.Namespace, Name: rule.Name}, pr); err != nil {
		t.Fatalf("expected PrometheusRule to be created even when metrics are not exposed: %v", err)
	}

	if !hasCondition(rule.Status.Conditions, "PrometheusRule", "MetricsNotExposed") {
		t.Fatalf("expected PrometheusRule/MetricsNotExposed condition, got=%v", rule.Status.Conditions)
	}
}

func TestReconcilePrometheusRule_DisabledAfterEnabled_DeletesResource(t *testing.T) {
	t.Parallel()
	scheme := newScheme(t)

	rule := newSearchRule(func(sr *searchrulerv1alpha1.SearchRule) {
		sr.Spec.PrometheusRule = &searchrulerv1alpha1.PrometheusRuleSpec{Enabled: true}
	})
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	r := &SearchRuleReconciler{Client: c, Scheme: scheme, PrometheusRuleSupported: true, MetricsExposed: true}

	if err := r.reconcilePrometheusRule(context.Background(), rule, false); err != nil {
		t.Fatalf("first reconcile: %v", err)
	}
	if !hasCondition(rule.Status.Conditions, "PrometheusRule", "Synced") {
		t.Fatalf("expected Synced condition after enable, got=%v", rule.Status.Conditions)
	}

	// User flips enabled to false.
	rule.Spec.PrometheusRule.Enabled = false
	if err := r.reconcilePrometheusRule(context.Background(), rule, false); err != nil {
		t.Fatalf("second reconcile: %v", err)
	}

	pr := &monitoringv1.PrometheusRule{}
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: rule.Namespace, Name: rule.Name}, pr); !apierrors.IsNotFound(err) {
		t.Fatalf("expected PrometheusRule to be deleted, got err=%v", err)
	}

	// And the previously-set status condition must be gone, otherwise the
	// SearchRule keeps reporting Synced/MetricsNotExposed forever.
	for _, c := range rule.Status.Conditions {
		if c.Type == "PrometheusRule" {
			t.Fatalf("expected PrometheusRule condition to be removed, still got=%v", c)
		}
	}
}

func TestReconcilePrometheusRule_RefusesToAdoptUnmanagedResource(t *testing.T) {
	t.Parallel()
	scheme := newScheme(t)

	// A PrometheusRule named "demo" already exists, owned by some other
	// controller (or by nothing at all). The operator must NOT take it over.
	pre := &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "default",
			Labels:    map[string]string{"managed-by": "human"},
		},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pre).Build()
	r := &SearchRuleReconciler{Client: c, Scheme: scheme, PrometheusRuleSupported: true, MetricsExposed: true}

	rule := newSearchRule(func(sr *searchrulerv1alpha1.SearchRule) {
		sr.Spec.PrometheusRule = &searchrulerv1alpha1.PrometheusRuleSpec{Enabled: true}
	})
	err := r.reconcilePrometheusRule(context.Background(), rule, false)
	if err == nil {
		t.Fatalf("expected error when PrometheusRule is not owned by the SearchRule")
	}

	// Original resource untouched: same labels, no controller owner.
	got := &monitoringv1.PrometheusRule{}
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: "default", Name: "demo"}, got); err != nil {
		t.Fatalf("get pre-existing PR: %v", err)
	}
	if got.Labels["managed-by"] != "human" {
		t.Fatalf("operator overwrote a foreign PrometheusRule: labels=%v", got.Labels)
	}
	if len(got.Spec.Groups) != 0 {
		t.Fatalf("operator overwrote a foreign PrometheusRule spec: groups=%v", got.Spec.Groups)
	}
	if !hasCondition(rule.Status.Conditions, "PrometheusRule", "PrometheusRuleError") {
		t.Fatalf("expected PrometheusRuleError condition, got=%v", rule.Status.Conditions)
	}
}

func TestDeletePrometheusRuleIfExists_SkipsUnmanagedResource(t *testing.T) {
	t.Parallel()
	scheme := newScheme(t)
	pre := &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pre).Build()
	r := &SearchRuleReconciler{Client: c, Scheme: scheme, PrometheusRuleSupported: true, MetricsExposed: true}

	rule := newSearchRule(nil) // no PrometheusRule spec — would normally trigger cleanup
	if err := r.reconcilePrometheusRule(context.Background(), rule, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := &monitoringv1.PrometheusRule{}
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: "default", Name: "demo"}, got); err != nil {
		t.Fatalf("expected the foreign PrometheusRule to remain, got err=%v", err)
	}
}

func hasCondition(conds []metav1.Condition, condType, reason string) bool {
	for _, c := range conds {
		if c.Type == condType && c.Reason == reason {
			return true
		}
	}
	return false
}
