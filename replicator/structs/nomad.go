package structs

import (
	nomad "github.com/hashicorp/nomad/api"
)

// Set of possible states for a node.
const (
	NodeStatusInit  = "initializing"
	NodeStatusReady = "ready"
	NodeStatusDown  = "down"
)

// Scaling direction types indicate the allowed scaling actions.
const (
	ScalingDirectionOut  = "Out"
	ScalingDirectionIn   = "In"
	ScalingDirectionNone = "None"
)

// NomadClient exposes all API methods needed to interact with the Nomad API,
// evaluate cluster capacity and allocations and make scaling decisions.
type NomadClient interface {
	ClusterScalingSafe(*ClusterCapacity, *WorkerPool) bool

	// DrainNode places a worker node in drain mode to stop future allocations
	// and migrate existing allocations to other worker nodes.
	DrainNode(string) error

	// EvaluatePoolScaling evaluates a worker pool capacity and utilization,
	// and determines whether a scaling operation is required and safe to
	// implement.
	EvaluatePoolScaling(*ClusterCapacity, *WorkerPool, *JobScalingPolicies) (bool, error)

	// EvaluateJobScaling compares the consumed resource percentages of a Job
	// group against its scaling policy to determine whether a scaling event is
	// required.
	EvaluateJobScaling(string, []*GroupScalingPolicy) error

	// GetAllocationStats discovers the resources consumed by a particular Nomad
	// allocation.
	GetAllocationStats(*nomad.Allocation, *GroupScalingPolicy)

	// GetJobAllocations identifies all allocations for an active job.
	GetJobAllocations([]*nomad.AllocationListStub, *GroupScalingPolicy)

	// IsJobInDeployment checks to see whether the supplied Nomad job is currently
	// in the process of a deployment.
	IsJobInDeployment(string) bool

	// JobGroupScale scales a particular job group, confirming that the action
	// completes successfully.
	JobGroupScale(string, *GroupScalingPolicy, *ScalingState)

	// JobWatcher is the main entry point into Replicators process of reading and
	// updating its JobScalingPolicies tracking.
	JobWatcher(*JobScalingPolicies)

	// LeastAllocatedNode determines which worker pool node is consuming the
	// least amount of the cluster's most-utilized resource.
	LeastAllocatedNode(*ClusterCapacity, *ScalingState) (string, string)

	// NodeReverseLookup provides a method to get the ID of the worker pool node
	// running a given allocation.
	NodeReverseLookup(string) (string, error)

	// NodeWatcher provides an automated mechanism to discover worker pools and
	// nodes and populate the node registry.
	NodeWatcher(*NodeRegistry)

	// MostUtilizedResource calculates which resource is most-utilized across the
	// cluster. The worst-case allocation resource is prioritized when making
	// scaling decisions.
	MostUtilizedResource(*ClusterCapacity)

	// VerifyNodeHealth evaluates whether a specified worker node is a healthy
	// member of the Nomad cluster.
	VerifyNodeHealth(string) bool
}

// ClusterCapacity is the central object used to track and evaluate cluster
// capacity, utilization and stores the data required to make scaling
// decisions. All data stored in this object is disposable and is generated
// during each evaluation.
type ClusterCapacity struct {
	// NodeCount is the number of worker nodes in a ready and non-draining state
	// across the cluster.
	NodeCount int

	// ScalingMetric indicates the most-utilized allocation resource across the
	// cluster. The most-utilized resource is prioritized when making scaling
	// decisions like identifying the least-allocated worker node.
	ScalingMetric string

	// MaxAllowedUtilization represents the max allowed cluster utilization after
	// considering node fault-tolerance and task group scaling overhead.
	MaxAllowedUtilization int

	// ClusterTotalAllocationCapacity is the total allocation capacity across
	// the cluster.
	TotalCapacity AllocationResources

	// ClusterUsedAllocationCapacity is the consumed allocation capacity across
	// the cluster.
	UsedCapacity AllocationResources

	// TaskAllocation represents the total allocation requirements of a single
	// instance (count 1) of all running jobs across the cluster. This is used to
	// practively ensure the cluster has sufficient available capacity to scale
	// each task by +1 if an increase in capacity is required.
	TaskAllocation AllocationResources

	// NodeList is a list of all worker nodes in a known good state.
	NodeList []string

	// NodeAllocations is a slice of node allocations.
	NodeAllocations []*NodeAllocation

	// ScalingDirection is the direction in/out of cluster scaling we require
	// after performning the proper evalutation.
	ScalingDirection string
}

// NodeAllocation describes the resource consumption of a specific worker node.
type NodeAllocation struct {
	// NodeID is the unique ID of the worker node.
	NodeID string

	// NodeIP is the private IP of the worker node.
	NodeIP string

	// UsedCapacity represents the percentage of total cluster resources consumed
	// by the worker node.
	UsedCapacity AllocationResources
}

// TaskAllocation describes the resource requirements defined in the job
// specification.
type TaskAllocation struct {
	// TaskName is the name given to the task within the job specficiation.
	TaskName string

	// Resources tracks the resource requirements defined in the job spec and the
	// real-time utilization of those resources.
	Resources AllocationResources
}

// AllocationResources represents the allocation resource utilization.
type AllocationResources struct {
	MemoryMB      int
	CPUMHz        int
	DiskMB        int
	MemoryPercent float64
	CPUPercent    float64
	DiskPercent   float64
}
