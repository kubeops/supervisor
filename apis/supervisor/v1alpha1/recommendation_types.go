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
	"kubeops.dev/supervisor/crds"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kmapi "kmodules.xyz/client-go/api/v1"
	"kmodules.xyz/client-go/apiextensions"
)

const (
	ResourceKindRecommendation = "Recommendation"
	ResourceRecommendation     = "recommendation"
	ResourceRecommendations    = "recommendations"
)

// RecommendationSpec defines the desired state of Recommendation
type RecommendationSpec struct {
	// Description specifies the reason why this recommendation is generated.
	// +optional
	Description string `json:"description,omitempty"`

	// VulnerabilityReport specifies any kind vulnerability report like cve fixed information
	VulnerabilityReport *VulnerabilityReport `json:"vulnerabilityReport,omitempty"`

	// Target specifies the APIGroup, Kind & Name of the target resource for which the recommendation is generated
	Target core.TypedLocalObjectReference `json:"target"`

	// Operation holds a kubernetes object yaml which will be applied when this recommendation will be executed.
	// It should be a valid kubernetes resource yaml containing apiVersion, kind and metadata fields.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:EmbeddedResource
	Operation runtime.RawExtension `json:"operation"`

	// Recommender holds the name and namespace of the component which generate this recommendation.
	Recommender kmapi.ObjectReference `json:"recommender"`

	// The recommendation will be executed within the given Deadline.
	// To maintain deadline, Parallelism can be compromised.
	// +optional
	Deadline *metav1.Time `json:"deadline,omitempty"`

	// If RequireExplicitApproval is set to `true` then the Recommendation must be Approved manually.
	// Recommendation won't be executed without manual approval and any kind of ApprovalPolicy will be ignored.
	// +optional
	RequireExplicitApproval bool `json:"requireExplicitApproval,omitempty"`

	// Rules defines OperationPhaseRules. It contains three identification rules of successful execution of the operation,
	// progressing execution of the operation & failed execution of the operation.
	// Example:
	// rules:
	//   success:    `has(self.status.phase) && self.status.phase == 'Successful'`
	//   inProgress: `has(self.status.phase) && self.status.phase == 'Progressing'`
	//   failed:     `has(self.status.phase) && self.status.phase == 'Failed'`
	Rules OperationPhaseRules `json:"rules"`

	// BackoffLimit specifies the number of retries before marking this recommendation failed.
	// By default set as five(5).
	// If BackoffLimit is zero(0), the operation will be tried to executed only once.
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=10
	BackoffLimit *int32 `json:"backoffLimit,omitempty"`
}

type ReportGenerationStatus string

const (
	ReportGenerationStatusSuccess ReportGenerationStatus = "Success"
	ReportGenerationStatusFailure ReportGenerationStatus = "Failure"
)

type VulnerabilityReport struct {
	Status  ReportGenerationStatus `json:"status,omitempty"`
	Message string                 `json:"message,omitempty"`
	// Fixed represents the list of CVEs fixed if the recommendation is applied
	Fixed *CVEReport `json:"fixed,omitempty"`
	// Known represents the list of CVEs known to exist after the recommendation is applied
	Known *CVEReport `json:"known,omitempty"`
}

type Vulnerability struct {
	VulnerabilityID string `json:"vulnerabilityID,omitempty"`
	PrimaryURL      string `json:"primaryURL,omitempty"`
	Severity        string `json:"severity,omitempty"`
}

type CVEReport struct {
	Count           map[string]int  `json:"count,omitempty"`
	Vulnerabilities []Vulnerability `json:"vulnerabilities,omitempty"`
}

// OperationPhaseRules defines three identification rules of successful execution of the operation,
// progressing execution of the operation & failed execution of the operation.
// To specifies any field of the Operation object, the rule must start with the word `self`.
// Example:
//
//	.status.phase -> self.status.phase
//	.status.observedGeneration -> self.status.observedGeneration
//
// The rules can be any valid expression supported by CEL(Common Expression Language).
// Ref: https://github.com/google/cel-spec
type OperationPhaseRules struct {
	// Success defines a rule to identify the successful execution of the operation.
	// Example:
	//   success: `has(self.status.phase) && self.status.phase == 'Successful'`
	// Here self.status.phase is pointing to .status.phase field of the Operation object.
	// When .status.phase field presents and becomes `Successful`, the Success rule will satisfy.
	Success string `json:"success"`

	// InProgress defines a rule to identify that applied operation is progressing.
	// Example:
	//   inProgress: `has(self.status.phase) && self.status.phase == 'Progressing'`
	// Here self.status.phase is pointing to .status.phase field of the Operation object.
	// When .status.phase field presents and becomes `Progressing`, the InProgress rule will satisfy.
	InProgress string `json:"inProgress"`

	// Failed defines a rule to identify that applied operation is failed.
	// Example:
	//   inProgress: `has(self.status.phase) && self.status.phase == 'Failed'`
	// Here self.status.phase is pointing to .status.phase field of the Operation object.
	// When .status.phase field presents and becomes `Failed`, the Failed rule will satisfy.
	Failed string `json:"failed"`
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
	// Specifies the Approval Status of the Recommendation.
	// Possible values are `Pending`, `Approved`, `Rejected`
	// Pending: Recommendation is yet to Approved or Rejected
	// Approved: Recommendation is permitted to execute.
	// Rejected: Recommendation is rejected and never be executed.
	// +optional
	// +kubebuilder:default=Pending
	ApprovalStatus ApprovalStatus `json:"approvalStatus"`

	// Specifies the Recommendation current phase.
	// Possible values are:
	// Pending : Recommendation misses at least one pre-requisite for executing the operation.
	//           It also tells that some user action is needed.
	// Skipped : Operation is skipped because of Rejection ApprovalStatus.
	// Waiting : Recommendation is waiting for the MaintenanceWindow to execute the operation
	//           or waiting for others Recommendation to complete far maintaining Parallelism.
	// InProgress : The operation execution is successfully started and waiting for its final status.
	// Succeeded : Operation has been successfully executed.
	// Failed : Operation execution has not completed successfully i.e. encountered an error
	// +optional
	Phase RecommendationPhase `json:"phase,omitempty"`

	// A message indicating details about Recommendation current phase.
	// +optional
	// +kubebuilder:default=WaitingForApproval
	Reason string `json:"reason"`

	// Specifies Reviewer's details.
	// +optional
	Reviewer *Subject `json:"reviewer,omitempty"`

	// Specifies Reviewer's comment.
	// +optional
	Comments string `json:"comments,omitempty"`

	// Contains review timestamp
	// +optional
	ReviewTimestamp *metav1.Time `json:"reviewTimestamp,omitempty"`

	// ApprovedWindow specifies the time window configuration for the Recommendation execution.
	// +optional
	ApprovedWindow *ApprovedWindow `json:"approvedWindow,omitempty"`

	// Parallelism imposes some restriction to Recommendation execution.
	// Possible values are:
	// Namespace: Only one Recommendation can be executed at a time in a namespace.
	// Target: Only one Recommendation for a given target can be executed at a time.
	// TargetAndNamespace: Only one Recommendation for a given target can be executed at a time in a namespace.
	// +optional
	// +kubebuilder:default=Namespace
	Parallelism Parallelism `json:"parallelism,omitempty"`

	// observedGeneration is the most recent generation observed for this resource. It corresponds to the
	// resource's generation, which is updated on mutation by the API Server.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions applied to the Recommendation.
	// +optional
	Conditions []kmapi.Condition `json:"conditions,omitempty"`

	// Outdated is indicating details whether the Recommendation is outdated or not.
	// If the value is `true`, then Recommendation will not be executed. This indicates that after generating the Recommendation,
	// the targeted resource is changed in such a way that the generated Recommendation has become outdated & can't be executed anymore.
	//
	// +optional
	// +kubebuilder:default=false
	Outdated bool `json:"outdated"`

	// CreatedOperationRef holds the created operation name.
	// +optional
	CreatedOperationRef *core.LocalObjectReference `json:"createdOperationRef,omitempty"`

	// FailedAttempt holds the number of times the operation is failed.
	// +optional
	// +kubebuilder:default=0
	FailedAttempt int32 `json:"failedAttempt"`
}

// +kubebuilder:validation:Enum=Pending;Skipped;Waiting;InProgress;Succeeded;Failed
type RecommendationPhase string

const (
	Pending    RecommendationPhase = "Pending"
	Skipped    RecommendationPhase = "Skipped"
	Waiting    RecommendationPhase = "Waiting"
	InProgress RecommendationPhase = "InProgress"
	Succeeded  RecommendationPhase = "Succeeded"
	Failed     RecommendationPhase = "Failed"
)

// +kubebuilder:validation:Enum=Immediate;NextAvailable;SpecificDates
type WindowType string

const (
	Immediate     WindowType = "Immediate"
	NextAvailable WindowType = "NextAvailable"
	SpecificDates WindowType = "SpecificDates"
)

// ApprovedWindow Scenarios:
//
// Scenario 1: User provides nothing and default MaintenanceWindow will be used. If any default window(cluster scoped or namespaced) is not found,
//
//	Recommendation will be in `Pending` state and waiting for maintenance window to be created.
//	Default MaintenanceWindow Priority: NamespaceScoped > ClusterScoped.
//	Note: If NamespaceScoped default MaintenanceWindow is found, ClusterScoped default MaintenanceWindow is skipped(if any).
//
// Scenario 2: User provides window type `Immediate` and ops request will be created immediately.
//
// Scenario 3: User provides a specific MaintenanceWindow and that will be used or an error will be thrown if given MaintenanceWindow is not found.
//
// Scenario 4: User provides window type `NextAvailable` and the ops request will be executed in the next available MaintenanceWindow.
//
//	Firstly, next namespace scoped available window will be used. If there is no MaintenanceWindow is found in the same namespace
//	then next available ClusterMaintenanceWindow will be used.
//	If there is no available Window is found in that time, Recommendation will be in `Pending` state and waiting for maintenance window
//	to be created.
//
// Scenario 5: User provides window type `SpecificDates`. In this case, user must provide at least one DateWindows in the dates field.
//
//	Otherwise controller will throw an error. DateWindow is only be used for window type `SpecificDates`
type ApprovedWindow struct {
	// Window defines the ApprovedWindow type
	// Possible values are:
	// Immediate: Recommendation will be executed immediately
	// NextAvailable: Recommendation will be executed in the next Available window
	// SpecificDates: Recommendation will be executed in the given dates.
	Window WindowType `json:"window,omitempty"`

	// MaintenanceWindow holds the reference of the MaintenanceWindow resource
	// +optional
	MaintenanceWindow *kmapi.TypedObjectReference `json:"maintenanceWindow,omitempty"`

	// Dates holds a list of DateWindow when Recommendation is permitted to execute
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

//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Outdated",type="boolean",JSONPath=".status.outdated"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Recommendation is the Schema for the recommendations API
type Recommendation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RecommendationSpec   `json:"spec,omitempty"`
	Status RecommendationStatus `json:"status,omitempty"`
}

//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
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

func (Recommendation) CustomResourceDefinition() *apiextensions.CustomResourceDefinition {
	return crds.MustCustomResourceDefinition(GroupVersion.WithResource(ResourceRecommendations))
}
