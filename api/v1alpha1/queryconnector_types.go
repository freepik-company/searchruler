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

// QueryConnectorCredentials TODO
type QueryConnectorCredentials struct {
	SyncInterval string    `json:"syncInterval,omitempty"`
	SecretRef    SecretRef `json:"secretRef"`
}

// QueryConnectorSpec defines the desired state of QueryConnector.
type QueryConnectorSpec struct {
	URL           string                    `json:"url"`
	Headers       map[string]string         `json:"headers,omitempty"`
	TlsSkipVerify bool                      `json:"tlsSkipVerify,omitempty"`
	Credentials   QueryConnectorCredentials `json:"credentials,omitempty"`
}

// QueryConnectorStatus defines the observed state of QueryConnector.
type QueryConnectorStatus struct {
	Conditions []metav1.Condition `json:"conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type==\"ResourceSynced\")].status",description=""
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[?(@.type==\"State\")].reason",description=""
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description=""

// QueryConnector is the Schema for the queryconnectors API.
type QueryConnector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   QueryConnectorSpec   `json:"spec,omitempty"`
	Status QueryConnectorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// QueryConnectorList contains a list of QueryConnector.
type QueryConnectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []QueryConnector `json:"items"`
}

func init() {
	SchemeBuilder.Register(&QueryConnector{}, &QueryConnectorList{})
}
