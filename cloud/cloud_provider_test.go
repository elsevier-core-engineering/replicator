package cloud

import (
	"testing"

	"github.com/d3sw/replicator/replicator/structs"
)

// Test the parsing of scaling meta configuration parameters and
// validate required parameters are correctly detected as missing.
func TestScalingProvider_ProviderFactory(t *testing.T) {
	// Verify an error is thrown when we pass a configuration without
	// specifying the replicator_provider entry.
	workerPool := structs.NewWorkerPool()

	if _, err := NewScalingProvider(workerPool); err == nil {
		t.Fatalf("no exception was raised when we attempted to create a scaling " +
			"provider without specifying the required parameters")
	}

	// Verify an exception is thrown when we specify an invalid scaling
	// provider.
	workerPool.ProviderName = "foo"
	if _, err := NewScalingProvider(workerPool); err == nil {
		t.Fatalf("no exception was raised when we attempted to create a scaling " +
			"provider with an invalid provider type")
	}

	// Verify an exception is thrown when we specify a valid scaling provider
	// but don't include the other required configuration parameters.
	workerPool.ProviderName = "aws"
	if _, err := NewScalingProvider(workerPool); err == nil {
		t.Fatalf("no exception was raised when we specified a valid scaling " +
			"provider but failed to include all required configuration parameters")
	}

	// Verify no exception is raised when we pass all required parameters.
	workerPool.Region = "us-east-1"
	if _, err := NewScalingProvider(workerPool); err != nil {
		t.Fatalf("an exception was raised when we specified a valid scaling " +
			"provider and all required configuration parameters")
	}
}
