package main

import (
	"fmt"
	ogcli "github.com/opsgenie/opsgenie-go-sdk/client"
	sch "github.com/opsgenie/opsgenie-go-sdk/schedule"
	"github.com/opsgenie/opsgenie-go-sdk/_samples/constants"
)

func main() {
	cli := new(ogcli.OpsGenieClient)
	cli.SetAPIKey(constants.APIKey)

	schCli, cliErr := cli.Schedule()

	if cliErr != nil {
		panic(cliErr)
	}

	req := sch.GetScheduleRequest{Name:""}
	response, schErr := schCli.Get(req)

	if schErr != nil {
		panic(schErr)
	}

	fmt.Printf("Id: %s\n", response.Id)
	fmt.Printf("Name: %s\n", response.Name)
	fmt.Printf("Team: %s\n", response.Team)
	fmt.Printf("Rules:\n")
	for _, rule := range response.Rules {
		fmt.Printf("Id: %s\n", rule.Id)
		fmt.Printf("Name: %s\n", rule.Name)
		fmt.Printf("StartDate: %s\n", rule.StartDate)
		fmt.Printf("EndDate: %s\n", rule.EndDate)
		fmt.Printf("Rotation Type: %s\n", rule.RotationType)
		fmt.Printf("Rotation Length: %d\n", rule.RotationLength)
		fmt.Printf("Participants: %s\n", rule.Participants)
		fmt.Printf("Restrictions:\n")
		for _, restriction := range rule.Restrictions {
			fmt.Printf("Start Day: %s\n", restriction.StartDay)
			fmt.Printf("Start Time: %s\n", restriction.StartTime)
			fmt.Printf("End Day: %s\n", restriction.EndDay)
			fmt.Printf("End Time: %s\n", restriction.EndTime)
			fmt.Printf("\n")
		}
		fmt.Printf("\n")
	}
}