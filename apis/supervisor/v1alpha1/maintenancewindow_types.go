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
	"kmodules.xyz/client-go/apiextensions"
	"kubeops.dev/supervisor/crds"
)

const (
	ResourceKindMaintenanceWindow = "MaintenanceWindow"
	ResourceMaintenanceWindow     = "maintenancewindow"
	ResourceMaintenanceWindows    = "maintenancewindows"
)

// MaintenanceWindowSpec defines the desired state of MaintenanceWindow
type MaintenanceWindowSpec struct {
	// +optional
	IsDefault bool `json:"isDefault,omitempty"`
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

// MaintenanceWindowStatus defines the observed state of MaintenanceWindow
type MaintenanceWindowStatus struct {
	// Specifies the current phase of the database
	// +optional
	// +kubebuilder:default=UnderReview
	Status ApprovalStatus `json:"status,omitempty"`
	// observedGeneration is the most recent generation observed for this resource. It corresponds to the
	// resource's generation, which is updated on mutation by the API Server.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// Conditions applied to the database, such as approval or denial.
	// +optional
	Conditions []kmapi.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// MaintenanceWindow is the Schema for the maintenancewindows API
type MaintenanceWindow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MaintenanceWindowSpec   `json:"spec,omitempty"`
	Status MaintenanceWindowStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MaintenanceWindowList contains a list of MaintenanceWindow
type MaintenanceWindowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MaintenanceWindow `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MaintenanceWindow{}, &MaintenanceWindowList{})
}

func (_ MaintenanceWindow) CustomResourceDefinition() *apiextensions.CustomResourceDefinition {
	return crds.MustCustomResourceDefinition(GroupVersion.WithResource(ResourceMaintenanceWindows))
}
