package main

import (
	"github.com/opsgenie/opsgenie-go-sdk/_samples/constants"
	"fmt"
	ogcli "github.com/opsgenie/opsgenie-go-sdk/client"
	"github.com/opsgenie/opsgenie-go-sdk/alertsv2/savedsearches"
)

func main() {
	cli := new(ogcli.OpsGenieClient)
	cli.SetAPIKey(constants.APIKey)

	alertCli, _ := cli.AlertV2()

	request := savedsearches.DeleteSavedSearchRequest{
		Name: "list-blue-team-alerts",
	}

	_, err := alertCli.DeleteSavedSearch(request)

	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("Deleted")
	}
}
