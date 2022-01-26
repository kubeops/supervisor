package v1alpha1

const (
	DefaultMaintenanceWindowKey        = "supervisor.appscode.com/is-default-maintenance-window"
	DefaultClusterMaintenanceWindowKey = "supervisor.appscode.com/is-default-cluster-maintenance-window"
)

// List of Condition reasons
const (
	RecommendationSuccessfullyCreated = "RecommendationSuccessfullyCreated"
	SuccessfullyCreatedOperation      = "SuccessfullyCreatedOperation"
	SuccessfullyExecutedOperation     = "SuccessfullyExecutedOperation"
)
