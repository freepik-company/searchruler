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
	"reflect"
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tidwall/gjson"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"prosimcorp.com/SearchRuler/api/v1alpha1"
	"prosimcorp.com/SearchRuler/internal/controller/searchrule"
	"prosimcorp.com/SearchRuler/internal/pools"
)

type RuleMetricT struct {
	Name   string
	Help   string
	Labels []string
}

var (
	// Basic metrics definition (global)
	basicMetrics = map[string]RuleMetricT{
		"searchrule_value": {
			Name:   "searchrule_value",
			Help:   "Value of the search rule",
			Labels: []string{"rule"},
		},
		"searchrule_state": {
			Name:   "searchrule_state",
			Help:   "State of the search rule",
			Labels: []string{"rule", "state"},
		},
	}

	// Old rule metric to check if the metric has changed in each iteration
	oldRuleMetrics = map[string]*RuleMetricT{}

	// Default rule metrics
	defaultRuleMetrics = map[string]*prometheus.GaugeVec{}
	customRuleMetrics  = map[string]*prometheus.GaugeVec{}

	// Prometheus registry
	prometheusRegistry = *prometheus.NewRegistry()

	// States for the rules
	ruleStates = []string{
		searchrule.RuleNormalState,
		searchrule.RuleFiringState,
		searchrule.RulePendingFiringState,
		searchrule.RulePendingResolvedState,
	}
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

	// Metrics http handler
	http.Handle("/metrics", promhttp.HandlerFor(&prometheusRegistry, promhttp.HandlerOpts{}))

	// Start the metrics server
	server := &http.Server{Addr: rulesMetricsAddr}

	// Update the metrics every 15 seconds
	go func() {
		ticker := time.NewTicker(time.Duration(rulesMetricsRefreshSec) * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				err := updateMetrics(rulesPool)
				if err != nil {
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

// updateMetrics updates the metrics for the rules
func updateMetrics(rulesPool *pools.RulesStore) (err error) {
	// Get all the rules from the pool
	rules := rulesPool.GetAll()

	// Register custom metrics if they exist for each rule in the pool
	for _, rule := range rules {
		// If the rule has custom metrics, register them
		if rule.SearchRule.Spec.CustomMetrics != nil {
			// For each custom metric in the rule
			for _, customMetric := range rule.SearchRule.Spec.CustomMetrics {

				// Collect label keys
				var labelKeys []string
				for _, label := range customMetric.Labels {
					labelKeys = append(labelKeys, label.Name)
				}

				// Create the metric with the labels
				if _, exists := customRuleMetrics[customMetric.Name]; !exists {
					customRuleMetrics[customMetric.Name] = prometheus.NewGaugeVec(
						prometheus.GaugeOpts{
							Name: customMetric.Name,
							Help: customMetric.Help,
						},
						labelKeys,
					)
				}

				// If the metric is registered in the old metrics, check if it has changed
				if oldRuleMetrics[customMetric.Name] != nil {
					if oldRuleMetrics[customMetric.Name].Help != customMetric.Help ||
						!reflect.DeepEqual(oldRuleMetrics[customMetric.Name].Labels, labelKeys) {

						// Set old metric with the new values
						oldRuleMetrics[customMetric.Name].Name = customMetric.Name
						oldRuleMetrics[customMetric.Name].Help = customMetric.Help
						oldRuleMetrics[customMetric.Name].Labels = labelKeys

						// Create a new registry for prometheus. This is needed so we can not unregister
						// a metric with the same name but different labels. It's mandatory to remove the old
						// registry and create a new one.
						prometheusRegistry = *prometheus.NewRegistry()

						// Force the garbage collector to free memory of the old registry
						runtime.GC()

						// Initialize the basic metrics to the new registry
						err = initializeBasicMetrics()
						if err != nil {
							return fmt.Errorf("failed to initialize basic metrics: %w", err)
						}
					}
				} else {
					// If the metric is not in memory, add it to the old metrics
					oldRuleMetrics[customMetric.Name] = &RuleMetricT{
						Name:   customMetric.Name,
						Help:   customMetric.Help,
						Labels: labelKeys,
					}
				}

				// Register the metric if it's not already registered
				if err := prometheusRegistry.Register(customRuleMetrics[customMetric.Name]); err != nil {
					if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
						return fmt.Errorf("failed to register metric: %w", err)
					}
				}

				// Parse aggregation map returned from elasticsearch as JSON
				ruleJSON, err := json.Marshal(rule.Aggregations)
				if err != nil {
					return fmt.Errorf("failed to marshal rule: %w", err)
				}

				// Get the field aggregationMap from the aggregation JSON
				aggregationMap := gjson.Get(string(ruleJSON), customMetric.AggregationMap)
				if !aggregationMap.Exists() {
					return fmt.Errorf("aggregation map not found: %s", customMetric.AggregationMap)
				}

				// Update the metric with the values from the aggregation map
				for _, aggregation := range aggregationMap.Array() {
					labels, value, err := getLabelsValue(customMetric, aggregation)
					if err != nil {
						return fmt.Errorf("failed to get labels and value: %w", err)
					}

					customRuleMetrics[customMetric.Name].WithLabelValues(labels...).Set(value)
				}
			}
		}

		// At the end, update the default metrics values
		for name, metric := range defaultRuleMetrics {
			for _, rule := range rules {
				switch name {
				case "searchrule_value":
					metric.WithLabelValues(rule.SearchRule.Name).Set(float64(rule.Value))
				case "searchrule_state":
					// Set the state of the rule with 1 if it's the same as the state in the ruleStates array
					for _, state := range ruleStates {
						if rule.State == state {
							metric.WithLabelValues(rule.SearchRule.Name, state).Set(1)
							continue
						}
						metric.WithLabelValues(rule.SearchRule.Name, state).Set(0)
					}
				}
			}
		}
	}
	return nil

}

// getLabelsValue returns the labels and value for a custom metric
func getLabelsValue(customMetric v1alpha1.CustomMetric, aggregation gjson.Result) ([]string, float64, error) {

	// Get the value from the rule
	value := gjson.Get(aggregation.String(), customMetric.Value).Float()

	// Get the labels from the rule
	labels := make([]string, len(customMetric.Labels))
	for i, label := range customMetric.Labels {
		if label.StaticValue {
			labels[i] = label.Value
			continue
		}
		labels[i] = gjson.Get(aggregation.String(), label.Value).String()
	}

	return labels, value, nil
}
