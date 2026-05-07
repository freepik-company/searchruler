/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package metrics

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"

	"freepik.com/searchruler/api/v1alpha1"
)

func TestValidateMetricName(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"akamai_5xx_by_host", "searchrule_akamai_5xx_by_host", false},
		{"a", "searchrule_a", false},
		{"_underscore", "searchrule__underscore", false},
		{"value", "", true},                  // resolves to reserved searchrule_value
		{"state", "", true},                  // resolves to reserved searchrule_state
		{"", "", true},                       // empty
		{"with-hyphen", "", true},            // hyphen banned
		{"123leading", "", true},             // leading digit banned
		{"with:colon", "", true},             // colon reserved
		{"with space", "", true},             // whitespace banned
	}
	for _, c := range cases {
		c := c
		t.Run(c.in, func(t *testing.T) {
			t.Parallel()
			got, err := validateMetricName(c.in)
			if (err != nil) != c.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, c.wantErr)
			}
			if !c.wantErr && got != c.want {
				t.Fatalf("got=%q want=%q", got, c.want)
			}
		})
	}
}

// makeAggregations turns a JSON literal into the same shape the operator
// stores in pools.Rule.Aggregations (the gjson Result.Value() output is a
// generic map[string]interface{}).
func makeAggregations(t *testing.T, raw string) interface{} {
	t.Helper()
	var v interface{}
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		t.Fatalf("test fixture parse: %v", err)
	}
	return v
}

func TestExtractCustomSamples(t *testing.T) {
	t.Parallel()
	t.Run("nominal: terms aggregation with bucket_script value", func(t *testing.T) {
		t.Parallel()
		agg := makeAggregations(t, `{
			"by_domain": {
				"buckets": [
					{"key": "cp.freepik.com", "doc_count": 767, "errors": {"doc_count": 172}, "error_percentage": {"value": 22.43}},
					{"key": "www.freepik.com", "doc_count": 1000, "errors": {"doc_count": 4}, "error_percentage": {"value": 0.4}}
				]
			}
		}`)
		cm := v1alpha1.CustomMetric{
			Name:           "akamai_5xx_by_host",
			AggregationMap: "by_domain.buckets",
			Labels:         []v1alpha1.MetricLabel{{Name: "host", Value: "key"}},
			Value:          "error_percentage.value",
		}
		got, truncated, err := extractCustomSamples(agg, cm)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if truncated {
			t.Fatalf("did not expect truncation")
		}
		if len(got) != 2 {
			t.Fatalf("got %d samples, want 2", len(got))
		}
		if got[0].labelValues[0] != "cp.freepik.com" || got[0].value != 22.43 {
			t.Fatalf("first sample mismatch: %+v", got[0])
		}
		if got[1].labelValues[0] != "www.freepik.com" || got[1].value != 0.4 {
			t.Fatalf("second sample mismatch: %+v", got[1])
		}
	})

	t.Run("default value path defaults to _count", func(t *testing.T) {
		t.Parallel()
		agg := makeAggregations(t, `{"by_pod":{"buckets":[{"key":"foo","doc_count":42}]}}`)
		cm := v1alpha1.CustomMetric{
			Name:           "errors_by_pod",
			AggregationMap: "by_pod.buckets",
			Labels:         []v1alpha1.MetricLabel{{Name: "pod", Value: "key"}},
		}
		got, _, err := extractCustomSamples(agg, cm)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if len(got) != 1 || got[0].value != 42 {
			t.Fatalf("expected single sample with value=42, got %+v", got)
		}
	})

	t.Run("static label is emitted verbatim", func(t *testing.T) {
		t.Parallel()
		agg := makeAggregations(t, `{"by_pod":{"buckets":[{"key":"foo","doc_count":1},{"key":"bar","doc_count":2}]}}`)
		cm := v1alpha1.CustomMetric{
			Name:           "errors_by_pod",
			AggregationMap: "by_pod.buckets",
			Labels: []v1alpha1.MetricLabel{
				{Name: "pod", Value: "key"},
				{Name: "tier", Value: "warning", StaticValue: true},
			},
		}
		got, _, _ := extractCustomSamples(agg, cm)
		for _, s := range got {
			if s.labelValues[1] != "warning" {
				t.Fatalf("static label not propagated: %+v", s)
			}
		}
	})

	t.Run("missing label path skips the bucket", func(t *testing.T) {
		t.Parallel()
		agg := makeAggregations(t, `{"by":{"buckets":[{"key":"a","doc_count":1},{"doc_count":2}]}}`)
		cm := v1alpha1.CustomMetric{
			Name:           "x",
			AggregationMap: "by.buckets",
			Labels:         []v1alpha1.MetricLabel{{Name: "k", Value: "key"}},
		}
		got, _, _ := extractCustomSamples(agg, cm)
		if len(got) != 1 {
			t.Fatalf("expected 1 sample after skipping malformed bucket, got %d", len(got))
		}
	})

	t.Run("non-numeric value path skips the bucket", func(t *testing.T) {
		t.Parallel()
		agg := makeAggregations(t, `{"by":{"buckets":[{"key":"a","val":[1,2,3]}]}}`)
		cm := v1alpha1.CustomMetric{
			Name:           "x",
			AggregationMap: "by.buckets",
			Labels:         []v1alpha1.MetricLabel{{Name: "k", Value: "key"}},
			Value:          "val",
		}
		got, _, _ := extractCustomSamples(agg, cm)
		if len(got) != 0 {
			t.Fatalf("expected 0 samples (value is array), got %d", len(got))
		}
	})

	t.Run("aggregation_map missing returns error", func(t *testing.T) {
		t.Parallel()
		agg := makeAggregations(t, `{"by_other":{"buckets":[]}}`)
		cm := v1alpha1.CustomMetric{
			Name:           "x",
			AggregationMap: "missing.path.buckets",
			Labels:         []v1alpha1.MetricLabel{{Name: "k", Value: "key"}},
		}
		_, _, err := extractCustomSamples(agg, cm)
		if err == nil {
			t.Fatalf("expected error for missing path")
		}
	})

	t.Run("nil aggregations is a noop", func(t *testing.T) {
		t.Parallel()
		got, trunc, err := extractCustomSamples(nil, v1alpha1.CustomMetric{Name: "x", AggregationMap: "any"})
		if err != nil || trunc || got != nil {
			t.Fatalf("unexpected: got=%v trunc=%v err=%v", got, trunc, err)
		}
	})

	t.Run("object instead of array yields a single sample", func(t *testing.T) {
		t.Parallel()
		agg := makeAggregations(t, `{"global":{"key":"only","value":7}}`)
		cm := v1alpha1.CustomMetric{
			Name:           "x",
			AggregationMap: "global",
			Labels:         []v1alpha1.MetricLabel{{Name: "k", Value: "key"}},
			Value:          "value",
		}
		got, _, _ := extractCustomSamples(agg, cm)
		if len(got) != 1 || got[0].labelValues[0] != "only" || got[0].value != 7 {
			t.Fatalf("unexpected: %+v", got)
		}
	})

	t.Run("over the bucket limit truncates and reports", func(t *testing.T) {
		t.Parallel()
		var b strings.Builder
		b.WriteString(`{"big":{"buckets":[`)
		for i := 0; i < customMetricBucketLimit+5; i++ {
			if i > 0 {
				b.WriteString(",")
			}
			b.WriteString(`{"key":"k`)
			b.WriteString(strings.Repeat("x", 1))
			b.WriteString(`","doc_count":1}`)
		}
		b.WriteString(`]}}`)
		agg := makeAggregations(t, b.String())
		cm := v1alpha1.CustomMetric{
			Name:           "big",
			AggregationMap: "big.buckets",
			Labels:         []v1alpha1.MetricLabel{{Name: "k", Value: "key"}},
		}
		got, trunc, err := extractCustomSamples(agg, cm)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if !trunc {
			t.Fatalf("expected truncation flag")
		}
		if len(got) != customMetricBucketLimit {
			t.Fatalf("expected %d samples, got %d", customMetricBucketLimit, len(got))
		}
	})
}

func TestValidateCustomMetric_Labels(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		cm      v1alpha1.CustomMetric
		wantErr bool
	}{
		{
			name: "valid name + labels",
			cm: v1alpha1.CustomMetric{
				Name: "ok",
				Labels: []v1alpha1.MetricLabel{
					{Name: "host", Value: "key"},
					{Name: "tier_warning", Value: "warn"},
				},
			},
		},
		{
			name: "label name with hyphen rejected",
			cm: v1alpha1.CustomMetric{
				Name:   "ok",
				Labels: []v1alpha1.MetricLabel{{Name: "with-hyphen", Value: "key"}},
			},
			wantErr: true,
		},
		{
			name: "label name colliding with reserved is rejected",
			cm: v1alpha1.CustomMetric{
				Name:   "ok",
				Labels: []v1alpha1.MetricLabel{{Name: "rule", Value: "key"}},
			},
			wantErr: true,
		},
		{
			name: "label name colliding with searchrule_namespace is rejected",
			cm: v1alpha1.CustomMetric{
				Name:   "ok",
				Labels: []v1alpha1.MetricLabel{{Name: "searchrule_namespace", Value: "x"}},
			},
			wantErr: true,
		},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			err := c.cm.Validate()
			if (err != nil) != c.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, c.wantErr)
			}
		})
	}
}

func TestCustomMetricManager_GaugeFor_RejectsConflictingLabels(t *testing.T) {
	t.Parallel()
	// Use a fresh manager + private registry so we don't touch the global one.
	m := newCustomMetricManager()
	reg := prometheus.NewRegistry()
	// Register a gauge with our manager pretending the global is reg.
	g := prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "searchrule_test_a"}, []string{"searchrule_namespace", "rule", "host"})
	if err := reg.Register(g); err != nil {
		t.Fatalf("register: %v", err)
	}
	m.gauges["searchrule_test_a"] = g
	m.labels["searchrule_test_a"] = []string{"searchrule_namespace", "rule", "host"}

	if _, err := m.gaugeFor("searchrule_test_a", "", []string{"searchrule_namespace", "rule", "host"}); err != nil {
		t.Fatalf("same labels should be accepted, got err: %v", err)
	}
	if _, err := m.gaugeFor("searchrule_test_a", "", []string{"searchrule_namespace", "rule", "pod"}); err == nil {
		t.Fatalf("different labels should be rejected")
	}
}

func TestCustomMetricManager_PruneCustom_GarbageCollectsStale(t *testing.T) {
	t.Parallel()
	m := newCustomMetricManager()
	g := prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "searchrule_test_gc"}, []string{"searchrule_namespace", "rule", "host"})
	// Note: we register on the package-level prometheusRegistry because the
	// manager calls prometheusRegistry.Unregister directly.
	if err := prometheusRegistry.Register(g); err != nil {
		t.Fatalf("register: %v", err)
	}
	defer prometheusRegistry.Unregister(g)
	m.gauges["searchrule_test_gc"] = g
	m.labels["searchrule_test_gc"] = []string{"searchrule_namespace", "rule", "host"}

	// First tick: emit one series so the manager records previousCustom for it.
	g.WithLabelValues("ns", "rule", "host1").Set(1)
	seenFirst := map[customSeriesKey]struct{}{
		{metric: "searchrule_test_gc", joined: "ns\x1frule\x1fhost1"}: {},
	}
	m.pruneCustom(seenFirst)
	if _, ok := m.gauges["searchrule_test_gc"]; !ok {
		t.Fatalf("gauge should still be registered after first emission")
	}

	// Subsequent empty ticks: simulate staleGaugeTickThreshold ticks with
	// no samples. After the threshold the gauge must be unregistered.
	for i := 0; i < staleGaugeTickThreshold; i++ {
		m.pruneCustom(map[customSeriesKey]struct{}{})
	}
	if _, ok := m.gauges["searchrule_test_gc"]; ok {
		t.Fatalf("gauge should have been GC'd after %d empty ticks", staleGaugeTickThreshold)
	}
}

func TestCustomMetricManager_PruneCustom_ResetsIdleOnReuse(t *testing.T) {
	t.Parallel()
	m := newCustomMetricManager()
	g := prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "searchrule_test_reuse"}, []string{"searchrule_namespace", "rule", "host"})
	if err := prometheusRegistry.Register(g); err != nil {
		t.Fatalf("register: %v", err)
	}
	defer prometheusRegistry.Unregister(g)
	m.gauges["searchrule_test_reuse"] = g
	m.labels["searchrule_test_reuse"] = []string{"searchrule_namespace", "rule", "host"}

	// Several empty ticks short of the GC threshold.
	for i := 0; i < staleGaugeTickThreshold-1; i++ {
		m.pruneCustom(map[customSeriesKey]struct{}{})
	}
	// One tick with samples must reset the counter.
	m.pruneCustom(map[customSeriesKey]struct{}{
		{metric: "searchrule_test_reuse", joined: "ns\x1frule\x1fh"}: {},
	})
	// Another full streak of empty ticks; the gauge must survive because the
	// reset happened mid-streak.
	for i := 0; i < staleGaugeTickThreshold-1; i++ {
		m.pruneCustom(map[customSeriesKey]struct{}{})
	}
	if _, ok := m.gauges["searchrule_test_reuse"]; !ok {
		t.Fatalf("gauge should still be registered; idle counter reset failed")
	}
}
