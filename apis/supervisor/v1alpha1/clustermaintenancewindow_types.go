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
)

// ClusterMaintenancewindowSpec defines the desired state of ClusterMaintenancewindow
type ClusterMaintenancewindowSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of ClusterMaintenancewindow. Edit clustermaintenancewindow_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// ClusterMaintenancewindowStatus defines the observed state of ClusterMaintenancewindow
type ClusterMaintenancewindowStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// ClusterMaintenancewindow is the Schema for the clustermaintenancewindows API
type ClusterMaintenancewindow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterMaintenancewindowSpec   `json:"spec,omitempty"`
	Status ClusterMaintenancewindowStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ClusterMaintenancewindowList contains a list of ClusterMaintenancewindow
type ClusterMaintenancewindowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterMaintenancewindow `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterMaintenancewindow{}, &ClusterMaintenancewindowList{})
}
