package replicator

import (
	"testing"

	"github.com/d3sw/replicator/client"
	"github.com/d3sw/replicator/notifier"
	"github.com/d3sw/replicator/replicator/structs"
	"github.com/d3sw/replicator/testutil"
)

func TestFailsafe_FaileSafeCheck(t *testing.T) {
	t.Parallel()

	c, s := testutil.MakeClientWithConfig(t)
	defer s.Stop()

	consul, _ := client.NewConsulClient(s.HTTPAddr, "")
	c.ConsulClient = consul

	// Setup a worker pool object.
	workerPool := &structs.WorkerPool{
		Name:           "example-group",
		RetryThreshold: 3,
	}

	// Setup our state object and set helper fields.
	state := &structs.ScalingState{}
	state.ResourceType = ClusterType
	state.ResourceName = workerPool.Name
	state.StatePath = "replicator/config/state/nodes/" + workerPool.Name

	// Setup failure notification message to send to failsafe methods.
	message := &notifier.FailureMessage{
		AlertUID:     "test-uid",
		ResourceID:   workerPool.Name,
		ResourceType: state.ResourceType,
	}

	// Verify the circuit breaker trips if we've exceeded the retry threshold.
	state.FailureCount = workerPool.RetryThreshold + 1
	if FailsafeCheck(state, c, workerPool.RetryThreshold, message) {
		t.Fatal("expected failsafe mode to be enabled but it was disabled")
	}

	// Verify the circuit breaker does not trip when the failure count is
	// below the retry threshold.
	state.FailsafeMode = false
	state.FailureCount = workerPool.RetryThreshold - 1
	if !FailsafeCheck(state, c, workerPool.RetryThreshold, message) {
		t.Fatal("expected failsafe mode to be disabled but it was enabled")
	}

	// Verify the circuit breaker trips when the failure count matches the retry
	// threshold.
	state.FailsafeMode = false
	state.FailureCount = workerPool.RetryThreshold
	if FailsafeCheck(state, c, workerPool.RetryThreshold, message) {
		t.Fatal("expected failsafe mode to be enabled but it was disabled")
	}

	// Verify the circuit breaker returns as tripped if it was already tripped.
	state.FailsafeMode = true
	state.FailureCount = 0
	if FailsafeCheck(state, c, workerPool.RetryThreshold, message) {
		t.Fatal("expected failsafe mode to be enabled but it was disabled")
	}
}

func TestFailsafe_SetFailsafeMode(t *testing.T) {
	t.Parallel()

	c, s := testutil.MakeClientWithConfig(t)
	defer s.Stop()

	consul, _ := client.NewConsulClient(s.HTTPAddr, "")
	c.ConsulClient = consul

	// Setup a worker pool object.
	workerPool := &structs.WorkerPool{
		Name:           "example-group",
		RetryThreshold: 3,
	}

	// Setup our state object and set helper fields.
	state := &structs.ScalingState{}
	state.FailureCount = 3
	state.FailsafeMode = true
	state.ResourceType = ClusterType
	state.ResourceName = workerPool.Name
	state.StatePath = "replicator/config/state/nodes/" + workerPool.Name

	// Setup failure notification message to send to failsafe methods.
	message := &notifier.FailureMessage{
		AlertUID:     "test-uid",
		ResourceID:   workerPool.Name,
		ResourceType: state.ResourceType,
	}

	// Verify a request to disable failsafe mode works.
	enabled := false
	SetFailsafeMode(state, c, enabled, message)

	if state.FailsafeMode != enabled {
		t.Fatalf("expected FailsafeMode to be %v but got %v", enabled,
			state.FailsafeMode)
	}

	// Verify the failure count is reset when the failsafe circuit
	// breaker is reset.
	if state.FailureCount != 0 {
		t.Fatalf("expected failure count to %v but got %v", 0, state.FailureCount)
	}
}
