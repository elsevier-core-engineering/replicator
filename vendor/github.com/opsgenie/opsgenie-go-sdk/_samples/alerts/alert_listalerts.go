package main

import (
	"fmt"

	"github.com/opsgenie/opsgenie-go-sdk/alerts"
	ogcli "github.com/opsgenie/opsgenie-go-sdk/client"
	"github.com/opsgenie/opsgenie-go-sdk/_samples/constants"
)

func main() {
	cli := new(ogcli.OpsGenieClient)
	cli.SetAPIKey(constants.APIKey)

	alertCli, cliErr := cli.Alert()

	if cliErr != nil {
		panic(cliErr)
	}

	// list the alerts
	listreq := alerts.ListAlertsRequest{}
	listresp, listErr := alertCli.List(listreq)

	if listErr != nil {
		panic(listErr)
	}

	for _, alert := range listresp.Alerts {
		fmt.Printf("Id: %s\n", alert.ID)
		fmt.Printf("Alias: %s\n", alert.Alias)
		fmt.Printf("Message: %s\n", alert.Message)
		fmt.Printf("Status: %s\n", alert.Status)
		fmt.Printf("IsSeen?: %t\n", alert.IsSeen)
		fmt.Printf("Acknowledged?: %t\n", alert.Acknowledged)
		fmt.Printf("Created at: %d\n", alert.CreatedAt)
		fmt.Printf("Updated at: %d\n", alert.UpdatedAt)
		fmt.Printf("Tiny id: %s\n", alert.TinyID)
		fmt.Printf("Owner: %s\n", alert.Owner)
	}
}
