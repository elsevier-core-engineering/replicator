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

	req := team.DeleteTeamRequest{Name:""}
	response, teamErr := teamCli.Delete(req)

	if teamErr != nil {
		panic(teamErr)
	}

	fmt.Printf("status: %s\n", response.Status)
	fmt.Printf("code: %d\n", response.Code)
}
