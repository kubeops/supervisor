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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kmapi "kmodules.xyz/client-go/api/v1"
	"kmodules.xyz/client-go/apiextensions"
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
	// If the Timezone is not set or "" or "UTC", the given times and dates are considered as UTC.
	// If the name is "Local", the given times and dates are considered as server local timezone.
	//
	// Otherwise, the Timezone should specify a location name corresponding to a file
	// in the IANA Time Zone database, such as "Asia/Dhaka", "America/New_York", .
	// Ref: https://www.iana.org/time-zones
	//      https://en.wikipedia.org/wiki/List_of_tz_database_time_zones
	// +optional
	Timezone *string `json:"timezone,omitempty"`
	// Days consists of a map of DayOfWeek and corresponding list of TimeWindow.
	// There is `Logical OR` relationship between Days and Dates.
	// Example:
	//  days:
	//    Monday:
	//     - start: 10:40AM
	//       end: 7:00PM
	// +optional
	Days map[DayOfWeek][]TimeWindow `json:"days,omitempty"`
	// Dates consists of a list of Dates as Maintenance time.
	// Dates are always needed to be given in UTC format.
	// Format: yyyy-mm-ddThh.mm.ssZ [Here Z stands for Zero time zone / UTC time zone / GMT (+0000)]
	// Example:
	//  dates:
	//   - start: 2022-01-24T00:00:18Z
	//     end: 2022-01-24T23:41:18Z
	// +optional
	Dates []DateWindow `json:"dates,omitempty"`
}

// +kubebuilder:validation:Enum=Sunday;Monday;Tuesday;Wednesday;Thursday;Friday;Saturday
type DayOfWeek string

const (
	Sunday    DayOfWeek = "Sunday"
	Monday    DayOfWeek = "Monday"
	Tuesday   DayOfWeek = "Tuesday"
	Wednesday DayOfWeek = "Wednesday"
	Thursday  DayOfWeek = "Thursday"
	Friday    DayOfWeek = "Friday"
	Saturday  DayOfWeek = "Saturday"
)

type DateWindow struct {
	Start metav1.Time `json:"start"`
	End   metav1.Time `json:"end"`
}

type TimeWindow struct {
	Start kmapi.TimeOfDay `json:"start"`
	End   kmapi.TimeOfDay `json:"end"`
}

// MaintenanceWindowStatus defines the observed state of MaintenanceWindow
type MaintenanceWindowStatus struct {
	// Specifies the current phase of the database
	// +optional
	// +kubebuilder:default=Pending
	Status ApprovalStatus `json:"status,omitempty"`
	// observedGeneration is the most recent generation observed for this resource. It corresponds to the
	// resource's generation, which is updated on mutation by the API Server.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// Conditions applied to the database, such as approval or denial.
	// +optional
	Conditions []kmapi.Condition `json:"conditions,omitempty"`
}

//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Default",type="boolean",JSONPath=".spec.isDefault"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// MaintenanceWindow is the Schema for the maintenancewindows API
type MaintenanceWindow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MaintenanceWindowSpec   `json:"spec,omitempty"`
	Status MaintenanceWindowStatus `json:"status,omitempty"`
}

//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
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

func (MaintenanceWindow) CustomResourceDefinition() *apiextensions.CustomResourceDefinition {
	return crds.MustCustomResourceDefinition(GroupVersion.WithResource(ResourceMaintenanceWindows))
}
