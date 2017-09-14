package client

import (
	"reflect"
	"sync"
	"testing"

	"github.com/elsevier-core-engineering/replicator/helper"
	"github.com/elsevier-core-engineering/replicator/replicator/structs"
	"github.com/hashicorp/nomad/api"
)

// Test the parsing of scaling meta configuration parameters and
// validate required parameters are correctly detected as missing.
func TestNodeDiscovery_ParseConfig(t *testing.T) {
	meta := make(map[string]string)

	// Define required meta configuration parameters.
	requiredKeys := []string{
		"replicator_enabled",
		"replicator_max",
		"replicator_min",
		"replicator_notification_uid",
		"replicator_provider",
		"replicator_region",
		"replicator_worker_pool",
	}

	// Test that a node with no scaling configuration correctly indicates
	// that all required configuration parameters are missing.
	emptyConfigReturn := helper.ParseMetaConfig(meta, requiredKeys)
	if len(emptyConfigReturn) != len(requiredKeys) {
		t.Fatalf("expected %v required parameters reported as missing, got %v",
			len(requiredKeys), len(emptyConfigReturn))
	}

	// Add one configuration parameter.
	meta["replicator_worker_pool"] = "example-group"

	partialConfigReturn := helper.ParseMetaConfig(meta, requiredKeys)
	if len(partialConfigReturn) != len(requiredKeys)-1 {
		t.Fatalf("expected %v required parameters reported as missing, got %v",
			len(requiredKeys)-1, len(partialConfigReturn))
	}

	// Add the rest of the meta configuration parameters.
	meta["replicator_enabled"] = "true"
	meta["replicator_max"] = "3"
	meta["replicator_min"] = "1"
	meta["replicator_notification_uid"] = "Test01"
	meta["replicator_provider"] = "aws"
	meta["replicator_region"] = "us-east-1"

	// Test that a node with a complete and valid scaling configuration
	// correctly indicates there are no required configuration parameters
	// are missing.
	fullConfigReturn := helper.ParseMetaConfig(meta, requiredKeys)
	if len(fullConfigReturn) != 0 {
		t.Fatalf("expected 0 required parameters reported as missing, got %v",
			len(fullConfigReturn))
	}
}

// Test the processing of a node configuration. This process parses
// the meta configuration parameters and decodes them into a worker
// pool object. This object is used during the registration process.
func TestNodeDiscovery_ProcessNodeConfig(t *testing.T) {
	// Get a list of nodes, all in a ready state with drain mode disabled.
	nodes := mockNodes(false, true, structs.NodeStatusReady)

	for _, node := range nodes {
		// Retrieve mock node details
		nodeRecord := mockNode(node)

		// Confirm node processing completes successfully on a normalized
		// node record.
		_, err := ProcessNodeConfig(nodeRecord)
		if err != nil {
			t.Fatalf("expected node processing to complete with no error, got %v",
				err)
		}

		// Change meta configuration parameters to an invalid type and
		// confirm node processing thows an error when attempting to decode
		nodeRecord.Meta["replicator_cooldown"] = "300.0"
		_, err = ProcessNodeConfig(nodeRecord)
		if err == nil {
			t.Fatalf("expected parse error (strconv.ParseInt) during node " +
				"processing but no exception was thrown")
		}

		// Modify node record to remove required keys and confirm node
		// processing throws an error.
		delete(nodeRecord.Meta, "replicator_max")
		_, err = ProcessNodeConfig(nodeRecord)
		if err == nil {
			t.Fatalf("node with missing meta configuration parameters failed to " +
				"throw an exception where one was expected.")
		}

		// Strip all meta configuration parameters and confirm node
		// processing throws an error.
		nodeRecord.Meta = make(map[string]string)
		_, err = ProcessNodeConfig(nodeRecord)
		if err == nil {
			t.Fatalf("node with no meta configuration parameters failed to " +
				"throw an exception where one was expected")
		}
	}

}

// Test the ability to determine if the node registry has been updated.
func TestNodeDiscovery_RegistryUpdated(t *testing.T) {
	// Obtain a new node registry object.
	nodeRegistry := newNodeRegistry()

	// Confirm a node registry with no changes returns false.
	if updated := NodeRegistryUpdated(nodeRegistry); updated {
		t.Fatalf("node registry change detected when no change present")
	}

	nodeRegistry.RegisteredNodes["ec2026ec-3632-7cb6-a3d2-88b9e254c793"] =
		"example-group"

	// Confirm a node registry with changes returns true.
	if updated := NodeRegistryUpdated(nodeRegistry); !updated {
		t.Fatalf("node registry change not detected when it should have been")
	}
}

// Test the deregistration of worker pools and nodes within a worker pool.
func TestNodeDiscovery_Deregister(t *testing.T) {
	// Obtain a new node registry object.
	nodeRegistry := newNodeRegistry()

	// Confirm an error is thrown when attempting to deregister a node
	// that was not prevously registered.
	if err := Deregister("bad-node-id", nodeRegistry); err == nil {
		t.Fatalf("no exception was raised when attempting to deregister " +
			"a node that was not previously registered")
	}

	// Simulate a worker pool with multuple nodes, all in a ready state
	// and not in drain mode.
	nodes := mockNodes(false, false, structs.NodeStatusReady)

	// Register the worker pool and nodes so we have something to
	// run deregistrations against.
	for _, node := range nodes {
		nodeRecord := mockNode(node)

		nodeConfig, err := ProcessNodeConfig(nodeRecord)
		if err != nil {
			t.Fatalf("an unexpected exception occurred while processing node "+
				"configuration: %v", err)
		}

		Register(nodeRecord, nodeConfig, nodeRegistry)
	}

	// Setup expected node registry state.
	expectedRegistry := newNodeRegistry()
	expectedRegistry.RegisteredNodes["ec2026ec-3632-7cb6-a3d2-88b9e254c793"] =
		"example-group"
	expectedRegistry.WorkerPools["example-group"] = &structs.WorkerPool{
		Cooldown:         300,
		FaultTolerance:   1,
		Max:              3,
		Min:              1,
		NotificationUID:  "Test01",
		Region:           "us-east-1",
		RetryThreshold:   3,
		ScalingEnabled:   true,
		ScalingThreshold: 3,
		Name:             "example-group",
		Nodes: map[string]*api.Node{
			"ec2026ec-3632-7cb6-a3d2-88b9e254c793": {
				ID:         "ec2026ec-3632-7cb6-a3d2-88b9e254c793",
				Datacenter: "dc1",
				Name:       "example-node-one",
				Drain:      false,
				Status:     structs.NodeStatusReady,
				Meta: map[string]string{
					"replicator_enabled":          "true",
					"replicator_max":              "3",
					"replicator_min":              "1",
					"replicator_notification_uid": "Test01",
					"replicator_provider":         "aws",
					"replicator_region":           "us-east-1",
					"replicator_worker_pool":      "example-group",
				},
			},
		},
	}

	// Attempt to deregister the second node.
	if err := Deregister(nodes[1].ID, nodeRegistry); err != nil {
		t.Fatalf("an unexpected error occurred while attempting to deregister a " +
			"node from an existing worker pool")
	}

	// Copy the pointer reference to our scaling provider.
	expectedRegistry.WorkerPools["example-group"].ScalingProvider =
		nodeRegistry.WorkerPools["example-group"].ScalingProvider

	// Copy node registration records.
	expectedRegistry.WorkerPools["example-group"].NodeRegistrations =
		nodeRegistry.WorkerPools["example-group"].NodeRegistrations

	// Validate the node registry matches our desired state after deregistration
	if !reflect.DeepEqual(nodeRegistry, expectedRegistry) {
		t.Fatalf("expected \n%#v\n\n after node deregistration, got \n\n%#v\n\n",
			expectedRegistry, nodeRegistry)
	}

	// Deregister last node and confirm worker pool is also deregistered.
	if err := Deregister(nodes[0].ID, nodeRegistry); err != nil {
		t.Fatalf("an unexpected error occurred while attempting to deregister a " +
			"node from an existing worker pool")
	}

	// Reset expected node registry and verify our state matches after
	// deregistration of the last node and the worker pool.
	expectedRegistry = newNodeRegistry()
	if !reflect.DeepEqual(nodeRegistry, expectedRegistry) {
		t.Fatalf("expected \n%#v\n\n after node deregistration, got \n\n%#v\n\n",
			expectedRegistry, nodeRegistry)
	}
}

// Test the registration of new worker pools and nodes within a worker pool.
func TestNodeDiscovery_RegisterNode(t *testing.T) {
	// Obtain a new node registry object.
	nodeRegistry := newNodeRegistry()

	// Setup expected node registry state.
	expected := newNodeRegistry()
	expected.RegisteredNodes["ec2026ec-3632-7cb6-a3d2-88b9e254c793"] =
		"example-group"
	expected.WorkerPools["example-group"] = &structs.WorkerPool{
		Cooldown:         300,
		FaultTolerance:   1,
		Max:              3,
		Min:              1,
		NotificationUID:  "Test01",
		Region:           "us-east-1",
		RetryThreshold:   3,
		ScalingEnabled:   true,
		ScalingThreshold: 3,
		Name:             "example-group",
		Nodes: map[string]*api.Node{
			"ec2026ec-3632-7cb6-a3d2-88b9e254c793": {
				ID:         "ec2026ec-3632-7cb6-a3d2-88b9e254c793",
				Datacenter: "dc1",
				Name:       "example-node-one",
				Drain:      false,
				Status:     structs.NodeStatusReady,
				Meta: map[string]string{
					"replicator_enabled":          "true",
					"replicator_max":              "3",
					"replicator_min":              "1",
					"replicator_notification_uid": "Test01",
					"replicator_provider":         "aws",
					"replicator_region":           "us-east-1",
					"replicator_worker_pool":      "example-group",
				},
			},
		},
	}

	// Simulate a worker pool with multuple nodes, all in a ready state
	// and not in drain mode.
	nodes := mockNodes(false, false, structs.NodeStatusReady)

	// Retrieve detailed node configuration for first node.
	nodeRecord := mockNode(nodes[0])
	nodeConfig, err := ProcessNodeConfig(nodeRecord)
	if err != nil {
		t.Fatalf("an unexpected exception occurred while processing node "+
			"configuration: %v", err)
	}

	// Register new worker pool and new node.
	if err = Register(nodeRecord, nodeConfig, nodeRegistry); err != nil {
		t.Fatalf("an unexpected error occurred during registration: %v", err)
	}

	// Copy the pointer reference to our scaling provider.
	expected.WorkerPools["example-group"].ScalingProvider =
		nodeRegistry.WorkerPools["example-group"].ScalingProvider

	// Copy node registration records.
	expected.WorkerPools["example-group"].NodeRegistrations =
		nodeRegistry.WorkerPools["example-group"].NodeRegistrations

	// Validate the dynamic node registry matches our desired state.
	if !reflect.DeepEqual(nodeRegistry, expected) {
		t.Fatalf("expected \n%#v\n\n, got \n\n%#v\n\n", expected, nodeRegistry)
	}

	// Retrieve detailed node configuration for second node.
	nodeRecord = mockNode(nodes[1])
	nodeConfig, err = ProcessNodeConfig(nodeRecord)
	if err != nil {
		t.Fatalf("an unexpected exception occurred while processing node "+
			"configuration: %v", err)
	}

	// Update our expected node registry state.
	expected.RegisteredNodes["ec282e52-4fb6-5950-ef5b-257fced6313c"] =
		"example-group"
	workerPoolNodes := expected.WorkerPools["example-group"].Nodes
	workerPoolNodes["ec282e52-4fb6-5950-ef5b-257fced6313c"] = &api.Node{
		ID:         "ec282e52-4fb6-5950-ef5b-257fced6313c",
		Datacenter: "dc1",
		Name:       "example-node-two",
		Drain:      false,
		Status:     structs.NodeStatusReady,
		Meta: map[string]string{
			"replicator_enabled":          "true",
			"replicator_max":              "3",
			"replicator_min":              "1",
			"replicator_notification_uid": "Test01",
			"replicator_provider":         "aws",
			"replicator_region":           "us-east-1",
			"replicator_worker_pool":      "example-group",
		},
	}

	// Register second node to existing worker pool.
	Register(nodeRecord, nodeConfig, nodeRegistry)

	// Copy the pointer reference to our scaling provider.
	expected.WorkerPools["example-group"].ScalingProvider =
		nodeRegistry.WorkerPools["example-group"].ScalingProvider

	// Validate the dynamic node registry matches our desired state.
	if !reflect.DeepEqual(nodeRegistry, expected) {
		t.Fatalf("expected \n%#v\n\n, got \n\n%#v\n\n", expected, nodeRegistry)
	}

	// Update meta configuration parameters on second node to trigger a
	// worker pool configuration change.
	nodeRecord.Meta["replicator_max"] = "5"
	nodeConfig, err = ProcessNodeConfig(nodeRecord)
	if err != nil {
		t.Fatalf("an unexpected exception occurred while processing node "+
			"configuration: %v", err)
	}

	// Update expected state.
	expected.WorkerPools["example-group"].Max = 5
	workerPoolNodes = expected.WorkerPools["example-group"].Nodes
	workerPoolNode := workerPoolNodes["ec282e52-4fb6-5950-ef5b-257fced6313c"]
	workerPoolNode.Meta["replicator_max"] = "5"

	// Register node with updated meta config parameters.
	Register(nodeRecord, nodeConfig, nodeRegistry)

	// Confirm worker pool is updated to reflect changes.
	workerPool := nodeRegistry.WorkerPools["example-group"]
	expectedPool := expected.WorkerPools["example-group"]

	if !reflect.DeepEqual(workerPool, expectedPool) {
		t.Fatalf("expected \n%#v\n\n, got \n\n%#v\n\n", expectedPool, workerPool)
	}

	// Reset node registry object and confirm an exception is raised when
	// we attempt to register a node in a down state.
	nodeRegistry = newNodeRegistry()
	nodeRecord.Status = structs.NodeStatusDown
	if err = Register(nodeRecord, nodeConfig, nodeRegistry); err == nil {
		t.Fatalf("no exception was raised when we attempted to register " +
			"a node in a down state")
	}

	// Reset node registry object and confirm an exception is raised when
	// we attempt to register a node in drain mode.
	nodeRegistry = newNodeRegistry()
	nodeRecord.Status = structs.NodeStatusReady
	nodeRecord.Drain = true
	if err = Register(nodeRecord, nodeConfig, nodeRegistry); err == nil {
		t.Fatalf("no exception was raised when we attempted to register " +
			"a node with drain mode enabled")
	}

	// Reset node registry object and confirm an exception is raised when
	// we attempt to register a node with an invalid scaling provider.
	nodeRegistry = newNodeRegistry()
	nodes = mockNodes(false, false, structs.NodeStatusReady)
	nodeRecord = mockNode(nodes[0])
	nodeRecord.Meta["replicator_provider"] = "foo"
	nodeConfig, err = ProcessNodeConfig(nodeRecord)
	if err != nil {
		t.Fatalf("an unexpected exception occured while processing node "+
			"configuration: %v", err)
	}
	if err := Register(nodeRecord, nodeConfig, nodeRegistry); err == nil {
		t.Fatalf("no exception was raised when we attempted to register " +
			"a node with an invalid scaling provider configured")
	}
}

func newNodeRegistry() *structs.NodeRegistry {
	// Initialize a new node registry object.
	return &structs.NodeRegistry{
		WorkerPools:     make(map[string]*structs.WorkerPool),
		RegisteredNodes: make(map[string]string),
		Lock:            sync.RWMutex{},
	}
}

func mockNodes(drain bool, singleton bool, status string) (nodes []*api.NodeListStub) {
	// Build a node stub record that permits the caller to toggle
	// the status and drain mode.
	nodes = append(nodes, &api.NodeListStub{
		ID:         "ec2026ec-3632-7cb6-a3d2-88b9e254c793",
		Datacenter: "dc1",
		Name:       "example-node-one",
		Drain:      drain,
		Status:     status,
	})

	if singleton == false {
		// Build a node stub record that is static.
		nodes = append(nodes, &api.NodeListStub{
			ID:         "ec282e52-4fb6-5950-ef5b-257fced6313c",
			Datacenter: "dc1",
			Name:       "example-node-two",
			Drain:      false,
			Status:     structs.NodeStatusReady,
		})
	}

	return
}

func mockNode(node *api.NodeListStub) (nodeRecord *api.Node) {
	// Initialize a new full node object.
	nodeRecord = &api.Node{
		ID:         node.ID,
		Datacenter: node.Datacenter,
		Name:       node.Name,
		Drain:      node.Drain,
		Status:     node.Status,
		Meta:       make(map[string]string),
	}

	// Build meta configuration parameters
	meta := make(map[string]string)
	meta["replicator_enabled"] = "true"
	meta["replicator_max"] = "3"
	meta["replicator_min"] = "1"
	meta["replicator_notification_uid"] = "Test01"
	meta["replicator_provider"] = "aws"
	meta["replicator_region"] = "us-east-1"
	meta["replicator_worker_pool"] = "example-group"

	// Add meta configuration parameters to node record.
	nodeRecord.Meta = meta

	return
}
