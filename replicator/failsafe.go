package replicator

import (
	"fmt"
	"time"

	"github.com/d3sw/replicator/logging"
	"github.com/d3sw/replicator/notifier"
	"github.com/d3sw/replicator/replicator/structs"
)

// Provides failsafe notification types.
const (
	ClusterType = "worker_pool"
	JobType     = "job_group"
)

const (
	clusterMessage = "cluster_failsafe_mode"
	jobMessage     = "job_group_failsafe_mode"
)

// FailsafeCheck implements the failsafe mode circuit breaker that will
// trip automatically if enough critical failures are detected. Once
// tripped, the circuit breaker must be reset by a human operator.
func FailsafeCheck(state *structs.ScalingState, config *structs.Config,
	threshold int, message *notifier.FailureMessage) (passing bool) {
	// Assume we're in a good state until proven otherwise.
	passing = true

	// If the failsafe circuit breaker has been tripped already, we can fail
	// quickly here.
	if state.FailsafeMode {
		return false
	}

	// If scaling attempt failures have reached or exceed the threshold,
	// trip the failsafe circuit breaker.
	if state.FailureCount >= threshold {
		passing = false
	}

	switch passing {
	case true:
		logging.Debug("core/failsafe: the failsafe check passes for %v %v, "+
			"scaling operations should be permitted", message.ResourceType,
			message.ResourceID)
	case false:
		SetFailsafeMode(state, config, true, message)
	}

	return
}

// SetFailsafeMode is used to toggle the distributed failsafe mode lock.
func SetFailsafeMode(state *structs.ScalingState, config *structs.Config,
	enabled bool, message *notifier.FailureMessage) (err error) {

	switch enabled {
	case true:
		if !state.FailsafeMode {
			// Send a notification message.
			sendFailsafeNotification(message, state, config)
		}

		// Suppress logging output if we're being called from the CLI tools.
		if !state.FailsafeAdmin {
			logging.Warning("core/failsafe: %v %v has been placed in failsafe "+
				"mode. No scaling operations will be permitted from any running "+
				"copies of Replicator until failsafe is administratively disabled",
				message.ResourceType, message.ResourceID)
		}

	case false:
		// Reset the failure count to allow Replicator to start with a clean
		// slate during the next evaluation.
		state.FailureCount = 0

		if !state.FailsafeAdmin {
			logging.Info("core/failsafe: disabling failsafe mode for %v %v",
				message.ResourceType, message.ResourceID)
		}
	}

	// Toggle failsafe mode flag.
	state.FailsafeMode = enabled

	// Attempt to update the persistent state tracking information.
	err = config.ConsulClient.PersistState(state)

	if err != nil {
		return fmt.Errorf("core/failsafe: an attempt to update the persistent "+
			"state tracking information failed: %v", err)
	}

	return nil
}

// sendFailsafeNotification is used to setup a notification for either
// jobscaling or clusterscaling failure and send this to all configured backends.
func sendFailsafeNotification(message *notifier.FailureMessage,
	state *structs.ScalingState, config *structs.Config) {

	switch message.ResourceType {
	case ClusterType:
		message.Reason = clusterMessage

	case JobType:
		message.Reason = jobMessage
	}

	// Add state path to failure message.
	message.StatePath = state.StatePath

	// If a notification backend is configured, send a notification message.
	if len(config.Notification.Notifiers) > 0 {
		for _, not := range config.Notification.Notifiers {
			not.SendNotification(*message)
		}

		// Update last notification event timestamp.
		state.LastNotificationEvent = time.Now()

		// Write state tracking information to persistent storage.
		if err := config.ConsulClient.PersistState(state); err != nil {
			logging.Error("core/failsafe: %v", err)
		}
	}

}
