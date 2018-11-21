package main

import (
	"fmt"
	ogcli "github.com/opsgenie/opsgenie-go-sdk/client"
	"github.com/opsgenie/opsgenie-go-sdk/team"
	"github.com/opsgenie/opsgenie-go-sdk/_samples/constants"
)

func main() {
	cli := new(ogcli.OpsGenieClient)
	cli.SetAPIKey(constants.APIKey)

	teamCli, cliErr := cli.Team()

	if cliErr != nil {
		panic(cliErr)
	}

	req := team.ListTeamsRequest{}
	response, teamErr := teamCli.List(req)

	if teamErr != nil {
		panic(teamErr)
	}

	for _, team := range response.Teams {
		fmt.Printf("Id: %s\n", team.Id)
		fmt.Printf("Name: %s\n", team.Name)
		fmt.Printf("Members:\n")
		for _, member := range team.Members {
			fmt.Printf("User: %s\n", member.User)
			fmt.Printf("Role: %s\n", member.Role)
			fmt.Printf("\n")
		}
	}
}