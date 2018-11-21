package replicator

import (
	"fmt"
	"strings"
	"sync"
	"time"

	metrics "github.com/armon/go-metrics"

	"github.com/d3sw/replicator/client"
	"github.com/d3sw/replicator/logging"
	"github.com/d3sw/replicator/notifier"
	"github.com/d3sw/replicator/replicator/structs"
)

// asyncClusterScaling triggers concurrent cluster scaling operations for
// each worker pool in the node registry.
func (s *Server) asyncClusterScaling(nodeRegistry *structs.NodeRegistry,
	jobRegistry *structs.JobScalingPolicies) {

	// Setup our wait group to ensure we block until all worker pool scaling
	// operations have completed.
	var wg sync.WaitGroup

	// Get the current number of registered worker pools.
	poolCount := len(nodeRegistry.WorkerPools)

	// Register an entry to the wait group for each worker pool.
	wg.Add(poolCount)

	// Build a buffered channel to pass our scalable resources to worker threads.
	pools := make(chan string, poolCount)

	// Calculate the number of worker threads to initiate.
	maxConcurrency := s.config.ScalingConcurrency

	if poolCount < maxConcurrency {
		maxConcurrency = poolCount
	}

	logging.Debug("core/cluster_scaling: initiating %v concurrent scaling "+
		"threads to process %v worker pools", maxConcurrency, poolCount)

	// Initiate workers to implement worker pool scaling.
	for w := 1; w <= maxConcurrency; w++ {
		go s.workerPoolScaling(w, pools, nodeRegistry, jobRegistry, &wg)
	}

	// Add worker pools to the worker channel.
	for pool := range nodeRegistry.WorkerPools {
		pools <- pool
	}

	// Block on all worker pool scaling threads.
	wg.Wait()
}

// workerPoolScaling is a thread safe method for scaling an individual
// worker pool.
func (s *Server) workerPoolScaling(id int, pools <-chan string,
	nodeRegistry *structs.NodeRegistry, jobs *structs.JobScalingPolicies,
	wg *sync.WaitGroup) {

	// Setup references to clients for Nomad and Consul.
	nomadClient := s.config.NomadClient
	consulClient := s.config.ConsulClient

	for poolName := range pools {
		// Create anonymous function to simplify wg.Done()
		func() {
			defer wg.Done()
			logging.Debug("core/cluster_scaling: scaling thread %v evaluating scaling "+
				"for worker pool %v", id, poolName)

			// Obtain a read-only lock on the Node registry, grab a reference to
			// our worker pool object and release the lock.
			nodeRegistry.Lock.RLock()
			workerPool := nodeRegistry.WorkerPools[poolName]
			nodeRegistry.Lock.RUnlock()

			// Initialize a new disposable capacity object.
			poolCapacity := &structs.ClusterCapacity{}

			// Initialize a new scaling state object and set helper fields.
			workerPool.State = &structs.ScalingState{
				ResourceName: workerPool.Name,
				ResourceType: ClusterType,
				StatePath: s.config.ConsulKeyRoot + "/state/nodes/" +
					workerPool.Name,
			}

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
			if !FailsafeCheck(workerPool.State, s.config, workerPool.RetryThreshold, msg) {
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
			ok = checkPoolScalingThreshold(workerPool, s.config)
			if !ok {
				return
			}

			if poolCapacity.ScalingDirection != structs.ScalingDirectionNone {
				scaleMetric := poolCapacity.ScalingMetric

				logging.Info("core/cluster_scaling: worker pool %v requires a scaling "+
					"operation: (Direction: %v, Nodes: %v, Metric: %v, Capacity: %v, "+
					"Utilization: %v, Max Allowed: %v)", workerPool.Name,
					poolCapacity.ScalingDirection, len(workerPool.Nodes), scaleMetric.Type,
					scaleMetric.Capacity, scaleMetric.Utilization,
					poolCapacity.MaxAllowedUtilization)
			}

			if poolCapacity.ScalingDirection == structs.ScalingDirectionOut {
				// Initiate cluster scaling operation by calling the scaling provider.
				err = workerPool.ScalingProvider.Scale(workerPool, s.config, nodeRegistry)
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
				nodeID, nodeIP := nomadClient.LeastAllocatedNode(poolCapacity,
					workerPool.ProtectedNode)
				if nodeIP == "" || nodeID == "" {
					logging.Error("core/cluster_scaling: unable to identify the least "+
						"allocated node in worker pool %v", workerPool.Name)
					return
				}

				logging.Info("core/cluster_scaling: identified node %v as the least "+
					"allocated node in worker pool %v", nodeID, workerPool.Name)

				// Register the least allocated node as eligible for scaling actions.
				workerPool.State.EligibleNodes = append(workerPool.State.EligibleNodes,
					nodeIP)

				// Place the least allocated noded in drain mode.
				logging.Info("core/cluster_scaling: placing node %v from worker pool %v "+
					"in drain mode", nodeID, workerPool.Name)

				if err = nomadClient.DrainNode(nodeID); err != nil {
					logging.Error("core/cluster_scaling: an error occurred while "+
						"attempting to place node %v from worker pool %v in drain mode: "+
						"%v", nodeID, workerPool.Name, err)

					metrics.IncrCounter([]string{"cluster", workerPool.Name, "scale_in",
						"failure"}, 1)
					return
				}

				// Initiate cluster scaling operation by calling the scaling provider.
				err := workerPool.ScalingProvider.Scale(workerPool, s.config, nodeRegistry)
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

			// Our metric counter to track successful cluster scaling activities.
			m := fmt.Sprintf("scale_%s", strings.ToLower(poolCapacity.ScalingDirection))
			metrics.IncrCounter([]string{"cluster", workerPool.Name, m, "success"}, 1)
		}()
	}
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

// checkCooldownThreshold checks to see if a scaling cooldown threshold has
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
		logging.Debug("core/cluster_scaling: cluster scaling cooldown threshold "+
			"has not been reached: %v, scaling operations for worker pool %v should "+
			"not be permitted", cooldown, workerPool.Name)
		return false
	}

	logging.Debug("core/cluster_scaling: cluster scaling cooldown threshold %v "+
		"has been reached, scaling operations for worker pool %v should be "+
		"permitted", cooldown, workerPool.Name)

	return true
}
