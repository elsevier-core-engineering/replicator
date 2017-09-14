package client

import (
	"github.com/dariubs/percent"
	"github.com/elsevier-core-engineering/replicator/logging"
	"github.com/elsevier-core-engineering/replicator/replicator/structs"
	nomad "github.com/hashicorp/nomad/api"
	nomadStructs "github.com/hashicorp/nomad/nomad/structs"
)

// EvaluatePoolScaling evaluates a worker pool capacity and utilization,
// and determines whether a scaling operation is required and safe to
// implement.
func (c *nomadClient) EvaluatePoolScaling(capacity *structs.ClusterCapacity,
	workerPool *structs.WorkerPool,
	jobs *structs.JobScalingPolicies) (scale bool, err error) {

	// Determine the total capacity of the worker pool.
	if err = c.calculatePoolCapacity(capacity, workerPool); err != nil {
		return scale, err
	}

	// Determine the consumed capacity of the worker pool.
	if err = c.calculatePoolConsumed(capacity, workerPool); err != nil {
		return scale, err
	}

	// Determine the amount of capacity we should reserve for scaling
	// overhead on the worker pool.
	if err = c.calculateScalingReserve(capacity, jobs, workerPool); err != nil {
		return scale, err
	}

	// Determine the scaling metric by computing the most heavily utilized
	// scalable resource on the worker pool.
	c.MostUtilizedResource(capacity)

	// Compute the maximum allowed utilization of the most-utilized resource in
	// the worker pool.
	capacity.MaxAllowedUtilization =
		MaxAllowedClusterUtilization(capacity, workerPool.FaultTolerance, false)

	// Determine if a scaling operation is required.
	if scale = clusterScalingRequired(capacity, workerPool); !scale {
		return scale, err
	}

	// Check if the scaling operation is safe to implement.
	if scale = c.ClusterScalingSafe(capacity, workerPool); !scale {
		logging.Debug("client/cluster_scaling: cluster scaling operation (%v) "+
			"for worker pool %v fails to pass the safety check ",
			capacity.ScalingDirection, workerPool.Name)
		return scale, err
	}

	logging.Debug("client/cluster_scaling: cluster scaling operation (%v) for "+
		"worker pool %v passes the safety check and should be permitted",
		capacity.ScalingDirection, workerPool.Name)

	return scale, nil
}

// clusterScalingRequired determines if cluster scaling is required for a
// worker pool.
func clusterScalingRequired(capacity *structs.ClusterCapacity,
	workerPool *structs.WorkerPool) (scale bool) {
	var poolUtilization, poolCapacity int

	// Set the pool utilization and capacity values based on the prioritized
	// scaling metric.
	switch capacity.ScalingMetric {
	case ScalingMetricProcessor:
		poolCapacity = capacity.TotalCapacity.CPUMHz
		poolUtilization = capacity.UsedCapacity.CPUMHz
	case ScalingMetricMemory:
		poolCapacity = capacity.TotalCapacity.MemoryMB
		poolUtilization = capacity.UsedCapacity.MemoryMB
	default:
		poolCapacity = capacity.TotalCapacity.CPUMHz
		poolUtilization = capacity.UsedCapacity.CPUMHz
	}

	logging.Debug("client/cluster_scaling: computing scaling requirements for "+
		"worker pool %v: (Min: %v, Max: %v, Fault Tolerance: %v, Current: %v)",
		workerPool.Name, workerPool.Min, workerPool.Max, workerPool.FaultTolerance,
		len(workerPool.Nodes))

	// If the worker pool utilization is below the computed maximum threshold,
	// set the scaling direction inward.
	if (poolUtilization < capacity.MaxAllowedUtilization) ||
		(capacity.ScalingMetric == ScalingMetricNone) {

		capacity.ScalingDirection = ScalingDirectionIn
	}

	// If the worker pool utilization is above or equal to the computed maximum
	// threshold, check to see if we should scale the cluster out.
	if (poolUtilization >= capacity.MaxAllowedUtilization) &&
		(capacity.ScalingMetric != ScalingDirectionNone) {

		capacity.ScalingDirection = ScalingDirectionOut
	}

	logging.Debug("client/cluster_scaling: scaling requirements for worker pool "+
		"%v: (Metric: %v, Direction: %v, Capacity: %v, Utilization: %v, Max "+
		"Allowed: %v)", workerPool.Name, capacity.ScalingMetric,
		capacity.ScalingDirection, poolCapacity, poolUtilization,
		capacity.MaxAllowedUtilization)

	if capacity.ScalingDirection != ScalingDirectionNone {
		return true
	}

	return
}

// CalculatePoolCapacity computes the total capacity of a given worker pool.
func (c *nomadClient) calculatePoolCapacity(capacity *structs.ClusterCapacity,
	workerPool *structs.WorkerPool) (err error) {

	for _, node := range workerPool.Nodes {
		capacity.NodeCount++
		capacity.NodeList = append(capacity.NodeList, node.ID)

		capacity.TotalCapacity.CPUMHz += *node.Resources.CPU
		capacity.TotalCapacity.MemoryMB += *node.Resources.MemoryMB
		capacity.TotalCapacity.DiskMB += *node.Resources.DiskMB
	}

	return
}

// CalculatePoolConsumed computes the total consumed resources for a given
// worker pool and tracks the resources consumed by each worker node.
func (c *nomadClient) calculatePoolConsumed(capacity *structs.ClusterCapacity,
	workerPool *structs.WorkerPool) (err error) {

	q := &nomad.QueryOptions{}

	for node := range workerPool.Nodes {
		allocations, _, err := c.nomad.Nodes().Allocations(node, q)
		if err != nil {
			return err
		}

		// Create a new node allocation object.
		nodeInfo := &structs.NodeAllocation{
			NodeID:       node,
			UsedCapacity: structs.AllocationResources{},
		}

		for _, nodeAlloc := range allocations {
			if (nodeAlloc.ClientStatus == nomadStructs.TaskStateRunning) &&
				(nodeAlloc.DesiredStatus == nomadStructs.AllocDesiredStatusRun) {

				// Add the consumed resources to total worker pool consumption.
				capacity.UsedCapacity.CPUMHz += *nodeAlloc.Resources.CPU
				capacity.UsedCapacity.MemoryMB += *nodeAlloc.Resources.MemoryMB
				capacity.UsedCapacity.DiskMB += *nodeAlloc.Resources.DiskMB

				// Track the resources consumed by this worker node.
				nodeInfo.UsedCapacity.CPUMHz += *nodeAlloc.Resources.CPU
				nodeInfo.UsedCapacity.MemoryMB += *nodeAlloc.Resources.MemoryMB
				nodeInfo.UsedCapacity.DiskMB += *nodeAlloc.Resources.DiskMB

			}
		}

		// Add the node allocation record to the cluster status object.
		capacity.NodeAllocations = append(capacity.NodeAllocations, nodeInfo)
	}

	// Determine the percentage of overall cluster resources consumed and
	// calculate the amount of those resources consumed by the node.
	CalculateUsage(capacity)

	return
}

// calculateScalingReserve computes the total capacity required to increment
// all scalable jobs running on the worker pool by one. This capacity is
// held in reserve for future scaling overhead.
func (c *nomadClient) calculateScalingReserve(capacity *structs.ClusterCapacity,
	jobs *structs.JobScalingPolicies, workerPool *structs.WorkerPool) error {

	// Get detailed information about each job.
	for jobName := range jobs.Policies {
		// Determine if the job has a valid allocation on our worker pool.
		if ok := c.checkJobPlacement(jobName, workerPool); !ok {
			continue
		}

		job, _, err := c.nomad.Jobs().Info(jobName, &nomad.QueryOptions{})
		if err != nil {
			return err
		}

		// Iterate over groups and tasks to compute consumed capacity.
		for _, taskGroup := range job.TaskGroups {
			for _, task := range taskGroup.Tasks {
				capacity.TaskAllocation.CPUMHz += *task.Resources.CPU
				capacity.TaskAllocation.MemoryMB += *task.Resources.MemoryMB
				capacity.TaskAllocation.DiskMB += *task.Resources.DiskMB
			}
		}

	}

	return nil
}

// checkJobPlacement checks to see if a job is running on a specific worker
// pool.
func (c *nomadClient) checkJobPlacement(job string,
	workerPool *structs.WorkerPool) bool {

	allocs, _, err := c.nomad.Jobs().Allocations(job, false, &nomad.QueryOptions{})
	if err != nil {
		logging.Error("client/nomad: an error occurred while attempting to check "+
			"if job %v is running on worker pool %v: %v", job, workerPool.Name, err)
		return false
	}

	// Determine if any running allocations for the job have been placed on
	// a node in the worker pool.
	for _, alloc := range allocs {
		if !(alloc.DesiredStatus == nomadStructs.AllocDesiredStatusRun &&
			alloc.ClientStatus == nomadStructs.AllocClientStatusRunning) {
			continue
		}

		if _, ok := workerPool.Nodes[alloc.NodeID]; ok {
			return true
		}
	}

	return false
}

// ClusterScalingSafe determines if a cluster scaling operation can be safely
// executed.
func (c *nomadClient) ClusterScalingSafe(capacity *structs.ClusterCapacity,
	workerPool *structs.WorkerPool) (safe bool) {

	var poolUsedCapacity int

	switch capacity.ScalingMetric {
	case ScalingMetricProcessor:
		poolUsedCapacity = capacity.UsedCapacity.CPUMHz
	case ScalingMetricMemory:
		poolUsedCapacity = capacity.UsedCapacity.MemoryMB
	}

	if capacity.ScalingDirection == ScalingDirectionIn {
		// Determine if removing a node would violate safety thresholds or
		// declared minimums.
		if (capacity.NodeCount <= 1) || ((capacity.NodeCount - 1) < workerPool.Min) {
			logging.Debug("client/cluster_scaling: cluster scale-in operation " +
				"would violate safety thresholds or declared minimums")
			return
		}

		// Compute the new maximum allowed utilization after simulating the removal
		// of a worker node from the pool.
		newMaxAllowedUtilization := MaxAllowedClusterUtilization(capacity,
			workerPool.FaultTolerance, true)

		// Compare utilization against new maximum allowed utilization, if
		// utilization would be 90% or greater, we will not permit the scale-in
		// operation.
		newClusterUtilization :=
			percent.PercentOf(poolUsedCapacity, newMaxAllowedUtilization)

		logging.Debug("client/cluster_scaling: max allowed cluster utilization "+
			"after simulated node removal: %v (percent utilized: %v)",
			newMaxAllowedUtilization, newClusterUtilization)

		// Evaluate utilization against new maximum allowed threshold and stop if
		// a violation is present.
		if (poolUsedCapacity >= newMaxAllowedUtilization) ||
			(newClusterUtilization >= scaleInCapacityThreshold) {

			logging.Debug("client/cluster_scaling: cluster scale-in operation " +
				"would violate or is too close to the maximum allowed cluster " +
				"utilization threshold")
			return
		}
	} else if capacity.ScalingDirection == ScalingDirectionOut {
		// If scaling out would violate the Replicator max count, fail the safety
		// check.
		if (capacity.NodeCount + 1) > workerPool.Max {
			logging.Debug("client/cluster_scaling: cluster scale-out operation " +
				"would violate the scaling maximum threshold")
			return
		}
	}

	return true
}

// CalculateUsage computes the percentage of overall worker pool capacity
// consumed and computes the amount of that capacity consumed by each node.
func CalculateUsage(clusterInfo *structs.ClusterCapacity) {
	// For each allocation resource, calculate the percentage of overall cluster
	// capacity consumed.
	clusterInfo.UsedCapacity.CPUPercent = percent.PercentOf(
		clusterInfo.UsedCapacity.CPUMHz,
		clusterInfo.TotalCapacity.CPUMHz)

	clusterInfo.UsedCapacity.DiskPercent = percent.PercentOf(
		clusterInfo.UsedCapacity.DiskMB,
		clusterInfo.TotalCapacity.DiskMB)

	clusterInfo.UsedCapacity.MemoryPercent = percent.PercentOf(
		clusterInfo.UsedCapacity.MemoryMB,
		clusterInfo.TotalCapacity.MemoryMB)

	// Determine the amount of consumed resources consumed by each worker node.
	for _, nodeUsage := range clusterInfo.NodeAllocations {
		nodeUsage.UsedCapacity.CPUPercent = percent.PercentOf(
			nodeUsage.UsedCapacity.CPUMHz, clusterInfo.UsedCapacity.CPUMHz)
		nodeUsage.UsedCapacity.DiskPercent = percent.PercentOf(
			nodeUsage.UsedCapacity.DiskMB, clusterInfo.UsedCapacity.DiskMB)
		nodeUsage.UsedCapacity.MemoryPercent = percent.PercentOf(
			nodeUsage.UsedCapacity.MemoryMB, clusterInfo.UsedCapacity.MemoryMB)
	}
}
