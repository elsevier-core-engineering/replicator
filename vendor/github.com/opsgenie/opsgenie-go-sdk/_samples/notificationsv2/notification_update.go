package main

import (
	"fmt"

	"github.com/opsgenie/opsgenie-go-sdk/_samples/constants"
	ogcli "github.com/opsgenie/opsgenie-go-sdk/client"
	"github.com/opsgenie/opsgenie-go-sdk/notificationv2"
)

func main() {
	cli := new(ogcli.OpsGenieClient)
	cli.SetAPIKey(constants.APIKey)

	notificationCli, _ := cli.NotificationV2()

	identifier := &notificationv2.Identifier{
		Username: "user@company.com",
		RuleID: "example-notification-id",
	}

	criteria := notificationv2.Criteria{
		Type: notificationv2.MatchAllConditionsType,
		Conditions: []notificationv2.Condition{
			{
				Field:         "extra-properties",
				Key:           "system",
				Not:           true,
				Operation:     "equals",
				ExpectedValue: "mysql",
			},
		},
	}

	startHour := 3
	endHour := 15
	startMin := 5
	endMin := 30

	timeRestriction := notificationv2.TimeRestriction{
		Type: notificationv2.WeekendAndTimeOfDayTimeRestriction,
		Restrictions: []notificationv2.Restriction{
			{
				StartDay:  notificationv2.Monday,
				EndDay:    notificationv2.Friday,
				StartHour: startHour,
				EndHour:   endHour,
				StartMin:  startMin,
				EndMin:    endMin,
			},
		},
	}

	response, err := notificationCli.Update(notificationv2.UpdateNotificationRequest{
		Identifier:      identifier,
		Name:            "Test create-alert(changed)",
		Criteria:        criteria,
		TimeRestriction: timeRestriction,
		Enabled:         true,
		Order:           2,
	})
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Printf(
			"ID: %s, Name: %s, Enabled: %t\n",
			response.Notification.ID,
			response.Notification.Name,
			response.Notification.Enabled,
		)
	}
}
