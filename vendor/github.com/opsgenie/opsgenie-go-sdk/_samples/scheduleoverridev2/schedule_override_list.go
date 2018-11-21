package main

import (
	"github.com/opsgenie/opsgenie-go-sdk/_samples/constants"
	"github.com/opsgenie/opsgenie-go-sdk/schedulev2"
	"fmt"
	ogcli "github.com/opsgenie/opsgenie-go-sdk/client"
)

func main() {
	cli := new(ogcli.OpsGenieClient)
	cli.SetAPIKey(constants.APIKey)

	scheduleCli, _ := cli.ScheduleOverrideV2()

	identifier := &schedulev2.ScheduleIdentifier{
		Name: 			"Integration-schedule",
	}
	response, err := scheduleCli.List(schedulev2.ListScheduleOverrideRequest{
		ScheduleIdentifier: 	identifier,
	})

	if err != nil {
		fmt.Println(err.Error())
	} else {
		for i, schedule := range response.ScheduleOverrides {
			fmt.Printf("%dth Schedule Alias: %s, StartDate: %s, EndDate: %s" +
				",UserType: %s , UserID: %t, UserUsername: %s \n",
				i+1, schedule.Alias, schedule.StartDate, schedule.EndDate, schedule.User.Type, schedule.User.ID, schedule.User.Username)
			for j, rotation := range schedule.Rotations {
				fmt.Printf("%dth Rotation Name: %s, Start Date: %s, End Date: %s, Type: %s\n", j+1, rotation.Name, rotation.StartDate, rotation.EndDate, rotation.Type)
			}

		}
	}
}
