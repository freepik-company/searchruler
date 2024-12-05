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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RulerActionCredentials TODO
type RulerActionCredentials struct {
	SecretRef SecretRef `json:"secretRef"`
}

// WebHook TODO
type Webhook struct {
	Url           string                 `json:"url"`
	Verb          string                 `json:"verb"`
	Headers       map[string]string      `json:"headers,omitempty"`
	TlsSkipVerify bool                   `json:"tlsSkipVerify,omitempty"`
	Validator     string                 `json:"validator,omitempty"`
	Credentials   RulerActionCredentials `json:"credentials,omitempty"`
}

// RulerActionSpec defines the desired state of RulerAction.
type RulerActionSpec struct {
	Webhook Webhook `json:"webhook"`
}

// RulerActionStatus defines the observed state of RulerAction.
type RulerActionStatus struct {
	Conditions []metav1.Condition `json:"conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type==\"ResourceSynced\")].status",description=""
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[?(@.type==\"State\")].reason",description=""
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description=""

// RulerAction is the Schema for the ruleractions API.
type RulerAction struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RulerActionSpec   `json:"spec,omitempty"`
	Status RulerActionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RulerActionList contains a list of RulerAction.
type RulerActionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RulerAction `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RulerAction{}, &RulerActionList{})
}
