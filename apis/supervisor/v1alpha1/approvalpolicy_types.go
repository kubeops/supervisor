/*
Copyright AppsCode Inc. and Contributors

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
	kmapi "kmodules.xyz/client-go/api/v1"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ApprovalPolicy is the Schema for the approvalpolicies API
type ApprovalPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	MaintenanceWindowRef kmapi.TypedObjectReference `json:"maintenanceWindowRef"`
	// +optional
	Targets []TargetRef `json:"targets"`
}

type Operation struct {
	metav1.GroupKind `json:",inline"`
}

type TargetRef struct {
	metav1.GroupKind `json:",inline"`
	// +optional
	Operations []Operation `json:"operations,omitempty"`
}

//+kubebuilder:object:root=true

// ApprovalPolicyList contains a list of ApprovalPolicy
type ApprovalPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ApprovalPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ApprovalPolicy{}, &ApprovalPolicyList{})
}
