package api

import (
	"math"
	"time"

	"github.com/dariubs/percent"
	"github.com/elsevier-core-engineering/replicator/helper"
	"github.com/elsevier-core-engineering/replicator/logging"
	"github.com/elsevier-core-engineering/replicator/replicator/structs"
	nomad "github.com/hashicorp/nomad/api"
)

// Scaling metric types indicate the most-utilized resource across the cluster. When evaluating
// scaling decisions, the most-utilized resource will be prioritized.
const (
	ScalingMetricNone      = "None" // All supported allocation resources are unutilized.
	ScalingMetricDisk      = "Disk"
	ScalingMetricMemory    = "Memory"
	ScalingMetricProcessor = "CPU"
)

// Scaling direction types indicate the allowed scaling actions.
const (
	ScalingDirectionOut  = "Out"
	ScalingDirectionIn   = "In"
	ScalingDirectionNone = "None"
)

const scaleInCapacityThreshold = 90.0
const bytesPerMegabyte = 1024000

// Provides a wrapper to the Nomad API package.
type nomadClient struct {
	nomad *nomad.Client
}

// NewNomadClient is used to create a new client to interact with Nomad. The
// client implements the NomadClient interface.
func NewNomadClient(addr string) (structs.NomadClient, error) {
	config := nomad.DefaultConfig()
	config.Address = addr
	c, err := nomad.NewClient(config)
	if err != nil {
		return nil, err
	}

	return &nomadClient{nomad: c}, nil
}

// EvaluateClusterCapacity determines if a cluster scaling operation is required.
func (c *nomadClient) EvaluateClusterCapacity(capacity *structs.ClusterAllocation, config *structs.Config) (scalingRequired bool, err error) {
	var clusterUtilization, clusterCapacity int

	// Determine total cluster capacity.
	if err = c.ClusterAllocationCapacity(capacity); err != nil {
		return
	}

	// Determine total consumed cluster capacity.
	if err = c.ClusterAssignedAllocation(capacity); err != nil {
		return
	}

	// Determine reserved scaling capacity requirements for all running jobs.
	if err = c.TaskAllocationTotals(capacity); err != nil {
		return
	}

	// Determine most-utilized resource across cluster to identify scaling metric.
	c.MostUtilizedResource(capacity)

	// Determine the maximum allowed utilization of the cluster most-utilized cluster resource.
	// This value is calculated after considering job scaling overhead and node fault-tolerance.
	capacity.MaxAllowedUtilization =
		MaxAllowedClusterUtilization(capacity, config.ClusterScaling.NodeFaultTolerance, false)

	// Use the correct resource utilization value to compare against max allowed utilization.
	switch capacity.ScalingMetric {
	case ScalingMetricProcessor:
		clusterUtilization = capacity.UsedCapacity.CPUMHz
		clusterCapacity = capacity.TotalCapacity.CPUMHz
	case ScalingMetricMemory:
		clusterUtilization = capacity.UsedCapacity.MemoryMB
		clusterCapacity = capacity.TotalCapacity.MemoryMB
	}

	logging.Debug("api/nomad: Cluster Node Count (Min: %v, Max: %v, Fault Tolerance: %v, Current: %v)",
		config.ClusterScaling.MinSize, config.ClusterScaling.MaxSize, config.ClusterScaling.NodeFaultTolerance, capacity.NodeCount)

	// If current utilization is less than max allowed, check to see if we can
	// and should scale the cluster in.
	if (clusterUtilization < capacity.MaxAllowedUtilization) || (capacity.ScalingMetric == ScalingMetricNone) {
		capacity.ScalingDirection = ScalingDirectionIn

		logging.Debug("api/nomad: Cluster Scaling (Metric: %v, Direction: %v, Capacity: %v, Utilization: %v, Max Allowed: %v)",
			capacity.ScalingMetric, capacity.ScalingDirection, clusterCapacity, clusterUtilization, capacity.MaxAllowedUtilization)

		if !c.CheckClusterScalingSafety(capacity, config, ScalingDirectionIn) {
			logging.Debug("api/nomad: cluster scaling operation (scale-in) fails to pass the safety check")
			return
		}
		logging.Debug("api/nomad: cluster scaling operation (scale-in) passes the safety check and will be permitted")
	}

	// If current utilization is greater than max allowed, check to see if we can
	// and should scale the cluster out.
	if (clusterUtilization >= capacity.MaxAllowedUtilization) && (capacity.ScalingMetric != ScalingDirectionNone) {
		capacity.ScalingDirection = ScalingDirectionOut

		logging.Debug("api/nomad: Cluster Scaling (Metric: %v, Direction: %v, Capacity: %v, Utilization: %v, Max Allowed: %v)",
			capacity.ScalingMetric, capacity.ScalingDirection, clusterCapacity, clusterUtilization, capacity.MaxAllowedUtilization)

		if !c.CheckClusterScalingSafety(capacity, config, ScalingDirectionOut) {
			logging.Debug("api/nomad: cluster scaling operation (scale-out) fails to pass the safety check")
			return
		}

		logging.Debug("api/nomad: cluster scaling operation (scale-out) passes the safety check and will be permitted")
	}

	return true, nil
}

// CheckClusterScalingSafety determines if a cluster scaling operation can be safely executed.
func (c *nomadClient) CheckClusterScalingSafety(capacity *structs.ClusterAllocation, config *structs.Config, scaleDirection string) (safe bool) {
	var clusterUsedCapacity int

	switch capacity.ScalingMetric {
	case ScalingMetricProcessor:
		clusterUsedCapacity = capacity.UsedCapacity.CPUMHz
	case ScalingMetricMemory:
		clusterUsedCapacity = capacity.UsedCapacity.MemoryMB
	}

	if scaleDirection == ScalingDirectionIn {
		// Determine if removing a node would violate safety thresholds or declared minimums
		if (capacity.NodeCount <= 1) || ((capacity.NodeCount - 1) < config.ClusterScaling.MinSize) {
			logging.Debug("api/nomad: cluster scale-in operation would violate safety thresholds or declared minimums")
			return
		}

		// Calculate the new maximum allowed cluster utilization if we were to remove a node.
		newMaxAllowedUtilization := MaxAllowedClusterUtilization(capacity,
			config.ClusterScaling.NodeFaultTolerance, true)

		// Calculate utilization against new maximum allowed utilization, if utilization would be 90% or greater,
		// we will not permit the scale-in operation.
		newClusterUtilization := percent.PercentOf(clusterUsedCapacity, newMaxAllowedUtilization)

		logging.Debug("api/nomad: max allowed cluster utilization after simulated node removal: %v (percent utilized: %v)",
			newMaxAllowedUtilization, newClusterUtilization)

		// Evaluate utilization against new maximum allowed threshold and stop if a violation is present.
		if (clusterUsedCapacity >= newMaxAllowedUtilization) || (newClusterUtilization >= scaleInCapacityThreshold) {
			logging.Debug("api/nomad: cluster scale-in operation would violate or is too close to the maximum allowed cluster utilization threshold")
			return
		}
	} else if scaleDirection == ScalingDirectionOut {
		if (capacity.NodeCount + 1) > config.ClusterScaling.MaxSize {
			logging.Debug("api/nomad: cluster scale-out operation would violate declared maximum threshold")
			return
		}
	}

	// Determine if performing a scaling operation would violate the scaling cooldown period.
	if err := CheckClusterScalingTimeThreshold(config.ClusterScaling.CoolDown,
		config.ClusterScaling.AutoscalingGroup, NewAWSAsgService(config.Region)); err != nil {
		logging.Debug("api/nomad: %v", err)
		return
	}

	return true
}

// ClusterAllocationCapacity calculates the total cluster capacity and determines the
// number of available worker nodes.
func (c *nomadClient) ClusterAllocationCapacity(capacity *structs.ClusterAllocation) (err error) {
	// Retrieve a list of all worker nodes within the cluster.
	nodes, _, err := c.nomad.Nodes().List(&nomad.QueryOptions{})
	if err != nil {
		return err
	}

	// Get detailed information about each worker node, if the node is in a known-good
	// state, increment the node count, add the node to the node list and add its
	// resources to the overall cluster capacity.
	for _, node := range nodes {
		resp, _, err := c.nomad.Nodes().Info(node.ID, &nomad.QueryOptions{})
		if err != nil {
			return err
		}

		if (resp.Status == "ready") && (resp.Drain != true) {
			capacity.NodeCount++
			capacity.NodeList = append(capacity.NodeList, node.ID)
			capacity.TotalCapacity.CPUMHz += *resp.Resources.CPU
			capacity.TotalCapacity.MemoryMB += *resp.Resources.MemoryMB
			capacity.TotalCapacity.DiskMB += *resp.Resources.DiskMB
		}
	}

	return nil
}

// ClusterAssignedAllocation calculates the total consumed resources across the cluster
// and the amount of resources consumed by each worker node.
func (c *nomadClient) ClusterAssignedAllocation(clusterInfo *structs.ClusterAllocation) (err error) {
	for _, node := range clusterInfo.NodeList {
		allocations, _, err := c.nomad.Nodes().Allocations(node, &nomad.QueryOptions{})
		if err != nil {
			return err
		}

		// Instantiate a new object to track the resource consumption of the worker node.
		nodeInfo := &structs.NodeAllocation{
			NodeID:       node,
			UsedCapacity: structs.AllocationResources{},
		}

		for _, nodeAlloc := range allocations {
			if (nodeAlloc.ClientStatus == "running") && (nodeAlloc.DesiredStatus == "run") {

				// Add the consumed resources to the overall cluster consumed resource values.
				clusterInfo.UsedCapacity.CPUMHz += *nodeAlloc.Resources.CPU
				clusterInfo.UsedCapacity.MemoryMB += *nodeAlloc.Resources.MemoryMB
				clusterInfo.UsedCapacity.DiskMB += *nodeAlloc.Resources.DiskMB

				// Add the consumed resources to the node specific allocation object.
				nodeInfo.UsedCapacity.CPUMHz += *nodeAlloc.Resources.CPU
				nodeInfo.UsedCapacity.MemoryMB += *nodeAlloc.Resources.MemoryMB
				nodeInfo.UsedCapacity.DiskMB += *nodeAlloc.Resources.DiskMB
			}
		}

		// Add the node allocation record to the cluster status object.
		clusterInfo.NodeAllocations = append(clusterInfo.NodeAllocations, nodeInfo)
	}

	// Determine the percentage of overall cluster resources consumed and calculate
	// the amount of those resources consumed by the node.
	CalculateUsage(clusterInfo)

	return
}

// CalculateUsage determines the percentage of overall cluster resources consumed and
// calculates the amount of those resources consumed by each worker node.
func CalculateUsage(clusterInfo *structs.ClusterAllocation) {
	// For each allocation resource, calculate the percentage of overall cluster capacity
	// consumed.
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
		nodeUsage.UsedCapacity.CPUPercent = percent.PercentOf(nodeUsage.UsedCapacity.CPUMHz,
			clusterInfo.UsedCapacity.CPUMHz)
		nodeUsage.UsedCapacity.DiskPercent = percent.PercentOf(nodeUsage.UsedCapacity.DiskMB,
			clusterInfo.UsedCapacity.DiskMB)
		nodeUsage.UsedCapacity.MemoryPercent = percent.PercentOf(nodeUsage.UsedCapacity.MemoryMB,
			clusterInfo.UsedCapacity.MemoryMB)
	}
}

// LeaderCheck determines if the node running the daemon is the gossip pool leader.
func (c *nomadClient) LeaderCheck() bool {
	haveLeadership := false

	leader, err := c.nomad.Status().Leader()
	if (err != nil) || (len(leader) == 0) {
		logging.Error("api/nomad: failed to identify cluster leader")
	}

	self, err := c.nomad.Agent().Self()
	if err != nil {
		logging.Error("api/nomad: unable to retrieve local agent information")
	} else {

		if helper.FindIP(leader) == self.Member.Addr {
			haveLeadership = true
		}
	}

	return haveLeadership
}

// TaskAllocation determines the total allocation requirements of a single instance (count=1)
// of all running jobs across the cluster. This is used to practively ensure the cluster
// has sufficient available capacity to scale each task by +1 if an increase in capacity
// is required.
func (c *nomadClient) TaskAllocationTotals(capacityUsed *structs.ClusterAllocation) error {
	// TODO (e.westfall): Allow behavior to be configured; restrict this check to only jobs with a
	// scaling policy present.

	// Get all jobs across the cluster.
	jobs, _, err := c.nomad.Jobs().List(&nomad.QueryOptions{})
	if err != nil {
		return err
	}

	// Get detailed information about each job.
	for _, job := range jobs {
		resp, _, err := c.nomad.Jobs().Info(job.ID, &nomad.QueryOptions{})
		if err != nil {
			return err
		}

		// A job can contain multiple task groups which can themselves contain multiple tasks;
		// therefore we must iterate fully. The API only returns jobs in a running state.
		for _, taskG := range resp.TaskGroups {
			for _, task := range taskG.Tasks {
				capacityUsed.TaskAllocation.CPUMHz += *task.Resources.CPU
				capacityUsed.TaskAllocation.MemoryMB += *task.Resources.MemoryMB
				capacityUsed.TaskAllocation.DiskMB += *task.Resources.DiskMB
			}
		}
	}

	return nil
}

// MostUtilizedResource calculates the resource that is most-utilized across the cluster.
// This is used to determine the resource that should be prioritized when making scaling
// decisions like determining the least-allocated worker node.
//
// If all resources are completely unutilized, the scaling metric will be set to `None`
// and the daemon will take no actions.
func (c *nomadClient) MostUtilizedResource(alloc *structs.ClusterAllocation) {
	// Determine the resource that is consuming the greatest percentage of its overall cluster
	// capacity.
	max := (helper.Max(alloc.UsedCapacity.CPUPercent, alloc.UsedCapacity.MemoryPercent,
		alloc.UsedCapacity.DiskPercent))

	// Set the compute cluster scaling metric to the most-utilized resource.
	switch max {
	case 0:
		alloc.ScalingMetric = ScalingMetricNone
	case alloc.UsedCapacity.CPUPercent:
		alloc.ScalingMetric = ScalingMetricProcessor
	case alloc.UsedCapacity.DiskPercent:
		alloc.ScalingMetric = ScalingMetricDisk
	case alloc.UsedCapacity.MemoryPercent:
		alloc.ScalingMetric = ScalingMetricMemory
	}
}

// MostUtilizedGroupResource determines whether CPU or Mem are the most utilized
// resource of a Group.
func (c *nomadClient) MostUtilizedGroupResource(gsp *structs.GroupScalingPolicy) {
	max := (helper.Max(gsp.Tasks.Resources.CPUPercent,
		gsp.Tasks.Resources.MemoryPercent))

	switch max {
	case gsp.Tasks.Resources.CPUPercent:
		gsp.ScalingMetric = ScalingMetricProcessor
	case gsp.Tasks.Resources.MemoryPercent:
		gsp.ScalingMetric = ScalingMetricMemory
	}
}

// LeastAllocatedNode determines which worker node is consuming the lowest percentage of the
// resource identified as the most-utilized resource across the cluster. Since Nomad follows
// a bin-packing approach, when we need to remove a worker node in response to a scale-in
// activity, we want to identify the least-allocated node and target it for removal.
func (c *nomadClient) LeastAllocatedNode(clusterInfo *structs.ClusterAllocation) (nodeID, nodeIP string) {
	var lowestAllocation float64

	for _, nodeAlloc := range clusterInfo.NodeAllocations {
		switch clusterInfo.ScalingMetric {
		case ScalingMetricProcessor:
			if (lowestAllocation == 0) || (nodeAlloc.UsedCapacity.CPUPercent < lowestAllocation) {
				nodeID = nodeAlloc.NodeID
				lowestAllocation = nodeAlloc.UsedCapacity.CPUPercent
			}
		case ScalingMetricMemory:
			if (lowestAllocation == 0) || (nodeAlloc.UsedCapacity.MemoryPercent < lowestAllocation) {
				nodeID = nodeAlloc.NodeID
				lowestAllocation = nodeAlloc.UsedCapacity.MemoryPercent
			}
		case ScalingMetricNone:
			nodeID = nodeAlloc.NodeID
		}
	}

	// In order to perform downscaling of the cluster we need to have access
	// to the nodes IP address so  the AWS instance-id can be infered.
	resp, _, err := c.nomad.Nodes().Info(nodeID, &nomad.QueryOptions{})
	if err != nil {
		logging.Error("api/nomad: unable to determine nomad node IP address: %v", err)
	}
	nodeIP = resp.Attributes["unique.network.ip-address"]

	return
}

// DrainNode toggles the drain mode of a worker node. When enabled, no further allocations
// will be assigned and existing allocations will be migrated.
func (c *nomadClient) DrainNode(nodeID string) (err error) {
	// Initiate allocation draining for specified node.
	_, err = c.nomad.Nodes().ToggleDrain(nodeID, true, &nomad.WriteOptions{})
	if err != nil {
		return err
	}

	// Validate node has been placed in drain mode; fail fast if the node
	// failed to enter drain mode.
	resp, _, err := c.nomad.Nodes().Info(nodeID, &nomad.QueryOptions{})
	if (err != nil) || (resp.Drain != true) {
		return err
	}
	logging.Info("api/nomad: node %v has been placed in drain mode\n", nodeID)

	// Setup a ticker to poll the node allocations and report when all existing
	// allocations have been migrated to other worker nodes.
	ticker := time.NewTicker(time.Millisecond * 500)
	timeout := time.Tick(time.Minute * 3)

	for {
		select {
		case <-timeout:
			logging.Error("api/nomad: timeout %v reached while waiting for existing allocations to be migrated from node %v",
				timeout, nodeID)
			return nil
		case <-ticker.C:
			activeAllocations := 0

			// Get allocations assigned to the specified node.
			allocations, _, err := c.nomad.Nodes().Allocations(nodeID, &nomad.QueryOptions{})
			if err != nil {
				return err
			}

			// Iterate over allocations, if any are running or pending, increment the active
			// allocations counter.
			for _, nodeAlloc := range allocations {
				if (nodeAlloc.ClientStatus == "running") || (nodeAlloc.ClientStatus == "pending") {
					activeAllocations++
				}
			}

			if activeAllocations == 0 {
				logging.Info("api/nomad: node %v has no active allocations", nodeID)
				return nil
			}

			logging.Info("api/nomad: node %v has %v active allocations, pausing and will re-poll allocations", nodeID, activeAllocations)
		}
	}
}

// JobScale takes a Scaling Policy and then attempts to scale the desired job
// to the appropriate level whilst ensuring the event will not excede any job
// thresholds set.
func (c *nomadClient) JobScale(scalingDoc *structs.JobScalingPolicy) {

	// In order to scale the job, we need information on the current status of the
	// running job from Nomad.
	jobResp, _, err := c.nomad.Jobs().Info(scalingDoc.JobName, &nomad.QueryOptions{})

	if err != nil {
		logging.Info("api/nomad: unable to determine job info of %v", scalingDoc.JobName)
		return
	}

	// Use the current task count in order to determine whether or not a
	// scaling event will violate the min/max job policy and exit the function if
	// it would.
	for _, group := range scalingDoc.GroupScalingPolicies {

		if group.Scaling.ScaleDirection != "None" {

			for i, taskGroup := range jobResp.TaskGroups {
				if group.Scaling.ScaleDirection == "Out" && *taskGroup.Count >= group.Scaling.Max ||
					group.Scaling.ScaleDirection == "In" && *taskGroup.Count <= group.Scaling.Min {
					logging.Debug("api/nomad: scale %v not permitted due to constraints on job \"%v\" and group \"%v\"",
						group.Scaling.ScaleDirection, *jobResp.ID, group.GroupName)
					return
				}

				logging.Info("api/nomad: scaling action (%v) will now be initiated against job \"%v\" and group \"%v\"",
					group.Scaling.ScaleDirection, scalingDoc.JobName, group.GroupName)
				// Depending on the scaling direction decrement/incrament the count;
				// currently replicator only supports addition/subtraction of 1.
				if *taskGroup.Name == group.GroupName && group.Scaling.ScaleDirection == "Out" {
					*jobResp.TaskGroups[i].Count++
				}

				if *taskGroup.Name == group.GroupName && group.Scaling.ScaleDirection == "In" {
					*jobResp.TaskGroups[i].Count--
				}
			}
		}
	}
	// Nomad 0.5.5 introduced a Jobs.Validate endpoint within the API package
	// which validates the job syntax before submition.
	_, _, err = c.nomad.Jobs().Validate(jobResp, &nomad.WriteOptions{})
	if err != nil {
		return
	}

	// Submit the job to the Register API endpoint with the altered count number
	// and check that no error is returned.
	_, _, err = c.nomad.Jobs().Register(jobResp, &nomad.WriteOptions{})
	if err != nil {
		return
	}

	logging.Info("api/nomad: scaling action successfully taken against job \"%v\"", *jobResp.ID)
	return
}

// GetTaskGroupResources finds the defined resource requirements for a
// given Job.
func (c *nomadClient) GetTaskGroupResources(jobName string, groupPolicy *structs.GroupScalingPolicy) {
	jobs, _, err := c.nomad.Jobs().Info(jobName, &nomad.QueryOptions{})
	if err != nil {
		logging.Error("api/nomad: failed to retrieve job details for job %v: %v\n", jobName, err)
	}

	for _, group := range jobs.TaskGroups {
		for _, task := range group.Tasks {
			groupPolicy.Tasks.Resources.CPUMHz += *task.Resources.CPU
			groupPolicy.Tasks.Resources.MemoryMB += *task.Resources.MemoryMB
		}
	}
}

// EvaluateJobScaling identifies Nomad allocations representative of a Job group
// and compares the consumed resource percentages against the scaling policy to
// determine whether a scaling event is required.
func (c *nomadClient) EvaluateJobScaling(jobs []*structs.JobScalingPolicy) {
	for _, policy := range jobs {
		for _, gsp := range policy.GroupScalingPolicies {
			logging.Debug("api/nomad: evaluating scaling for job %v", policy.JobName)
			c.GetTaskGroupResources(policy.JobName, gsp)

			allocs, _, err := c.nomad.Jobs().Allocations(policy.JobName, false, &nomad.QueryOptions{})
			if err != nil {
				logging.Error("api/nomad: failed to retrieve allocations for job %v: %v\n", policy.JobName, err)
			}

			c.GetJobAllocations(allocs, gsp)
			c.MostUtilizedGroupResource(gsp)

			switch gsp.ScalingMetric {
			case ScalingMetricProcessor:
				if gsp.Tasks.Resources.CPUPercent > gsp.Scaling.ScaleOut.CPU {
					gsp.Scaling.ScaleDirection = ScalingDirectionOut
				}
			case ScalingMetricMemory:
				if gsp.Tasks.Resources.MemoryPercent > gsp.Scaling.ScaleOut.MEM {
					gsp.Scaling.ScaleDirection = ScalingDirectionOut
				}
			}

			if (gsp.Tasks.Resources.CPUPercent < gsp.Scaling.ScaleIn.CPU) &&
				(gsp.Tasks.Resources.MemoryPercent < gsp.Scaling.ScaleIn.MEM) {
				gsp.Scaling.ScaleDirection = ScalingDirectionIn
			}
		}
	}
}

// GetJobAllocations identifies all allocations for an active job.
func (c *nomadClient) GetJobAllocations(allocs []*nomad.AllocationListStub, gsp *structs.GroupScalingPolicy) {
	for _, allocationStub := range allocs {
		if (allocationStub.ClientStatus == "running") && (allocationStub.DesiredStatus == "run") {
			if alloc, _, err := c.nomad.Allocations().Info(allocationStub.ID, &nomad.QueryOptions{}); err == nil && alloc != nil {
				c.GetAllocationStats(alloc, gsp)
			}
		}
	}
}

// GetAllocationStats discovers the resources consumed by a particular Nomad
// allocation.
func (c *nomadClient) GetAllocationStats(allocation *nomad.Allocation, scalingPolicy *structs.GroupScalingPolicy) {
	stats, err := c.nomad.Allocations().Stats(allocation, &nomad.QueryOptions{})
	if err != nil {
		logging.Error("api/nomad: failed to retrieve allocation statistics from client %v: %v\n", allocation.NodeID, err)
		return
	}

	cs := stats.ResourceUsage.CpuStats
	ms := stats.ResourceUsage.MemoryStats

	scalingPolicy.Tasks.Resources.CPUPercent = percent.PercentOf(int(math.Floor(cs.TotalTicks)),
		scalingPolicy.Tasks.Resources.CPUMHz)
	scalingPolicy.Tasks.Resources.MemoryPercent = percent.PercentOf(int((ms.RSS / bytesPerMegabyte)),
		scalingPolicy.Tasks.Resources.MemoryMB)
}

// IsJobRunning checks to see whether the specified jobID has any currently
// task groups on the cluster.
func (c *nomadClient) IsJobRunning(jobID string) bool {

	_, _, err := c.nomad.Jobs().Summary(jobID, &nomad.QueryOptions{})

	if err != nil {
		return false
	}

	return true
}

// MaxAllowedClusterUtilization calculates the maximum allowed cluster utilization after
// taking into consideration node fault-tolerance and scaling overhead.
func MaxAllowedClusterUtilization(capacity *structs.ClusterAllocation, nodeFaultTolerance int, scaleIn bool) (maxAllowedUtilization int) {
	var allocTotal, capacityTotal int
	var internalScalingMetric string

	// Use the cluster scaling metric when determining total cluster capacity
	// and task group scaling overhead.
	switch capacity.ScalingMetric {
	case ScalingMetricMemory:
		internalScalingMetric = ScalingMetricMemory
		allocTotal = capacity.TaskAllocation.MemoryMB
		capacityTotal = capacity.TotalCapacity.MemoryMB
	default:
		internalScalingMetric = ScalingMetricProcessor
		allocTotal = capacity.TaskAllocation.CPUMHz
		capacityTotal = capacity.TotalCapacity.CPUMHz
	}

	nodeAvgAlloc := capacityTotal / capacity.NodeCount
	if scaleIn {
		capacityTotal = capacityTotal - nodeAvgAlloc
	}

	logging.Debug("api/nomad: Cluster Capacity (CPU [MHz]: %v, Memory [MB]: %v)",
		capacity.TotalCapacity.CPUMHz, capacity.TotalCapacity.MemoryMB)
	logging.Debug("api/nomad: Cluster Utilization (Scaling Metric: %v, CPU [MHz]: %v, Memory [MB]: %v)",
		capacity.ScalingMetric, capacity.UsedCapacity.CPUMHz, capacity.UsedCapacity.MemoryMB)
	logging.Debug("api/nomad: Scaling Metric (Algorithm): %v, Average Node Capacity: %v, Job Scaling Overhead: %v",
		internalScalingMetric, nodeAvgAlloc, allocTotal)

	maxAllowedUtilization = ((capacityTotal - allocTotal) - (nodeAvgAlloc * nodeFaultTolerance))

	return
}
