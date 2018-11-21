package main

import (
	"fmt"
	"os"
	"github.com/opsgenie/opsgenie-go-sdk/alerts"
	ogcli "github.com/opsgenie/opsgenie-go-sdk/client"
	"strings"
	"strconv"
)

func main() {
	args := os.Args

	var apiKey string
	var searchMessage string

	if len(args) == 2 {
		apiKey = string(args[1])
		searchMessage = ""
	} else if len(args) == 3 {
		apiKey = string(args[1])
		searchMessage = string(args[2])
	} else {
		fmt.Printf("Usage:\n\tclosealerts [API Key] -> Closes All Alerts.\n\tclosealerts [API Key] \"[Message]\" -> Closes Alerts with Given Message.\n")
		os.Exit(0)
	}

	cli := new(ogcli.OpsGenieClient)
	cli.SetAPIKey(apiKey)

	alertCli, cliErr := cli.Alert()

	if cliErr != nil {
		panic(cliErr)
	}

	countreq := alerts.CountAlertRequest{}

	countreq.APIKey = apiKey
	countreq.Status = "open"

	countresp, countErr := alertCli.Count(countreq)

	if countErr != nil {
		panic(countErr)
	}

	var totalOpenAlertCount int = countresp.Count
	var openAlertCount int = 0
	var totalClosedAlertCount int = 0

	for true {
		if openAlertCount == totalOpenAlertCount {
			break
		}

		// list the alerts
		listreq := alerts.ListAlertsRequest{}

		listreq.APIKey = apiKey

		var nextCreatedAt uint64 = 0

		listreq.Limit = 100
		listreq.Order = "asc"
		listreq.SortBy = "createdAt"
		listreq.CreatedAfter = nextCreatedAt
		listreq.Status = "open"

		listresp, listErr := alertCli.List(listreq)

		if listErr != nil {
			panic(listErr)
		}

		openAlertCount += len(listresp.Alerts)

		for _, alert := range listresp.Alerts {
			if alert.Status != "closed" {
				if len(searchMessage) != 0 {
					if strings.Compare(alert.Message, searchMessage) == 0 {
						closereq := alerts.CloseAlertRequest{}

						closereq.ID = alert.ID
						closereq.APIKey = apiKey

						closeresp, closeErr := alertCli.Close(closereq)

						if closeErr != nil {
							panic(closeErr)
						}

						if closeresp.Code != 200 {
							fmt.Errorf("Error when closing alert with ID: %s\n", alert.ID)
						}

						totalClosedAlertCount += 1
					}
				} else {
					closereq := alerts.CloseAlertRequest{}

					closereq.ID = alert.ID
					closereq.APIKey = apiKey

					closeresp, closeErr := alertCli.Close(closereq)

					if closeErr != nil {
						panic(closeErr)
					}

					if closeresp.Code != 200 {
						fmt.Errorf("Error when closing alert with ID: %s\n", alert.ID)
					}

					totalClosedAlertCount += 1
				}
			}

			nextCreatedAt = alert.CreatedAt
		}
	}

	fmt.Printf("Script has finished, " + strconv.Itoa(totalClosedAlertCount) + " alert(s) has been closed.\n")
}
