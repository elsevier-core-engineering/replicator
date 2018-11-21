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
		fmt.Printf("Usage:\n\tdeletealerts [API Key] -> Deletes All Alerts.\n\tdeletealerts [API Key] \"[Message]\" -> Deletes Alerts with Given Message.\n")
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

	countresp, countErr := alertCli.Count(countreq)

	if countErr != nil {
		panic(countErr)
	}

	var totalAlertCount int = countresp.Count
	var currentAlertCount int = 0
	var totalDeletedAlertCount int = 0

	for true {
		if currentAlertCount == totalAlertCount {
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

		listresp, listErr := alertCli.List(listreq)

		if listErr != nil {
			panic(listErr)
		}

		currentAlertCount += len(listresp.Alerts)

		for _, alert := range listresp.Alerts {
			if len(searchMessage) != 0 {
				if strings.Compare(alert.Message, searchMessage) == 0 {
					delreq := alerts.DeleteAlertRequest{}

					delreq.ID = alert.ID
					delreq.APIKey = apiKey

					delresp, delErr := alertCli.Delete(delreq)

					if delErr != nil {
						panic(delErr)
					}

					if delresp.Code != 200 {
						fmt.Errorf("Error when deleting alert with ID: %s\n", alert.ID)
					}

					totalDeletedAlertCount += 1
				}
			} else {
				delreq := alerts.DeleteAlertRequest{}

				delreq.ID = alert.ID
				delreq.APIKey = apiKey

				delresp, delErr := alertCli.Delete(delreq)

				if delErr != nil {
					panic(delErr)
				}

				if delresp.Code != 200 {
					fmt.Errorf("Error when deleting alert with ID: %s\n", alert.ID)
				}

				totalDeletedAlertCount += 1
			}

			nextCreatedAt = alert.CreatedAt
		}
	}

	fmt.Printf("Script has finished, " + strconv.Itoa(totalDeletedAlertCount) + " alert(s) has been deleted.\n")
}
