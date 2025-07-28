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
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"freepik.com/searchruler/internal/controller/searchrule"
	"freepik.com/searchruler/internal/pools"
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

	return nil

}
