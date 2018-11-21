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

	response, err := notificationCli.List(notificationv2.ListNotificationRequest{
		Identifier: &notificationv2.Identifier{
			Username:"user@company.com",
		},
	})
	if err != nil {
		fmt.Println(err.Error())
	} else {
		for i, notification := range response.Notifications {
			fmt.Printf("%d. %s %s\n", i, notification.Name, notification.ID)
		}
	}
}
