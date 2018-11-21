package cloud

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/d3sw/replicator/cloud/aws"
	"github.com/d3sw/replicator/logging"
	"github.com/d3sw/replicator/replicator/structs"
)

// BuiltinScalingProviders tracks the available scaling providers.
// The provider name is the name used when configuring nodes for autoscaling.
var BuiltinScalingProviders = map[string]ScalingProviderFactory{
	"aws": aws.NewAwsScalingProvider,
}

// ScalingProviderFactory is a factory method type for instantiating a new
// instance of a scaling provider.
type ScalingProviderFactory func(
	conf *structs.WorkerPool) (structs.ScalingProvider, error)

// NewScalingProvider is the entry point method for processing the scaling
// provider configuration in worker pool nodes, finding the correct factory
// method and setting up the scaling provider.
func NewScalingProvider(workerPool *structs.WorkerPool) (structs.ScalingProvider, error) {
	// Query configuration for scaling provider name.
	if workerPool.ProviderName == "" {
		return nil, fmt.Errorf("no scaling provider specified")
	}

	// Lookup the scaling provider factory function.
	providerFactory, ok := BuiltinScalingProviders[workerPool.ProviderName]
	if !ok {
		// Build a list of all supported scaling providers.
		providers := reflect.ValueOf(BuiltinScalingProviders).MapKeys()
		availableProviders := make([]string, len(providers))

		for i := 0; i < len(providers); i++ {
			availableProviders[i] = providers[i].String()
		}

		return nil, fmt.Errorf("unknown scaling provider %v, must be one of: %v",
			workerPool.ProviderName, strings.Join(availableProviders, ","))
	}

	// Setup the scaling provider.
	scalingProvider, err := providerFactory(workerPool)
	if err != nil {
		return nil, fmt.Errorf("an error occurred while setting up scaling "+
			"provider %v: %v", workerPool.ProviderName, err)
	}

	logging.Debug("cloud/scaling_provider: initialized scaling provider %v "+
		"for worker pool %v", workerPool.ProviderName, workerPool.Name)

	return scalingProvider, nil
}
