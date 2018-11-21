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

	req := team.GetTeamRequest{Name:""}
	response, teamErr := teamCli.Get(req)

	if teamErr != nil {
		panic(teamErr)
	}

	fmt.Printf("Id: %s\n", response.Id)
	fmt.Printf("Name: %s\n", response.Name)
	fmt.Printf("Members:\n")

	for _, member := range response.Members {
		fmt.Printf("User: %s\n", member.User)
		fmt.Printf("Role: %s\n", member.Role)
		fmt.Printf("\n")
	}
}