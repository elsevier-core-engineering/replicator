package replicator

import (
	"testing"

	"github.com/elsevier-core-engineering/replicator/client"
	"github.com/elsevier-core-engineering/replicator/replicator/structs"
	"github.com/elsevier-core-engineering/replicator/testutil"
)

func TestClusterScaling_scalingThreshold(t *testing.T) {
	t.Parallel()

	c, s := testutil.MakeClientWithConfig(t)
	defer s.Stop()

	consul, _ := client.NewConsulClient(s.HTTPAddr, "")
	c.ConsulClient = consul

	state := &structs.ScalingState{}

	workerPool := &structs.WorkerPool{}
	workerPool.Name = "example-pool"
	workerPool.ScalingThreshold = 3

	state.StatePath = "replicator/config/state/nodes/" + workerPool.Name
	workerPool.State = state

	// Check ScaleOut scenarios.
	workerPool.State.ScaleOutRequests = 2
	workerPool.State.ScalingDirection = structs.ScalingDirectionOut
	if !checkPoolScalingThreshold(workerPool, c) {
		t.Fatal("expected ClusterScaleOut to answer true but got false")
	}

	workerPool.State.ScaleOutRequests = 1
	workerPool.State.ScalingDirection = structs.ScalingDirectionOut
	if checkPoolScalingThreshold(workerPool, c) {
		t.Fatal("expected ClusterScaleOut to answer false but got true")
	}

	// Check ScaleIn scenarios.
	workerPool.State.ScaleInRequests = 2
	workerPool.State.ScalingDirection = structs.ScalingDirectionIn
	if !checkPoolScalingThreshold(workerPool, c) {
		t.Fatal("expected ClusterScaleIn to answer true but got false")
	}

	workerPool.State.ScaleInRequests = 1
	workerPool.State.ScalingDirection = structs.ScalingDirectionIn
	if checkPoolScalingThreshold(workerPool, c) {
		t.Fatal("expected ClusterScaleIn to answer false but got true")
	}

	// Check the default return and state setting.
	workerPool.State.ScalingDirection = structs.ScalingDirectionNone
	if checkPoolScalingThreshold(workerPool, c) {
		t.Fatal("expected ClusterScalingNone to answer false but got true")
	}

	if state.ScaleInRequests != 0 || state.ScaleOutRequests != 0 {
		t.Fatalf("expected state scale requests to be 0, got %v and %v",
			state.ScaleInRequests, state.ScaleOutRequests)
	}
}
