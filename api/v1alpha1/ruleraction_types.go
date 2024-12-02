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

// RulerActionCredentials TODO
type RulerActionCredentials struct {
	SecretRef SecretRef `json:"secretRef"`
}

// WebHook TODO
type Webhook struct {
	Url         string                 `json:"url"`
	Verb        string                 `json:"verb"`
	Headers     map[string]string      `json:"headers,omitempty"`
	Validator   string                 `json:"validator,omitempty"`
	Credentials RulerActionCredentials `json:"credentials,omitempty"`
}

// RulerActionSpec defines the desired state of RulerAction.
type RulerActionSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Webhook        Webhook `json:"webhook"`
	FiringInterval string  `json:"firingInterval"`
}

// RulerActionStatus defines the observed state of RulerAction.
type RulerActionStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Conditions []metav1.Condition `json:"conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// RulerAction is the Schema for the RulerActions API.
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
