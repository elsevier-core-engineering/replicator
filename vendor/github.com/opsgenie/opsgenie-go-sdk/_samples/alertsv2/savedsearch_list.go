package main

import (
	"github.com/opsgenie/opsgenie-go-sdk/_samples/constants"
	"github.com/opsgenie/opsgenie-go-sdk/alertsv2"
	"fmt"
	ogcli "github.com/opsgenie/opsgenie-go-sdk/client"
)

func main() {
	cli := new(ogcli.OpsGenieClient)
	cli.SetAPIKey(constants.APIKey)

	alertCli, _ := cli.AlertV2()

	request := alertsv2.LisSavedSearchRequest{}

	response, _ := alertCli.ListSavedSearches(request)

	for _, search := range response.SavedSearches {
		fmt.Println("ID: " + search.ID)
		fmt.Println("Name: " + search.Name)
	}
}