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
	core "k8s.io/api/core/v1"
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
	Description string                         `json:"description,omitempty"`
	Target      core.TypedLocalObjectReference `json:"target"`
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:EmbeddedResource
	Operation   runtime.RawExtension  `json:"operation"`
	Recommender kmapi.ObjectReference `json:"recommender"`
	// +optional
	Deadline *metav1.Time `json:"deadline,omitempty"`
}

// +kubebuilder:validation:Enum=Pending;Approved;Rejected
type ApprovalStatus string

const (
	ApprovalPending  ApprovalStatus = "Pending"
	ApprovalApproved ApprovalStatus = "Approved"
	ApprovalRejected ApprovalStatus = "Rejected"
)

// Subject contains a reference to the object or user identities a role binding applies to.  This can either hold a direct API object reference,
// or a value for non-objects such as user and group names.
// +structType=atomic
type Subject struct {
	// Kind of object being referenced. Values defined by this API group are "User", "Group", and "ServiceAccount".
	// If the Authorizer does not recognized the kind value, the Authorizer should report an error.
	Kind string `json:"kind"`
	// APIGroup holds the API group of the referenced subject.
	// Defaults to "" for ServiceAccount subjects.
	// Defaults to "rbac.authorization.k8s.io" for User and Group subjects.
	// +optional
	APIGroup string `json:"apiGroup,omitempty" protobuf:"bytes,2,opt.name=apiGroup"`
	// Name of the object being referenced.
	Name string `json:"name" protobuf:"bytes,3,opt,name=name"`
	// Namespace of the referenced object.  If the object kind is non-namespace, such as "User" or "Group", and this value is not empty
	// the Authorizer should report an error.
	// +optional
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,4,opt,name=namespace"`
}

// RecommendationStatus defines the observed state of Recommendation
type RecommendationStatus struct {
	// +optional
	// +kubebuilder:default=UnderReview
	Status ApprovalStatus `json:"status"`
	// +optional
	Reviewer *Subject `json:"reviewer,omitempty"`
	// +optional
	Comments string `json:"comments,omitempty"`
	// +optional
	ApprovedWindow *ApprovedWindow `json:"approvedWindow,omitempty"`
	// +optional
	// +kubebuilder:default=Namespace
	Parallelism Parallelism `json:"parallelism,omitempty"`
	// observedGeneration is the most recent generation observed for this resource. It corresponds to the
	// resource's generation, which is updated on mutation by the API Server.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// Conditions applied to the database, such as approval or denial.
	// +optional
	Conditions []kmapi.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:validation:Enum=Immediate;NextAvailable;SpecificDates
type WindowType string

const (
	Immediately   WindowType = "Immediate"
	NextAvailable WindowType = "NextAvailable"
	SpecificDates WindowType = "SpecificDates"
)

type ApprovedWindow struct {
	Window WindowType `json:"window,omitempty"`
	// +optional
	MaintenanceWindow kmapi.TypedObjectReference `json:"maintenanceWindow"`
	// +optional
	Dates []DateWindow `json:"dates,omitempty"`
}

// +kubebuilder:validation:Enum=Namespace;Target;TargetAndNamespace
type Parallelism string

const (
	QueuePerNamespace          Parallelism = "Namespace"
	QueuePerTarget             Parallelism = "Target"
	QueuePerTargetAndNamespace Parallelism = "TargetAndNamespace"
)

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
