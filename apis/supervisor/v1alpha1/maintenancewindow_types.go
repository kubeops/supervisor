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

// MaintenancewindowSpec defines the desired state of Maintenancewindow
type MaintenancewindowSpec struct {
	// +optional
	Sunday []TimeWindow `json:"sunday,omitempty"`
	// +optional
	Monday []TimeWindow `json:"monday,omitempty"`
	// +optional
	Tuesday []TimeWindow `json:"tuesday,omitempty"`
	// +optional
	Wednesday []TimeWindow `json:"wednesday,omitempty"`
	// +optional
	Thursday []TimeWindow `json:"thursday,omitempty"`
	// +optional
	Friday []TimeWindow `json:"friday,omitempty"`
	// +optional
	Saturday []TimeWindow `json:"saturday,omitempty"`
}

type TimeWindow struct {
	NotBefore kmapi.TimeOfDay `json:"notBefore"`
	NotAfter  kmapi.TimeOfDay `json:"notAfter"`
}

// MaintenancewindowStatus defines the observed state of Maintenancewindow
type MaintenancewindowStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Maintenancewindow is the Schema for the maintenancewindows API
type Maintenancewindow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MaintenancewindowSpec   `json:"spec,omitempty"`
	Status MaintenancewindowStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MaintenancewindowList contains a list of Maintenancewindow
type MaintenancewindowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Maintenancewindow `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Maintenancewindow{}, &MaintenancewindowList{})
}
