package replicator

import (
	"fmt"
	"os"

	"github.com/d3sw/replicator/logging"
	"github.com/d3sw/replicator/replicator/structs"
)

func (r *Server) nodeProtectionCheck(nodeReg *structs.NodeRegistry) error {
	// Check our runtime environment to see if we have a Nomad allocation.
	allocID := os.Getenv("NOMAD_ALLOC_ID")

	// If we're not running as a Nomad job, return immediately.
	if len(allocID) == 0 {
		return nil
	}

	// Perform a reverse lookup to get the Nomad node hosting our allocation.
	host, err := r.config.NomadClient.NodeReverseLookup(allocID)
	if err != nil || len(host) == 0 {
		return fmt.Errorf("Replicator is running as a Nomad job but we are "+
			"unable to determine the node hosting our allocation %v: %v",
			allocID, err)
	}

	nodeReg.Lock.Lock()
	// Ask the node registry to give us the worker pool for the node hosting
	// our allocation.
	pool, ok := nodeReg.RegisteredNodes[host]
	if !ok {
		return fmt.Errorf("running as a Nomad job but unable to determine the "+
			"worker pool for our host node %v", host)
	}

	// Register the node as protected in the node registry.
	if nodeReg.WorkerPools[pool].ProtectedNode != host {
		logging.Debug("core/node_protection: registering node %v from worker "+
			"pool %v as protected", host, pool)
		nodeReg.WorkerPools[pool].ProtectedNode = host
	}
	nodeReg.Lock.Unlock()

	return nil
}
