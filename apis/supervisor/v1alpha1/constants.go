package v1alpha1

const (
	DefaultMaintenanceWindowKey        = "supervisor.appscode.com/is-default-maintenance-window"
	DefaultClusterMaintenanceWindowKey = "supervisor.appscode.com/is-default-cluster-maintenance-window"
)

// List of Condition and Phase reasons
const (
	RecommendationSuccessfullyCreated = "RecommendationSuccessfullyCreated"
	SuccessfullyCreatedOperation      = "SuccessfullyCreatedOperation"
	SuccessfullyExecutedOperation     = "SuccessfullyExecutedOperation"
	WaitingForApproval                = "WaitingForApproval"
	WaitingForMaintenanceWindow       = "WaitingForMaintenanceWindow"
	StartedExecutingOperation         = "StartedExecutingOperation"
	RecommendationRejected            = "RecommendationRejected"
)
