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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// QueryConnectorSpec defines the desired state of QueryConnector.
type QueryConnectorSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	URL           string            `json:"url"`
	Headers       map[string]string `json:"headers,omitempty"`
	TlsSkipVerify bool              `json:"tlsSkipVerify,omitempty"`
	Credentials   Credentials       `json:"credentials,omitempty"`
}

// QueryConnectorStatus defines the observed state of QueryConnector.
type QueryConnectorStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Conditions []metav1.Condition `json:"conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// QueryConnector is the Schema for the QueryConnectors API.
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
