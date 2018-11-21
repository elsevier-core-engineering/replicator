package notifier

import (
	"fmt"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/d3sw/replicator/logging"
)

// PagerDutyProvider contains the required configuration to send PagerDuty
// notifications.
type PagerDutyProvider struct {
	config map[string]string
}

// Name returns the name of the notification endpoint in a lowercase, human
// readable format.
func (p *PagerDutyProvider) Name() string {
	return "pagerduty"
}

// NewPagerDutyProvider creates the PagerDuty notification provider.
func NewPagerDutyProvider(c map[string]string) (Notifier, error) {

	p := &PagerDutyProvider{
		config: c,
	}

	return p, nil
}

// SendNotification will send a notification to PagerDuty using the Event
// library call to create a new incident.
func (p *PagerDutyProvider) SendNotification(message FailureMessage) {

	// Format the message description.
	d := fmt.Sprintf("%s %s_%s_%s",
		message.AlertUID, message.ClusterIdentifier, message.Reason,
		message.ResourceID)

	// Setup the PagerDuty event structure which will then be used to trigger
	// the event call.
	event := pagerduty.Event{
		ServiceKey:  p.config["PagerDutyServiceKey"],
		Type:        "trigger",
		Description: d,
		Details:     message,
	}

	resp, err := pagerduty.CreateEvent(event)
	if err != nil {
		logging.Error("notifier/pagerduty: an error occurred creating the PagerDuty event: %v", err)
		return
	}

	logging.Info("notifier/pagerduty: incident %s has been triggerd", resp.IncidentKey)
}
