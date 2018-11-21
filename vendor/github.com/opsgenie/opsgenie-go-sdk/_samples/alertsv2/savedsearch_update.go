package main

import (
	"github.com/opsgenie/opsgenie-go-sdk/_samples/constants"
	"github.com/opsgenie/opsgenie-go-sdk/alertsv2"
	"fmt"
	ogcli "github.com/opsgenie/opsgenie-go-sdk/client"
	"github.com/opsgenie/opsgenie-go-sdk/alertsv2/savedsearches"
)

func main() {
	cli := new(ogcli.OpsGenieClient)
	cli.SetAPIKey(constants.APIKey)

	alertCli, _ := cli.AlertV2()

	request := savedsearches.UpdateSavedSearchRequest{
		Name:  "list-blue-team-alerts",
		Owner: alertsv2.User{Username: "user@opsgenie.com", },
		Teams: []alertsv2.Team{
			{Name: "green_team"},
		},
		NewName:     "list-green-team-alerts",
	}

	response, err := alertCli.UpdateSavedSearch(request)

	if err != nil {
		fmt.Println(err.Error())
	} else {
		savedSearch := response.SavedSearch

		fmt.Println("ID: " + savedSearch.ID)
		fmt.Println("Name: " + savedSearch.Name)
	}
}
