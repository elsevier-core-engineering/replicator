package structs

// ScalingProvider provides a standardized interface for implementing
// scaling support across different cloud providers or scaling technologies.
type ScalingProvider interface {
	// SafetyCheck should implement provider specific checks that
	// should be run before Replicator initiates a scaling operation.
	SafetyCheck(*WorkerPool) bool

	// Scale is the primary entry point for provider specific scaling
	// operations and is responsible for calling the appropriate
	// provider internal methods for scale-out and scale-in operations,
	// verification and rety (where applicable).
	Scale(*WorkerPool, *Config, *NodeRegistry) error
}
