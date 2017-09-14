package structs

import (
	"sync"
	"time"
)

// ScalingState provides a state object that represents the state
// of a scalable worker pool or job group.
type ScalingState struct {
	// EligibleNodes tracks nodes marked as eligible for targeted scaling actions
	EligibleNodes []string `json:"eligible_nodes"`

	// FailsafeAdmin tracks whether failsafe mode is being toggled via the CLI
	// tools.
	FailsafeAdmin bool `json:"failsafe_admin"`

	// FailsafeMode represents the status of the failsafe circuit breaker. This
	// will be tripped automatically when enough consecutive failures are
	// encountered.
	FailsafeMode bool `json:"failsafe_mode"`

	// FailureCount tracks the number of worker nodes that have failed to
	// successfully join the worker pool after a scale-out operation.
	FailureCount int `json:"failure_count"`

	// LastNotificationEvent tracks the time of the last notification send run
	// for this state object.
	LastNotificationEvent time.Time `json:"last_notification_event"`

	// LastScalingEvent represents the last time the daemon successfully
	// completed a cluster scaling action.
	LastScalingEvent time.Time `json:"last_scaling_event"`

	// LastUpdated tracks the last time the state tracking data was updated.
	LastUpdated time.Time `json:"last_updated"`

	// Lock provides a mutex lock to protect concurrent read/write
	// access to the object.
	Lock sync.RWMutex `json:"-"`

	// ProtectedNode represents the Nomad agent node on which the Replicator
	// leader is running. This node will be excluded when identifying an eligible
	// node for termination during scaling actions.
	ProtectedNode string `json:"protected_node"`

	// ResourceName provides a shortcut method for identifying the resource
	// this state is associated with.
	ResourceName string `json:"resource_name"`

	// ResourceType represents the type of resource being tracked by this object.
	ResourceType string `json:"resource_type"`

	// ScaleInRequests tracks the number of consecutive times replicator
	// has indicated the cluster worker pool should be scaled in.
	ScaleInRequests int `json:"scalein_requests"`

	// ScaleOutRequests tracks the number of consecutive times replicator
	// has indicated the cluster worker pool should be scaled out.
	ScaleOutRequests int `json:"scaleout_requests"`

	// ScalingDirection tracks the desired scaling direction of the resource,
	// when a scaling operation is being requested.
	ScalingDirection string `json:"-"`

	// StatePath stores the path where the object should be persisted.
	StatePath string `json:"state_path"`
}
