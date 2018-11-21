package notifier

import (
	"fmt"

	"github.com/d3sw/replicator/logging"
	alerts "github.com/opsgenie/opsgenie-go-sdk/alertsv2"
	ogclient "github.com/opsgenie/opsgenie-go-sdk/client"
)

// OpsGenieProvider contains the required configuration to send OpsGenie
// notifications.
type OpsGenieProvider struct {
	config map[string]string
}

// Name returns the name of the notification endpoint in a lowercase, human
// readable format.
func (og *OpsGenieProvider) Name() string {
	return "opsgenie"
}

// NewOpsGenieProvider creates the OpsGenie notification provider.
func NewOpsGenieProvider(c map[string]string) (Notifier, error) {

	og := &OpsGenieProvider{
		config: c,
	}

	return og, nil
}

// SendNotification will send a notification to OpsGenieProvider using the Event
// library call to create a new incident.
func (og *OpsGenieProvider) SendNotification(message FailureMessage) {

	// Format the message description.
	d := fmt.Sprintf("%s %s_%s_%s",
		message.AlertUID,
		message.ClusterIdentifier,
		message.Reason,
		message.ResourceID)

	ogClinet := new(ogclient.OpsGenieClient)
	ogClinet.SetAPIKey(og.config["OpsGenieAPIKey"])

	alertCli, _ := ogClinet.AlertV2()
	request := alerts.CreateAlertRequest{
		Message:     "replicator notification",
		Alias:       message.AlertUID,
		Description: d,
		Details: map[string]string{
			"alert_uid":          message.AlertUID,
			"cluster_identifier": message.ClusterIdentifier,
			"reason":             message.Reason,
			"resource_id":        message.ResourceID,
		},
		Entity: message.ResourceID,
		Source: "replicator",
	}

	resp, err := alertCli.Create(request)
	if err != nil {
		logging.Error("notifier/opsgenie: an error occurred creating the OpsGenie event: %v", err)
		return
	}

	logging.Info("notifier/opsgenie: incident %s has been triggerd", resp.RequestID)
}
