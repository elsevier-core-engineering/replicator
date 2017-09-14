package helper

import (
	"fmt"
	"reflect"
	"regexp"
	"time"

	"github.com/elsevier-core-engineering/replicator/logging"
	"github.com/elsevier-core-engineering/replicator/replicator/structs"
	"github.com/mitchellh/hashstructure"
)

// FindIP will return the IP address from a string. This is used to deal with
// responses from the Nomad API that contain the port such as 127.0.0.1:4646.
func FindIP(input string) string {
	numBlock := "(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])"
	regexPattern := numBlock + "\\." + numBlock + "\\." + numBlock + "\\." + numBlock

	regEx := regexp.MustCompile(regexPattern)
	return regEx.FindString(input)
}

// Max returns the largest float from a variable length list of floats.
func Max(values ...float64) float64 {
	max := values[0]
	for _, i := range values[1:] {
		if i > max {
			max = i
		}
	}

	return max
}

// Min returns the smallest float from a variable length list of floats.
func Min(values ...float64) float64 {
	min := values[0]
	for _, i := range values[1:] {
		if i < min {
			min = i
		}
	}
	return min
}

// ParseMetaConfig parses meta parameters from a Nomad agent or job
// configuration and validates required keys are present. If any required
// keys are found to be missing, these are returned. Otherwise, an empty
// slice is returned.
func ParseMetaConfig(meta map[string]string, reqKeys []string) (missing []string) {
	// Iterate over the required configuration parameters and
	// record any that are missing.
	for _, reqKey := range reqKeys {
		if _, ok := meta[reqKey]; !ok {
			missing = append(missing, reqKey)
		}
	}
	return
}

// HasObjectChanged compares two objects to determine if they have changed.
func HasObjectChanged(objectA, objectB interface{}) (changed bool, err error) {
	objectAHash, err := hashstructure.Hash(objectA, nil)
	if err != nil {
		return false, fmt.Errorf("error hashing first object %v of type %v: %v",
			objectA, reflect.TypeOf(objectA), err)
	}

	objectBHash, err := hashstructure.Hash(objectB, nil)
	if err != nil {
		return false, fmt.Errorf("error hashing second object %v of type %v: %v",
			objectA, reflect.TypeOf(objectA), err)
	}

	if objectAHash != objectBHash {
		changed = true
	}

	return
}

// FindNodeByAddress is a helper method that searches the node registry
// to determine if a node has been registered with a specific worker pool.
//
// The method searches by node IP address and if no result is found, will
// continue polling the node registry for up to 5 minutes.
func FindNodeByAddress(nodeRegistry *structs.NodeRegistry,
	workerPoolName, nodeAddress string) (ok bool) {

	// Setup a ticker to poll the node registry for the specified worker node
	// and retry up to a specified timeout.
	ticker := time.NewTicker(time.Second * 10)
	timeout := time.Tick(time.Minute * 5)

	logging.Info("core/helper: searching for a registered node with address "+
		"%v in worker pool %v", nodeAddress, workerPoolName)

	for {
		select {
		case <-timeout:
			logging.Error("core/helper: timeout reached while searching the "+
				"node registry for a node with address %v registered in worker "+
				"pool %v", nodeAddress, workerPoolName)
			return

		case <-ticker.C:
			// Obtain a read-only lock on the node registry, retrieve a copy
			// of the worker pool object and release the lock.
			nodeRegistry.Lock.RLock()
			workerPool := nodeRegistry.WorkerPools[workerPoolName]
			nodeRegistry.Lock.RUnlock()

			for registeredNode, nodeRecord := range workerPool.Nodes {
				if FindIP(nodeRecord.HTTPAddr) == nodeAddress {
					logging.Info("core/helper: node %v was found as a registered "+
						"and healhty node with address %v in worker pool %v",
						registeredNode, nodeAddress, workerPoolName)
					return true
				}
			}

			logging.Debug("core/helper: a node with address %v was not found as a "+
				"registered and healthy node in worker pool %v, pausing and "+
				"checking again", nodeAddress, workerPoolName)
		}
	}
}

//FindNodeByRegistrationTime does stuff and things.
// FindNodeByRegistrationTime is a helper method that watches a worker pool
// in the node registry for a newly launched node.
//
// The method watches the node registration records which include the date
// and time the node was registered and looks for a node that was launched
// anytime within the last 60 seconds or later. If no result is found, the
// method will  continue polling the node registry for up to 5 minutes.
func FindNodeByRegistrationTime(nodeRegistry *structs.NodeRegistry,
	workerPoolName string) (node string, err error) {

	// Setup struct to track most recent instance node information.
	instanceTracking := &structs.MostRecentNode{}

	// Calculate node launch threshold.
	launchThreshold := time.Now().Add(-60 * time.Second)
	logging.Debug("LAUNCH THRESHOLD: %v", launchThreshold)

	// Setup a ticker to poll the health status of the specified worker node
	// and retry up to a specified timeout.
	ticker := time.NewTicker(time.Second * 10)
	timeout := time.Tick(time.Minute * 5)

	logging.Info("core/helper: determining most recently launched " +
		"worker node")

	for {
		select {

		case <-timeout:
			err = fmt.Errorf("core/cluster_scaling: timeout reached while "+
				"attempting to determine the most recently launched node in worker "+
				"pool %v", workerPoolName)
			logging.Error("%v", err)
			return

		case <-ticker.C:
			// Obtain a read-only lock on the node registry, retrieve a copy
			// of the worker pool object and release the lock.
			nodeRegistry.Lock.RLock()
			workerPool := nodeRegistry.WorkerPools[workerPoolName]
			nodeRegistry.Lock.RUnlock()

			// Iterate over and determine the most recent instance.
			for node, nodeRegistration := range workerPool.NodeRegistrations {
				logging.Debug("core/cluster_scaling: node %v was discovered %v",
					node, nodeRegistration)

				if nodeRegistration.After(instanceTracking.MostRecentLaunch) {
					instanceTracking.MostRecentLaunch = nodeRegistration
					instanceTracking.InstanceID = node
				}
			}

			if instanceTracking.MostRecentLaunch.After(launchThreshold) {
				logging.Debug("core/cluster_scaling: node %v is the most recently "+
					"launched worker node", instanceTracking.InstanceID)
				return instanceTracking.InstanceID, nil
			}

			logging.Debug("core/cluster_scaling: node %v is the most recently "+
				"launched worker node discovered but its launch time %v is not "+
				"within the launch threshold %v", instanceTracking.InstanceID,
				instanceTracking.MostRecentLaunch, launchThreshold)
		}
	}
}
