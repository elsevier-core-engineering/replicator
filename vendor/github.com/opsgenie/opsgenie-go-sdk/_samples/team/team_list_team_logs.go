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

	req := team.ListTeamLogsRequest{Name: ""}
	response, teamErr := teamCli.ListLogs(req)

	if teamErr != nil {
		panic(teamErr)
	}

	fmt.Printf("Last Key: %s\n", response.LastKey)
	for _, log := range response.Logs {
		fmt.Printf("Log: %s\n", log.Log)
		fmt.Printf("Owner: %s\n", log.Owner)
		fmt.Printf("CreatedAt: %d\n", log.CreatedAt)
		fmt.Printf("\n")
	}
}