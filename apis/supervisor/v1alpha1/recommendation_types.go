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
	"k8s.io/apimachinery/pkg/runtime"
	kmapi "kmodules.xyz/client-go/api/v1"
	"kmodules.xyz/client-go/apiextensions"
	"kubeops.dev/supervisor/crds"
)

const (
	ResourceKindRecommendation = "Recommendation"
	ResourceRecommendation     = "recommendation"
	ResourceRecommendations    = "recommendations"
)

// RecommendationSpec defines the desired state of Recommendation
type RecommendationSpec struct {
	Description string `json:"description,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:EmbeddedResource
	Operation runtime.RawExtension `json:"operation"`
	// +optional
	MaintenanceWindow *kmapi.TypedObjectReference `json:"maintenanceWindow,omitempty"`
	// +optional
	// +kubebuilder:default=UnderReview
	Status ApprovalStatus `json:"status"`
	// +optional
	Approver *ApproverInfo `json:"approver,omitempty"`
}

// +kubebuilder:validation:Enum=UnderReview;Approved;Rejected
type ApprovalStatus string

const (
	RecommendationUnderReview ApprovalStatus = "UnderReview"
	RecommendationApproved    ApprovalStatus = "Approved"
	RecommendationRejected    ApprovalStatus = "Rejected"
)

type ApproverInfo struct {
	Name  string `json:"name"`
	Notes string `json:"notes,omitempty"`
}

// RecommendationStatus defines the observed state of Recommendation
type RecommendationStatus struct {
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

// Recommendation is the Schema for the recommendations API
type Recommendation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RecommendationSpec   `json:"spec,omitempty"`
	Status RecommendationStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RecommendationList contains a list of Recommendation
type RecommendationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Recommendation `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Recommendation{}, &RecommendationList{})
}

func (_ Recommendation) CustomResourceDefinition() *apiextensions.CustomResourceDefinition {
	return crds.MustCustomResourceDefinition(GroupVersion.WithResource(ResourceRecommendations))
}
