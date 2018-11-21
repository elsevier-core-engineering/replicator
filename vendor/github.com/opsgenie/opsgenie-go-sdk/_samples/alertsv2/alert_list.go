package main

import (
	"github.com/opsgenie/opsgenie-go-sdk/_samples/constants"
	"github.com/opsgenie/opsgenie-go-sdk/alertsv2"
	"fmt"
	ogcli "github.com/opsgenie/opsgenie-go-sdk/client"
	"strconv"
)

func main() {

	cli := new(ogcli.OpsGenieClient)
	cli.SetAPIKey(constants.APIKey)

	alertCli, _ := cli.AlertV2()

	response, err := alertCli.List(alertsv2.ListAlertRequest{
		Limit:                5,
		Offset:               0,
		SearchIdentifierType: alertsv2.Name,

	})
	if err != nil {
		fmt.Println(err.Error())
	} else {
		for i, alert := range response.Alerts {
			fmt.Println(strconv.Itoa(i) + ". " + alert.Message)
		}
	}
}
