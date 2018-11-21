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
	}

	criteria := notificationv2.Criteria{
		Type: notificationv2.MatchAllConditionsType,
		Conditions: []notificationv2.Condition{
			{
				Field:         notificationv2.ExtraPropertiesField,
				Key:           "system",
				Not:           true,
				Operation:     notificationv2.EqualsConditionOperation,
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

	notificationTime := []notificationv2.NotificationTime{
		notificationv2.FifteenMinutesAgoNotificationTime,
	}

	timeAmount := 1
	steps := []notificationv2.Step{
		{
			SendAfter: notificationv2.SendAfter{
				TimeAmount: timeAmount,
				TimeUnit:   notificationv2.Minutes,
			},
			Contact: notificationv2.Contact{
				Method: notificationv2.EmailNotifyMethod,
				To:     "user@company.com",
			},
			Enabled: true,
		},
	}

	repeat := notificationv2.Repeat{Enabled: false}

	response, err := notificationCli.Create(notificationv2.CreateNotificationRequest{
		Identifier:      identifier,
		Name:            "Test create-alert",
		ActionType:      notificationv2.CreateAlertActionType,
		NotificationTime: notificationTime,
		Criteria:        criteria,
		TimeRestriction: timeRestriction,
		Order:           1,
		Steps:           steps,
		Repeat:          repeat,
		Enabled:         true,
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
