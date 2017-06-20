package replicator

import (
	"github.com/elsevier-core-engineering/replicator/client"
	"github.com/elsevier-core-engineering/replicator/logging"
	"github.com/elsevier-core-engineering/replicator/replicator/structs"
)

func checkScalingThreshold(state *structs.State, direction string, config *structs.Config) (scale bool) {

	switch direction {
	case client.ScalingDirectionIn:
		state.ClusterScaleInRequests++
		state.ClusterScaleOutRequests = 0
		if state.ClusterScaleInRequests == config.ClusterScaling.ScalingThreshold {
			state.ClusterScaleInRequests = 0
			logging.Debug("core/cluster_scaling: scale in requests %v has reached threshold %v",
				state.ClusterScaleInRequests, config.ClusterScaling.ScalingThreshold)
			scale = true
		}

		logging.Debug("core/cluster_scaling: scale in requests %v has not been reached threshold %v",
			state.ClusterScaleInRequests, config.ClusterScaling.ScalingThreshold)

	case client.ScalingDirectionOut:
		state.ClusterScaleOutRequests++
		state.ClusterScaleInRequests = 0
		if state.ClusterScaleOutRequests == config.ClusterScaling.ScalingThreshold {
			state.ClusterScaleOutRequests = 0
			logging.Debug("core/cluster_scaling: scale out requests %v has reached threshold %v",
				state.ClusterScaleInRequests, config.ClusterScaling.ScalingThreshold)
			scale = true
		}

		logging.Debug("core/cluster_scaling: scale out requests %v has not been reached threshold %v",
			state.ClusterScaleOutRequests, config.ClusterScaling.ScalingThreshold)

	default:
		state.ClusterScaleInRequests = 0
		state.ClusterScaleOutRequests = 0
	}

	// One way or another we have updated our internal state, therefore this needs
	// to be written to our persistant state store.
	if err := config.ConsulClient.WriteState(config, state); err != nil {
		logging.Error("core:cluster_scaling: unable to update cluster scaling state to persistant store: %v", err)
		scale = false
	}

	return scale
}
