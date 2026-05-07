/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package v1alpha1

import (
	"fmt"
	"regexp"
)

// promIdentifierRe is the strict subset of the Prometheus metric/label name
// grammar `[a-zA-Z_][a-zA-Z0-9_]*`. Colon is reserved for recording rules
// and not allowed in user-supplied identifiers.
var promIdentifierRe = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// reservedMetricNames are the operator's own gauges. A custom metric whose
// `searchrule_<Name>` resolves to one of these is rejected to avoid
// shadowing the legacy series.
var reservedMetricNames = map[string]struct{}{
	"searchrule_value": {},
	"searchrule_state": {},
}

// reservedLabelNames are the labels the operator emits implicitly on every
// custom metric sample. A user label with the same name would be silently
// overwritten at sample-emission time, so we reject it up-front.
var reservedLabelNames = map[string]struct{}{
	"searchrule_namespace": {},
	"rule":                 {},
}

// ValidateCustomMetricName returns nil iff name is a valid suffix for a
// `searchrule_<name>` Prometheus metric and does not collide with the
// operator's reserved gauges. Lives in the API package so any consumer
// (operator runtime, admission webhook, tests) can call it without import
// cycles.
func ValidateCustomMetricName(name string) error {
	if name == "" {
		return fmt.Errorf("customMetric.name is required")
	}
	if !promIdentifierRe.MatchString(name) {
		return fmt.Errorf("customMetric.name %q must match %s",
			name, promIdentifierRe.String())
	}
	full := "searchrule_" + name
	if _, reserved := reservedMetricNames[full]; reserved {
		return fmt.Errorf("customMetric.name %q resolves to reserved metric %q",
			name, full)
	}
	return nil
}

// Validate checks the entire CustomMetric for the static issues we can
// detect before talking to Elasticsearch (name shape, reserved name, label
// shape, reserved labels, duplicate label names). Runtime issues such as
// missing aggregation paths or non-numeric values surface separately
// during reconciliation.
func (cm CustomMetric) Validate() error {
	if err := ValidateCustomMetricName(cm.Name); err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(cm.Labels))
	for _, lbl := range cm.Labels {
		if !promIdentifierRe.MatchString(lbl.Name) {
			return fmt.Errorf("customMetric %q label name %q must match %s",
				cm.Name, lbl.Name, promIdentifierRe.String())
		}
		if _, reserved := reservedLabelNames[lbl.Name]; reserved {
			return fmt.Errorf("customMetric %q label name %q is reserved (the operator emits it implicitly)",
				cm.Name, lbl.Name)
		}
		// Prometheus' GaugeVec rejects duplicate label names at
		// registration time with a generic error; catch it here so the
		// failure shows up in `kubectl get searchrule` instead of only
		// in the operator's logs.
		if _, dup := seen[lbl.Name]; dup {
			return fmt.Errorf("customMetric %q has duplicate label name %q",
				cm.Name, lbl.Name)
		}
		seen[lbl.Name] = struct{}{}
	}
	return nil
}
