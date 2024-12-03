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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

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
	Name string `json:"name"`
	Data string `json:"data,omitempty"`
}

// QueryConnectorRef TODO
type QueryConnectorRef struct {
	Name string `json:"name"`
}

// SearchRuleSpec defines the desired state of SearchRule.
type SearchRuleSpec struct {
	Description       string            `json:"description,omitempty"`
	QueryConnectorRef QueryConnectorRef `json:"queryConnectorRef"`
	CheckInterval     string            `json:"checkInterval"`
	Elasticsearch     Elasticsearch     `json:"elasticsearch"`
	Condition         Condition         `json:"condition"`
	ActionRef         ActionRef         `json:"actionRef"`
}

// SearchRuleStatus defines the observed state of SearchRule.
type SearchRuleStatus struct {
	Conditions []metav1.Condition `json:"conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

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
