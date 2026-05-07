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

package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tidwall/gjson"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"freepik.com/searchruler/api/v1alpha1"
	"freepik.com/searchruler/internal/controller/searchrule"
	"freepik.com/searchruler/internal/pools"
)

type RuleMetricT struct {
	Name   string
	Help   string
	Labels []string
}

const (
	// metricNamePrefix is prepended to every CustomMetric.Name when the
	// gauge is registered with Prometheus. The full metric name then is
	// `searchrule_<Name>`.
	metricNamePrefix = "searchrule_"

	// customMetricBucketLimit caps the number of buckets emitted per
	// SearchRule per refresh tick. Aggregations on user-provided fields
	// can blow up to tens of thousands of buckets and crash the operator
	// with OOM; the limit is defensive. The truncation is observable via
	// the `searchrule_custom_metrics_truncated_total` counter.
	customMetricBucketLimit = 1000
)

var (
	// Basic metrics definition (global). The namespace label is exported as
	// `searchrule_namespace` rather than the more obvious `namespace` to
	// avoid colliding with the target labels Prometheus injects when
	// scraping via a ServiceMonitor. Prometheus' default conflict policy
	// (honor_labels=false) would silently rename our `namespace` label to
	// `exported_namespace`, breaking the PromQL the operator generates.
	basicMetrics = map[string]RuleMetricT{
		"searchrule_value": {
			Name:   "searchrule_value",
			Help:   "Value of the search rule",
			Labels: []string{"searchrule_namespace", "rule"},
		},
		"searchrule_state": {
			Name:   "searchrule_state",
			Help:   "State of the search rule",
			Labels: []string{"searchrule_namespace", "rule", "state"},
		},
	}

	// Default rule metrics
	defaultRuleMetrics = map[string]*prometheus.GaugeVec{}

	// Prometheus registry
	prometheusRegistry = *prometheus.NewRegistry()

	// States for the rules
	ruleStates = []string{
		searchrule.RuleNormalState,
		searchrule.RuleFiringState,
		searchrule.RulePendingFiringState,
		searchrule.RulePendingResolvedState,
	}

	// customMetricsTruncated counts how many times a SearchRule emitted
	// more buckets than customMetricBucketLimit and we had to discard the
	// tail. Operators can wire this to an alert or a Grafana row.
	customMetricsTruncated = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "searchrule_custom_metrics_truncated_total",
		Help: "Times the operator truncated the bucket list for a custom metric because it exceeded the per-rule limit",
	}, []string{"searchrule_namespace", "rule", "metric"})

	// customMgr owns every dynamically-registered GaugeVec coming from
	// spec.customMetrics. It runs in the same process as the metrics http
	// handler so register/unregister and label-set tracking happen under
	// a single mutex.
	customMgr = newCustomMetricManager()
)

// Run starts the metrics server for the rules
func Run(ctx context.Context, rulesMetricsAddr string, rulesPool *pools.RulesStore,
	rulesMetricsRefreshSec int) (err error) {

	logger := log.FromContext(ctx)

	logger.Info(fmt.Sprintf("Starting rules metrics server on %s", rulesMetricsAddr))

	// Initialize the basic metrics
	err = initializeBasicMetrics()
	if err != nil {
		return fmt.Errorf("failed to initialize metrics: %w", err)
	}
	if err := prometheusRegistry.Register(customMetricsTruncated); err != nil {
		return fmt.Errorf("failed to register custom-metrics counter: %w", err)
	}

	// Metrics http handler
	http.Handle("/metrics", promhttp.HandlerFor(&prometheusRegistry, promhttp.HandlerOpts{}))

	// Start the metrics server
	server := &http.Server{Addr: rulesMetricsAddr}

	// Update the metrics every N seconds (controlled by --rules-metrics-refresh-rate).
	go func() {
		ticker := time.NewTicker(time.Duration(rulesMetricsRefreshSec) * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := updateMetrics(ctx, rulesPool); err != nil {
					logger.Info(fmt.Sprintf("Failed to update metrics: %v", err))
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Start the server
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Info(fmt.Sprintf("Metrics server error: %v", err))
	}

	return nil
}

// initializeBasicMetrics initializes the basic metrics for the rules
func initializeBasicMetrics() error {
	for name, item := range basicMetrics {
		defaultRuleMetrics[name] = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: item.Name,
				Help: item.Help,
			},
			item.Labels,
		)
		// Register the metric if it's not already registered
		if err := prometheusRegistry.Register(defaultRuleMetrics[name]); err != nil {
			return fmt.Errorf("failed to register metric: %w", err)
		}
	}

	return nil
}

// updateMetrics refreshes both the legacy gauges and every dynamically
// declared custom metric, and prunes label tuples that no longer match any
// SearchRule in the pool so /metrics never advertises stale series.
//
// Reads via Snapshot so the reconciler can safely overwrite `rule.Aggregations`
// in-place between its own Get/Set without producing torn interface{} reads
// here.
func updateMetrics(ctx context.Context, rulesPool *pools.RulesStore) error {
	logger := log.FromContext(ctx)
	rules := rulesPool.Snapshot()

	// Track which (rule, metric, labelTuple) combinations we publish in
	// this tick so we can diff against the previous tick and delete
	// orphaned series at the end.
	seenBasic := map[basicSeriesKey]struct{}{}
	seenCustom := map[customSeriesKey]struct{}{}

	for _, rule := range rules {
		ns := rule.SearchRule.Namespace
		ruleName := rule.SearchRule.Name

		// --- legacy gauges -----------------------------------------------
		if g, ok := defaultRuleMetrics["searchrule_value"]; ok {
			g.WithLabelValues(ns, ruleName).Set(rule.Value)
			seenBasic[basicSeriesKey{metric: "searchrule_value", labels: [3]string{ns, ruleName, ""}}] = struct{}{}
		}
		if g, ok := defaultRuleMetrics["searchrule_state"]; ok {
			for _, state := range ruleStates {
				v := 0.0
				if rule.State == state {
					v = 1
				}
				g.WithLabelValues(ns, ruleName, state).Set(v)
				seenBasic[basicSeriesKey{metric: "searchrule_state", labels: [3]string{ns, ruleName, state}}] = struct{}{}
			}
		}

		// --- custom metrics ----------------------------------------------
		for _, cm := range rule.SearchRule.Spec.CustomMetrics {
			fullName, err := validateMetricName(cm.Name)
			if err != nil {
				logger.Info(fmt.Sprintf("custom metric on %s/%s rejected: %v", ns, ruleName, err))
				continue
			}
			if rule.Aggregations == nil {
				continue
			}
			samples, truncated, err := extractCustomSamples(rule.Aggregations, cm)
			if err != nil {
				logger.Info(fmt.Sprintf("custom metric %q on %s/%s: %v", fullName, ns, ruleName, err))
				continue
			}
			if truncated {
				customMetricsTruncated.WithLabelValues(ns, ruleName, fullName).Inc()
			}
			labelNames := append([]string{"searchrule_namespace", "rule"}, customMetricLabelNames(cm)...)
			gauge, err := customMgr.gaugeFor(fullName, cm.Help, labelNames)
			if err != nil {
				logger.Info(fmt.Sprintf("custom metric %q on %s/%s: %v", fullName, ns, ruleName, err))
				continue
			}
			for _, s := range samples {
				values := append([]string{ns, ruleName}, s.labelValues...)
				gauge.WithLabelValues(values...).Set(s.value)
				seenCustom[customSeriesKey{metric: fullName, joined: strings.Join(values, "\x1f")}] = struct{}{}
			}
		}
	}

	// Drop orphaned basic series.
	customMgr.pruneBasic(defaultRuleMetrics, seenBasic)
	// Drop orphaned custom series.
	customMgr.pruneCustom(seenCustom)
	return nil
}

// validateMetricName is a thin wrapper around v1alpha1.ValidateCustomMetricName
// that prepends the operator's `searchrule_` prefix. Kept in this package so
// the runtime tracker only knows about the resolved (full) Prometheus name.
func validateMetricName(name string) (string, error) {
	if err := v1alpha1.ValidateCustomMetricName(name); err != nil {
		return "", err
	}
	return metricNamePrefix + name, nil
}

// customMetricLabelNames returns the user-declared label names for a custom
// metric in deterministic order so Prometheus' GaugeVec invariant (constant
// label set) holds across reconciles.
func customMetricLabelNames(cm v1alpha1.CustomMetric) []string {
	names := make([]string, 0, len(cm.Labels))
	for _, lbl := range cm.Labels {
		names = append(names, lbl.Name)
	}
	return names
}

// customSample is the rendered form of one bucket: ordered label values
// (matching customMetricLabelNames) plus the numeric gauge value.
type customSample struct {
	labelValues []string
	value       float64
}

// extractCustomSamples renders a list of customSample from a SearchRule's
// aggregations payload. Returns truncated=true when the underlying bucket
// array exceeded customMetricBucketLimit and the tail was discarded.
func extractCustomSamples(aggregations interface{}, cm v1alpha1.CustomMetric) ([]customSample, bool, error) {
	if aggregations == nil {
		return nil, false, nil
	}
	raw, err := json.Marshal(aggregations)
	if err != nil {
		return nil, false, fmt.Errorf("re-marshal aggregations: %w", err)
	}
	root := gjson.GetBytes(raw, cm.AggregationMap)
	if !root.Exists() {
		hint := ""
		if strings.HasPrefix(cm.AggregationMap, "aggregations.") {
			hint = " (note: aggregation_map is evaluated against the contents of the response's `aggregations` block; drop the `aggregations.` prefix)"
		}
		return nil, false, fmt.Errorf("aggregation_map path %q not found in aggregations%s",
			cm.AggregationMap, hint)
	}
	var buckets []gjson.Result
	if root.IsArray() {
		buckets = root.Array()
	} else {
		// Single-object fallback: emit one sample.
		buckets = []gjson.Result{root}
	}
	truncated := false
	if len(buckets) > customMetricBucketLimit {
		buckets = buckets[:customMetricBucketLimit]
		truncated = true
	}

	valuePath := cm.Value
	if valuePath == "" {
		// Default: a bare `terms` aggregation exposes its hit count as
		// `doc_count` per bucket. Pipeline aggregations (`max_bucket`,
		// `bucket_script`, etc.) yield `value` instead, but those need an
		// explicit Value path because the user always knows the structure.
		valuePath = "doc_count"
	}

	out := make([]customSample, 0, len(buckets))
	for _, b := range buckets {
		labelValues, ok := extractLabelValues(b, cm.Labels)
		if !ok {
			continue
		}
		raw := b.Get(valuePath)
		if !raw.Exists() {
			continue
		}
		// gjson.Float() coerces strings and ints; explicitly skip booleans
		// or arrays which would yield 0 silently.
		if raw.Type != gjson.Number && raw.Type != gjson.String {
			continue
		}
		out = append(out, customSample{labelValues: labelValues, value: raw.Float()})
	}
	return out, truncated, nil
}

// extractLabelValues materialises one row of label values for a bucket.
// Returns ok=false when any non-static label path is missing — the caller
// skips that bucket so the GaugeVec never sees a partial label set.
func extractLabelValues(bucket gjson.Result, labels []v1alpha1.MetricLabel) ([]string, bool) {
	out := make([]string, 0, len(labels))
	for _, lbl := range labels {
		if lbl.StaticValue {
			out = append(out, lbl.Value)
			continue
		}
		v := bucket.Get(lbl.Value)
		if !v.Exists() {
			return nil, false
		}
		out = append(out, v.String())
	}
	return out, true
}

// --- series tracker ---------------------------------------------------------

type basicSeriesKey struct {
	metric string
	labels [3]string // {ns, rule, state} (state empty for searchrule_value)
}

type customSeriesKey struct {
	metric string
	joined string // \x1f-separated label tuple including ns + rule
}

// staleGaugeTickThreshold is how many consecutive refreshes a custom gauge
// can stay empty (no SearchRule emitted samples for it) before the manager
// unregisters it from the Prometheus registry. With the default
// --rules-metrics-refresh-rate=10s this means ~5 minutes of inactivity.
const staleGaugeTickThreshold = 30

// customMetricManager owns dynamic GaugeVecs and the bookkeeping needed to
// release stale series. It is goroutine-safe: every public method takes the
// mutex.
type customMetricManager struct {
	mu sync.Mutex
	// gauges maps fullName -> registered GaugeVec.
	gauges map[string]*prometheus.GaugeVec
	// labels maps fullName -> the label names the gauge was registered
	// with. A subsequent rule using the same metric name MUST declare the
	// same labels in the same order.
	labels map[string][]string
	// idleTicks counts how many consecutive ticks we have not emitted any
	// sample for a registered metric. Reset to 0 on every emission and
	// once it exceeds staleGaugeTickThreshold the gauge is unregistered.
	idleTicks map[string]int
	// previous holds the series we emitted on the last tick keyed by both
	// metric name and label tuple, so we can DeleteLabelValues the diff.
	previousBasic  map[basicSeriesKey]struct{}
	previousCustom map[customSeriesKey]struct{}
}

func newCustomMetricManager() *customMetricManager {
	return &customMetricManager{
		gauges:         map[string]*prometheus.GaugeVec{},
		labels:         map[string][]string{},
		idleTicks:      map[string]int{},
		previousBasic:  map[basicSeriesKey]struct{}{},
		previousCustom: map[customSeriesKey]struct{}{},
	}
}

// gaugeFor returns the GaugeVec registered under fullName, creating it lazily
// the first time. Reusing the same name with a different label set is
// rejected — Prometheus client_golang would panic at .WithLabelValues, so we
// fail loudly upstream instead.
func (m *customMetricManager) gaugeFor(fullName, help string, labelNames []string) (*prometheus.GaugeVec, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, ok := m.gauges[fullName]; ok {
		if !equalSlices(m.labels[fullName], labelNames) {
			return nil, fmt.Errorf("metric %q already registered with labels %v; cannot re-register with %v",
				fullName, m.labels[fullName], labelNames)
		}
		// Note: do NOT reset idleTicks here. pruneCustom is the single
		// authority on idleness — it resets the counter only when the
		// gauge actually emitted samples this tick. Resetting here would
		// keep a gauge alive forever even if every bucket fails the
		// label/value extraction (e.g. a path that never exists), since
		// updateMetrics calls gaugeFor BEFORE iterating samples.
		return existing, nil
	}
	if help == "" {
		help = fmt.Sprintf("Custom metric %s exposed by SearchRuler from an Elasticsearch aggregation", fullName)
	}
	g := prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: fullName, Help: help}, labelNames)
	if err := prometheusRegistry.Register(g); err != nil {
		return nil, fmt.Errorf("register %q: %w", fullName, err)
	}
	m.gauges[fullName] = g
	cp := make([]string, len(labelNames))
	copy(cp, labelNames)
	m.labels[fullName] = cp
	return g, nil
}

// pruneBasic deletes label tuples on the operator's own gauges that we did
// not emit this tick (typically because their SearchRule was deleted).
func (m *customMetricManager) pruneBasic(metrics map[string]*prometheus.GaugeVec, seen map[basicSeriesKey]struct{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for prev := range m.previousBasic {
		if _, ok := seen[prev]; ok {
			continue
		}
		g, ok := metrics[prev.metric]
		if !ok {
			continue
		}
		switch prev.metric {
		case "searchrule_value":
			g.DeleteLabelValues(prev.labels[0], prev.labels[1])
		case "searchrule_state":
			g.DeleteLabelValues(prev.labels[0], prev.labels[1], prev.labels[2])
		}
	}
	m.previousBasic = seen
}

// pruneCustom deletes label tuples on dynamically-registered gauges that
// were absent from this tick. Then it tracks per-gauge idle ticks: a gauge
// that publishes zero samples for staleGaugeTickThreshold consecutive
// refreshes is fully unregistered so the /metrics endpoint stops advertising
// the empty gauge and the registry releases its memory.
func (m *customMetricManager) pruneCustom(seen map[customSeriesKey]struct{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for prev := range m.previousCustom {
		if _, ok := seen[prev]; ok {
			continue
		}
		gauge, ok := m.gauges[prev.metric]
		if !ok {
			continue
		}
		gauge.DeleteLabelValues(strings.Split(prev.joined, "\x1f")...)
	}
	m.previousCustom = seen

	// idleTicks tracking: which metrics emitted at least one series this tick?
	usedThisTick := map[string]struct{}{}
	for k := range seen {
		usedThisTick[k.metric] = struct{}{}
	}
	for name := range m.gauges {
		if _, used := usedThisTick[name]; used {
			m.idleTicks[name] = 0
			continue
		}
		m.idleTicks[name]++
		if m.idleTicks[name] >= staleGaugeTickThreshold {
			if g := m.gauges[name]; g != nil {
				prometheusRegistry.Unregister(g)
			}
			delete(m.gauges, name)
			delete(m.labels, name)
			delete(m.idleTicks, name)
		}
	}
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
