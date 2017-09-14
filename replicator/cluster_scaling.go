package replicator

import (
	"sync"
	"time"

	metrics "github.com/armon/go-metrics"
	"github.com/elsevier-core-engineering/replicator/client"
	"github.com/elsevier-core-engineering/replicator/logging"
	"github.com/elsevier-core-engineering/replicator/notifier"
	"github.com/elsevier-core-engineering/replicator/replicator/structs"
)

// asyncClusterScaling triggers concurrent cluster scaling operations for
// each worker pool in the node registry.
func (r *Runner) asyncClusterScaling(nodeRegistry *structs.NodeRegistry,
	jobRegistry *structs.JobScalingPolicies) {

	// Setup our wait group to ensure we block until all worker pool scaling
	// operations have completed.
	var wg sync.WaitGroup

	// Register an entry to the wait group for each worker pool.
	wg.Add(len(nodeRegistry.WorkerPools))

	for _, workerPool := range nodeRegistry.WorkerPools {
		go r.workerPoolScaling(workerPool.Name, nodeRegistry, jobRegistry, &wg)
	}

	// Block on all worker pool scaling threads.
	wg.Wait()

	return
}

// workerPoolScaling is a thread safe method for scaling an individual
// worker pool.
func (r *Runner) workerPoolScaling(poolName string,
	nodeRegistry *structs.NodeRegistry, jobs *structs.JobScalingPolicies,
	wg *sync.WaitGroup) {

	// Inform the wait group we have finished our task upon completion.
	defer wg.Done()

	// Obtain a read-only lock on the Node registry, grab a reference to
	// our worker pool object and release the lock.
	nodeRegistry.Lock.RLock()
	workerPool := nodeRegistry.WorkerPools[poolName]
	nodeRegistry.Lock.RUnlock()

	// Setup references to clients for Nomad and Consul.
	nomadClient := r.config.NomadClient
	consulClient := r.config.ConsulClient

	// Initialize a new disposable capacity object.
	poolCapacity := &structs.ClusterCapacity{}

	// Initialize a new scaling state object and set helper fields.
	workerPool.State = &structs.ScalingState{}
	workerPool.State.ResourceType = ClusterType
	workerPool.State.ResourceName = workerPool.Name
	workerPool.State.StatePath = r.config.ConsulKeyRoot + "/state/nodes/" +
		workerPool.Name

	// Attempt to load state from persistent storage.
	consulClient.ReadState(workerPool.State, true)

	// Setup a failure message to pass to the failsafe check.
	msg := &notifier.FailureMessage{
		AlertUID:     workerPool.NotificationUID,
		ResourceID:   workerPool.Name,
		ResourceType: ClusterType,
	}

	// If the worker pool is in failsafe mode, decline to perform any scaling
	// evaluation or action.
	if !FailsafeCheck(workerPool.State, r.config, workerPool.RetryThreshold, msg) {
		logging.Warning("core/cluster_scaling: worker pool %v is in failsafe "+
			"mode, no scaling evaluations will be performed", workerPool.Name)
		return
	}

	// Evaluate worker pool to determine if a scaling operation is required.
	scale, err := nomadClient.EvaluatePoolScaling(poolCapacity, workerPool, jobs)
	if err != nil || !scale {
		logging.Debug("core/cluster_scaling: scaling operation for worker pool %v "+
			"is either not required or not permitted: %v", workerPool.Name, err)
		return
	}

	// Copy the desired scsaling direction to the state object.
	workerPool.State.ScalingDirection = poolCapacity.ScalingDirection

	// Attempt to update state tracking information in Consul.
	if err = consulClient.PersistState(workerPool.State); err != nil {
		logging.Error("core/cluster_scaling: %v", err)
	}

	// Call the scaling provider safety check to determine if we should
	// proceed with scaling evaluation.
	if scale := workerPool.ScalingProvider.SafetyCheck(workerPool); !scale {
		logging.Debug("core/cluster_scaling: scaling operation for worker pool %v"+
			"is not permitted by the scaling provider", workerPool.Name)
		return
	}

	// Determine if the scaling cooldown threshold has been met.
	ok := checkCooldownThreshold(workerPool)
	if !ok {
		return
	}

	// Determine if we've reached the required number of consecutive scaling
	// requests.
	ok = checkPoolScalingThreshold(workerPool, r.config)
	if !ok {
		return
	}

	if poolCapacity.ScalingDirection == structs.ScalingDirectionOut {
		// Initiate cluster scaling operation by calling the scaling provider.
		err = workerPool.ScalingProvider.Scale(workerPool, r.config, nodeRegistry)
		if err != nil {
			logging.Error("core/cluster_scaling: an error occurred while "+
				"attempting a scaling operation against worker pool %v: %v",
				workerPool.Name, err)
			return
		}

		// Obtain a read/write lock on the node registry, write the worker
		// pool state object back to the node registry and release the lock.
		nodeRegistry.Lock.Lock()
		nodeRegistry.WorkerPools[workerPool.Name].State = workerPool.State
		nodeRegistry.Lock.Unlock()
	}

	if poolCapacity.ScalingDirection == client.ScalingDirectionIn {
		// Identify the least allocated node in the worker pool.
		nodeID, nodeIP := nomadClient.LeastAllocatedNode(poolCapacity, workerPool.State)
		if nodeIP == "" || nodeID == "" {
			logging.Error("core/cluster_scaling: unable to identify the least "+
				"allocated node in worker pool %v", workerPool.Name)
			return
		}

		// Register the least allocated node as eligible for scaling actions.
		workerPool.State.EligibleNodes = append(workerPool.State.EligibleNodes,
			nodeIP)

		// Place the least allocated noded in drain mode.
		if err = nomadClient.DrainNode(nodeID); err != nil {
			logging.Error("core/cluster_scaling: an error occurred while "+
				"attempting to place node %v from worker pool %v in drain mode: "+
				"%v", nodeID, workerPool.Name, err)
			return
		}

		// Initiate cluster scaling operation by calling the scaling provider.
		err := workerPool.ScalingProvider.Scale(workerPool, r.config, nodeRegistry)
		if err != nil {
			logging.Error("core/cluster_scaling: an error occurred while "+
				"attempting a scaling operation against worker pool %v: %v",
				workerPool.Name, err)
			return
		}

		// Obtain a read/write lock on the node registry, write the worker
		// pool state object back to the node registry and release the lock.
		nodeRegistry.Lock.Lock()
		nodeRegistry.WorkerPools[workerPool.Name].State = workerPool.State
		nodeRegistry.Lock.Unlock()
	}

	metrics.IncrCounter([]string{"cluster", "scale_out_success"}, 1)

	return
}

// checkPoolScalingThreshold determines if we've reached the required number
// of consecutive scaling attempts.
func checkPoolScalingThreshold(workerPool *structs.WorkerPool,
	config *structs.Config) (scale bool) {

	switch workerPool.State.ScalingDirection {
	case structs.ScalingDirectionIn:
		workerPool.State.ScaleInRequests++
		workerPool.State.ScaleOutRequests = 0

		if workerPool.State.ScaleInRequests == workerPool.ScalingThreshold {
			logging.Debug("core/cluster_scaling: cluster scale-in request %v for "+
				"worker pool %v meets the threshold %v",
				workerPool.State.ScaleInRequests, workerPool.Name,
				workerPool.ScalingThreshold)

			workerPool.State.ScaleInRequests = 0
			scale = true
		} else {
			logging.Debug("core/cluster_scaling: cluster scale-in request %v for "+
				"worker pool %v does not meet the threshold %v",
				workerPool.State.ScaleInRequests, workerPool.Name,
				workerPool.ScalingThreshold)
		}

	case structs.ScalingDirectionOut:
		workerPool.State.ScaleOutRequests++
		workerPool.State.ScaleInRequests = 0

		if workerPool.State.ScaleOutRequests == workerPool.ScalingThreshold {
			logging.Debug("core/cluster_scaling: cluster scale-out request %v for "+
				"worker pool %v meets the threshold %v",
				workerPool.State.ScaleOutRequests, workerPool.Name,
				workerPool.ScalingThreshold)

			workerPool.State.ScaleOutRequests = 0
			scale = true
		} else {
			logging.Debug("core/cluster_scaling: cluster scale-out request %v for "+
				"worker pool %v does not meet the threshold %v",
				workerPool.State.ScaleOutRequests, workerPool.Name,
				workerPool.ScalingThreshold)
		}

	default:
		workerPool.State.ScaleInRequests = 0
		workerPool.State.ScaleOutRequests = 0
	}

	if err := config.ConsulClient.PersistState(workerPool.State); err != nil {
		logging.Error("core:cluster_scaling: unable to update cluster scaling "+
			"state to persistent store at path %v: %v",
			workerPool.State.StatePath, err)
		scale = false
	}

	return scale
}

// CheckCooldownThreshold checks to see if a scaling cooldown threshold has
// been reached.
func checkCooldownThreshold(workerPool *structs.WorkerPool) bool {
	if workerPool.State.LastScalingEvent.IsZero() {
		logging.Debug("core/cluster_scaling: no previous scaling operations for "+
			"worker pool %v have occurred, scaling operations should be permitted.",
			workerPool.Name)
		return true
	}

	// Calculate the cooldown threshold.
	cooldown := workerPool.State.LastScalingEvent.Add(
		time.Duration(workerPool.Cooldown) * time.Second)

	if time.Now().Before(cooldown) {
		logging.Info("core/cluster_scaling: cluster scaling cooldown threshold "+
			"has not been reached: %v, scaling operations for worker pool %v should "+
			"not be permitted", cooldown, workerPool.Name)
		return false
	}

	logging.Debug("core/cluster_scaling: cluster scaling cooldown threshold %v "+
		"has been reached, scaling operations for worker pool %v should be "+
		"permitted", cooldown, workerPool.Name)

	return true
}
