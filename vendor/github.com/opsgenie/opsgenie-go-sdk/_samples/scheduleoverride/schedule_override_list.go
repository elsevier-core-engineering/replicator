package main

import (
	"fmt"

	"github.com/opsgenie/opsgenie-go-sdk/_samples/constants"
	ogcli "github.com/opsgenie/opsgenie-go-sdk/client"
	sch "github.com/opsgenie/opsgenie-go-sdk/schedule"
	"github.com/opsgenie/opsgenie-go-sdk/scheduleoverride"
)

func main() {
	cli := new(ogcli.OpsGenieClient)
	cli.SetAPIKey(constants.APIKey)

	schCli, cliErr := cli.Schedule()

	if cliErr != nil {
		panic(cliErr)
	}

	req := sch.ListSchedulesRequest{}
	response, schErr := schCli.List(req)

	if schErr != nil {
		panic(schErr)
	}
	for _, sch := range response.Schedules {
		fmt.Printf("Id: %s\n", sch.Id)
		fmt.Printf("Name: %s\n", sch.Name)
		fmt.Printf("Team: %s\n", sch.Team)

		fmt.Printf("Overrides:\n")

		overrideCli, cliErr := cli.ScheduleOverride()
		if cliErr != nil {
			panic(cliErr)
		}
		req := override.ListScheduleOverridesRequest{
			Schedule: sch.Id,
		}
		overrideResponse, overrideErr := overrideCli.List(req)

		if overrideErr != nil {
			panic(overrideErr)
		}

		for _, over := range overrideResponse.Overrides {
			fmt.Printf("Alias: %s\n", over.Alias)
			fmt.Printf("User: %s\n", over.User)
			fmt.Printf("StartDate: %s\n", over.StartDate)
			fmt.Printf("EndDate: %s\n", over.EndDate)
			fmt.Printf("Timezone: %s\n", over.Timezone)
			fmt.Printf("Rotations: ")
			for _, rot := range over.RotationIds {
				fmt.Printf("%s ", rot)
			}
			fmt.Printf("\n")
		}
		fmt.Printf("\n\n")
	}
}
