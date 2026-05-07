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

package v1alpha1

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Elasticsearch TODO
type Elasticsearch struct {
	Index string `json:"index"`

	ConditionField string `json:"conditionField"`

	QueryJSON string                `json:"queryJSON,omitempty"`
	Query     *apiextensionsv1.JSON `json:"query,omitempty"`
}

// Condition TODO
type Condition struct {
	Operator  string `json:"operator"`
	Threshold string `json:"threshold"`
	For       string `json:"for"`
}

// ActionRef TODO
type ActionRef struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace,omitempty"`
	Mode        string            `json:"mode,omitempty"`
	Data        string            `json:"data,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// QueryConnectorRef TODO
type QueryConnectorRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

// MetricLabel describes how to extract a single Prometheus label from each
// bucket emitted by a CustomMetric. Value is a gjson path resolved against
// the bucket object (e.g. `key`, `error_percentage.value`). When StaticValue
// is true, Value is emitted verbatim instead of being treated as a path —
// useful for tagging every sample of a metric with a constant label.
type MetricLabel struct {
	Name        string `json:"name"`
	Value       string `json:"value"`
	StaticValue bool   `json:"staticValue,omitempty"`
}

// CustomMetric exposes a Prometheus gauge derived from a list of buckets in
// the Elasticsearch response. The operator iterates AggregationMap (a gjson
// path to an array — typically a `terms` aggregation's `buckets`) and emits
// one sample per bucket with Labels mapped from the bucket fields and the
// numeric Value extracted from another path within the bucket. The metric
// is registered as `searchrule_<Name>` and always carries the additional
// labels `searchrule_namespace` and `rule` so multiple SearchRules can share
// the same custom metric name without collision.
type CustomMetric struct {
	// Name is the suffix of the Prometheus metric name. The exposed metric
	// is `searchrule_<Name>`. Must match `[a-zA-Z_][a-zA-Z_0-9]*` and must
	// not collide with the operator's reserved names (`value`, `state`).
	// +kubebuilder:validation:Pattern=`^[a-zA-Z_][a-zA-Z_0-9]*$`
	Name string `json:"name"`

	// Help is the Prometheus HELP text. Defaults to a generic description.
	Help string `json:"help,omitempty"`

	// AggregationMap is a gjson path to an array of buckets in the
	// Elasticsearch response, e.g. `aggregations.by_domain.buckets`. If the
	// path resolves to a single object instead of an array, the operator
	// emits a single sample.
	AggregationMap string `json:"aggregation_map"`

	// Labels are extracted from each bucket and attached to the sample.
	Labels []MetricLabel `json:"labels,omitempty"`

	// Value is the gjson path inside a bucket whose numeric content is
	// emitted as the gauge value. Defaults to `doc_count` when empty so a
	// bare `terms` aggregation works without configuration; for pipeline
	// aggregations such as `max_bucket` or `bucket_script`, set this
	// explicitly to e.g. `error_percentage.value` or `value`.
	Value string `json:"value,omitempty"`
}

// PrometheusRuleSpec configures the auto-generated PrometheusRule (CRD from
// the prometheus-operator project) that mirrors this SearchRule's condition
// against the searchrule_value metric exposed by the operator. When enabled,
// a PrometheusRule resource is created in the same namespace as the
// SearchRule, owned by it (so it is garbage-collected on deletion), and the
// Prometheus Operator picks it up automatically.
type PrometheusRuleSpec struct {
	// Enabled toggles the creation of the PrometheusRule for this SearchRule.
	Enabled bool `json:"enabled"`

	// AlertName overrides the alert name in the generated PrometheusRule.
	// Defaults to the SearchRule name.
	AlertName string `json:"alertName,omitempty"`

	// MetricName selects which entry of spec.customMetrics the generated
	// PromQL targets when the SearchRule defines more than one. Must match
	// one of customMetrics[*].name. When unset, the first custom metric is
	// used. Ignored when customMetrics is empty (the legacy
	// `searchrule_value` metric is used instead).
	MetricName string `json:"metricName,omitempty"`

	// Labels are merged into the alert labels.
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations are merged into the alert annotations.
	Annotations map[string]string `json:"annotations,omitempty"`
}

// SearchRuleSpec defines the desired state of SearchRule.
type SearchRuleSpec struct {
	Description       string              `json:"description,omitempty"`
	QueryConnectorRef QueryConnectorRef   `json:"queryConnectorRef"`
	CheckInterval     string              `json:"checkInterval"`
	Elasticsearch     Elasticsearch       `json:"elasticsearch"`
	Condition         Condition           `json:"condition"`
	ActionRef         *ActionRef          `json:"actionRef,omitempty"`
	PrometheusRule    *PrometheusRuleSpec `json:"prometheusRule,omitempty"`

	// CustomMetrics declares Prometheus gauges derived from the
	// Elasticsearch response aggregations. Each entry produces one
	// `searchrule_<Name>` metric with one sample per bucket of the
	// referenced aggregation. Useful when a SearchRule's condition is built
	// on top of a `max_bucket`/`min_bucket` aggregation and the alert needs
	// to expose the dimension that the bucket grouped by.
	// +kubebuilder:validation:MaxItems=10
	CustomMetrics []CustomMetric `json:"customMetrics,omitempty"`
}

// SearchRuleStatus defines the observed state of SearchRule.
type SearchRuleStatus struct {
	Conditions []metav1.Condition `json:"conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type==\"ResourceSynced\")].status",description=""
// +kubebuilder:printcolumn:name="AlertStatus",type="string",JSONPath=".status.conditions[?(@.type==\"State\")].reason",description=""
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description=""

// SearchRule is the Schema for the searchrules API.
type SearchRule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SearchRuleSpec   `json:"spec,omitempty"`
	Status SearchRuleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SearchRuleList contains a list of SearchRule.
type SearchRuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SearchRule `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SearchRule{}, &SearchRuleList{})
}
