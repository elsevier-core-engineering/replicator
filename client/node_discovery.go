package client

import (
	"fmt"
	"time"

	"github.com/elsevier-core-engineering/replicator/cloudprovider"
	"github.com/elsevier-core-engineering/replicator/helper"
	"github.com/elsevier-core-engineering/replicator/logging"
	"github.com/elsevier-core-engineering/replicator/replicator/structs"
	nomad "github.com/hashicorp/nomad/api"
	nomadStructs "github.com/hashicorp/nomad/nomad/structs"
	"github.com/mitchellh/hashstructure"
	"github.com/mitchellh/mapstructure"
)

// NodeWathcher is the method Replicator uses to perform discovery of all
// worker pool nodes in the Nomad cluster.
func (c *nomadClient) NodeWatcher(nodeRegistry *structs.NodeRegistry) {
	q := &nomad.QueryOptions{WaitIndex: 1, AllowStale: true}

	for {
		nodes, meta, err := c.nomad.Nodes().List(q)
		if err != nil {
			logging.Error("client/node_discovery: failed to retrieve nodes from "+
				"the Nomad API: %v", err)

			// Sleep as we don't want to retry the API call as fast as Go possibly can.
			time.Sleep(20 * time.Second)
			continue
		}

		if meta.LastIndex <= nodeRegistry.LastChangeIndex {
			logging.Debug("client/node_discovery: blocking query timed out, " +
				"restarting node discovery watcher")
			continue
		}

		for _, node := range nodes {
			if node.ModifyIndex <= nodeRegistry.LastChangeIndex {
				continue
			}

			if node.Drain == true && node.Status == nomadStructs.NodeStatusReady {
				logging.Warning("client/node_discovery: node %v has been placed in "+
					"drain mode, initiating deregistration of the node", node.ID)

				Deregister(node.ID, nodeRegistry)
				continue
			}

			switch node.Status {

			// If the node is in a ready state, determine if scaling has been
			// enabled. If so, register the node, otherwise deregister the node.
			case structs.NodeStatusReady:
				// Retrieve detailed node information to obtain meta configuration
				// parameters from the client stanza of the agent.
				nodeRecord, _, err := c.nomad.Nodes().Info(node.ID, &nomad.QueryOptions{})
				if err != nil {
					logging.Error("client/node_discovery: an error occurred while "+
						"attempting to retrieve node configuration details: %v", err)
					continue
				}

				// Retrieve node details and process the scaling configuration.
				nodeConfig, err := ProcessNodeConfig(nodeRecord)
				if err != nil {
					logging.Debug("client/node_discovery: an error occurred while "+
						"attempting to process the node configuration: %v", err)

					// If the node has been previously observed and registered, it
					// should be deregistered.
					if _, ok := nodeRegistry.RegisteredNodes[node.ID]; ok {
						Deregister(node.ID, nodeRegistry)
					}

					continue
				}

				if err := Register(nodeRecord, nodeConfig, nodeRegistry); err != nil {
					logging.Error("client/node_discovery: an error occurred while "+
						"attempting to register node %v: %v", nodeRecord.ID, err)
				}

				if !nodeConfig.ScalingEnabled {
					logging.Debug("client/node_discovery: scaling has been disabled "+
						"on node %v, initiating deregistration of the node", node.ID)
					Deregister(node.ID, nodeRegistry)
				}

			// If the node is down, deregister the node.
			case structs.NodeStatusDown:
				// BUG (e.westfall): We should not log like this as we will always
				// see nodes that are down until they are reaped. Consider checking
				// the meta index of each node.
				logging.Warning("client/node_discovery: node %v is down, initiating "+
					"deregistration of the node", node.ID)
				Deregister(node.ID, nodeRegistry)
			}
		}

		// Report discovered worker pools and their registered nodes.
		if updated := NodeRegistryUpdated(nodeRegistry); updated {
			nodeRegistry.Lock.Lock()
			for _, workerPool := range nodeRegistry.WorkerPools {
				nodes := make([]string, 0, len(workerPool.Nodes))
				for k := range workerPool.Nodes {
					nodes = append(nodes, k)
				}

				logging.Info("client/node_discovery: worker pool %v has %v healthy "+
					"nodes configured for scaling: %v", workerPool.Name,
					len(workerPool.Nodes), nodes)
			}
			nodeRegistry.Lock.Unlock()
		}

		// Update the last change indices to ensure our blocking query can
		// detect future node changes.
		nodeRegistry.Lock.Lock()
		nodeRegistry.LastChangeIndex = meta.LastIndex
		q.WaitIndex = meta.LastIndex
		nodeRegistry.Lock.Unlock()
	}
}

// ProcessNodeConfig retrieves detailed information about a node and processes
// configuration details. If the meta configuration parameters required for
// scaling and identification of the associated worker pool are successfully
// processed, they are returned.
func ProcessNodeConfig(node *nomad.Node) (pool *structs.WorkerPool, err error) {

	// Create a new worker pool record to hold processed meta
	// configuration parameters.
	result := structs.NewWorkerPool()

	// Required meta configuration keys.
	requiredKeys := []string{
		"replicator_enabled",
		"replicator_max",
		"replicator_min",
		"replicator_notification_uid",
		"replicator_provider",
		"replicator_region",
		"replicator_worker_pool",
	}

	// Parse meta configuration parameters and determine if any required
	// configuration keys are missing.
	missingKeys := helper.ParseMetaConfig(node.Meta, requiredKeys)

	// If none of the required keys are present, the node is not configured
	// for autoscaling and cannot be registered.
	if len(missingKeys) == len(requiredKeys) {
		err = fmt.Errorf("node %v does not contain a valid autoscaling "+
			"configuration", node.ID)
		return nil, err
	}

	// If some of the required keys are missing, the autoscaling configuration
	// is considered invalid and the node cannot be registered.
	if len(missingKeys) > 0 {
		err = fmt.Errorf("the autoscaling configuration for node %v is missing "+
			"required configuration parameters: %v", node.ID, missingKeys)
		return nil, err
	}

	// Setup configuration for our structure decoder.
	config := &mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		Result:           result,
	}

	// Create a new structure decoder.
	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return nil, err
	}

	// Decode the meta configuration parameters and build a worker pool
	// record.
	if err = decoder.Decode(node.Meta); err != nil {
		return nil, err
	}

	return result, nil
}

// Register is responsible for registering a newly discovered worker pool
// or registering a node with an previously discovered worker pool.
func Register(node *nomad.Node, workerPool *structs.WorkerPool,
	nodeRegistry *structs.NodeRegistry) error {

	// Decline to register the node if it is not in a ready state.
	if node.Status != structs.NodeStatusReady {
		return fmt.Errorf("an attempt to register node %v failed because the "+
			"node state was %v and must be %v", node.ID, node.Status,
			structs.NodeStatusReady)
	}

	// Decline to register the node if drain mode is enabled.
	if node.Drain {
		return fmt.Errorf("an attempt to register node %v failed because the "+
			"node is in drain mode", node.ID)
	}

	nodeRegistry.Lock.Lock()
	defer nodeRegistry.Lock.Unlock()

	// If the worker pool is already present in the registry, determine
	// if the node is already registered, otherwise, register the node.
	if existingPool, ok := nodeRegistry.WorkerPools[workerPool.Name]; ok {
		changed, err := helper.HasObjectChanged(existingPool, workerPool)
		if err != nil {
			logging.Error("client/node_discovery: unable to determine if the "+
				"worker pool configuration has been updated: %v", err)
		}

		// Update existing worker pool configuration.
		if changed {
			logging.Debug("client/node_discovery: worker pool configuration has " +
				"changed, updating.")
			existingPool.Max = workerPool.Max
			existingPool.Min = workerPool.Min
			existingPool.Region = workerPool.Region
			existingPool.Cooldown = workerPool.Cooldown
			existingPool.RetryThreshold = workerPool.RetryThreshold
			existingPool.FaultTolerance = workerPool.FaultTolerance
			existingPool.ScalingEnabled = workerPool.ScalingEnabled
			existingPool.NotificationUID = workerPool.NotificationUID
			existingPool.ScalingThreshold = workerPool.ScalingThreshold
		}

		// If the node is not already known to the worker pool, register it.
		if _, ok := existingPool.Nodes[node.ID]; !ok {
			if workerPool.ScalingEnabled {
				logging.Debug("client/node_discovery: registering node %v under "+
					"previously discovered worker pool %v", node.ID, workerPool.Name)

				// Add node registration record to track node discovery time.
				existingPool.NodeRegistrations[node.ID] = time.Now()

				// Register the node within the worker pool record.
				existingPool.Nodes[node.ID] = node

				// Register the node in the node registry.
				nodeRegistry.RegisteredNodes[node.ID] = workerPool.Name
			}
		}

		return nil
	}

	// Add node registration record to track node discovery time.
	workerPool.NodeRegistrations[node.ID] = time.Now()

	// Add the node to the worker pool.
	workerPool.Nodes[node.ID] = node

	// Register the appropriate scaling provider with the worker pool.
	scalingProvider, err := cloudprovider.NewScalingProvider(node.Meta)
	if err != nil {
		return fmt.Errorf("failed to initialize scaling provider for worker pool "+
			"%v: %v", workerPool.Name, err)
	}
	workerPool.ScalingProvider = scalingProvider

	logging.Debug("client/node_discovery: registering node %v with new worker "+
		"pool %v with registration timestamp %v", node.ID, workerPool.Name,
		workerPool.NodeRegistrations[node.ID])

	// Add an observed node record and register the node with the
	// worker pool.
	nodeRegistry.RegisteredNodes[node.ID] = workerPool.Name
	nodeRegistry.WorkerPools[workerPool.Name] = workerPool

	return nil
}

// Deregister is responsible for removing a node from a worker pool record.
// If after node deregistration, a worker pool has no remaining nodes, the
// worker pool is removed from the node registry.
func Deregister(node string, nodeRegistry *structs.NodeRegistry) (err error) {
	nodeRegistry.Lock.Lock()
	defer nodeRegistry.Lock.Unlock()

	// Return if there is no node registration to remove.
	pool, ok := nodeRegistry.RegisteredNodes[node]
	if !ok {
		return fmt.Errorf("node %v was not previously registered, no "+
			"deregistration is required", node)
	}

	logging.Debug("client/node_discovery: deregistring node %v from previously "+
		"discovered worker pool %v", node, pool)

	// Obtain a reference to the worker pool record.
	workerPool := nodeRegistry.WorkerPools[pool]

	// Remove the observed node record and deregister the node from the
	// worker pool.
	delete(workerPool.Nodes, node)
	delete(workerPool.NodeRegistrations, node)
	delete(nodeRegistry.RegisteredNodes, node)

	// If the worker pool has no registered nodes left, deregister the
	// worker pool.
	if len(workerPool.Nodes) <= 0 {
		logging.Warning("client/node_discovery: worker pool %v has no healthy "+
			"registered nodes, deregistering worker pool", workerPool.Name)
		delete(nodeRegistry.WorkerPools, workerPool.Name)
	}

	return
}

// NodeRegistryUpdated determines if the node registry has been updated
// and manages updating the node hash.
func NodeRegistryUpdated(nodeRegistry *structs.NodeRegistry) (updated bool) {
	nodeRegistry.Lock.Lock()
	defer nodeRegistry.Lock.Unlock()

	nodeHash, err := hashstructure.Hash(nodeRegistry.RegisteredNodes, nil)
	if err != nil {
		logging.Error("client/node_discovery: an error occurred while computing "+
			"the registered node hash: %v", err)
	}

	if nodeRegistry.RegisteredNodesHash != nodeHash {
		nodeRegistry.RegisteredNodesHash = nodeHash
		updated = true
	}

	return
}
