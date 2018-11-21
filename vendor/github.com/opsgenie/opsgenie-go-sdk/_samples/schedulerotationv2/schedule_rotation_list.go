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

	scheduleCli, _ := cli.ScheduleRotationV2()

	identifier := &schedulev2.ScheduleIdentifier{
		Name: 			"Integration-schedule",
	}
	response, err := scheduleCli.List(schedulev2.ListScheduleRotationRequest{
		ScheduleIdentifier: 	identifier,
	})

	if err != nil {
		fmt.Println(err.Error())
	} else {
		for i, schedule := range response.Schedule {
			fmt.Printf("%dth Schedule ID: %s, Name: %s, Description: %s" +
				",Timezone: %s , Enabled: %t, OwnerTeam Name: %s \n",
				i+1, schedule.ID, schedule.Name, schedule.Description, schedule.Timezone, schedule.Enabled, schedule.OwnerTeam.Name)
			for j, rotation := range schedule.Rotations {
				fmt.Printf("%dth Rotation Name: %s, Start Date: %s, End Date: %s, Type: %s\n", j+1, rotation.Name, rotation.StartDate, rotation.EndDate, rotation.Type)
			}

		}
	}
}
