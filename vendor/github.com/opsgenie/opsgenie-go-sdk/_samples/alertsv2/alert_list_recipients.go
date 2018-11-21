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

	request := alertsv2.ListAlertRecipientsRequest{
		Identifier: &alertsv2.Identifier{
			TinyID: "2",
		},
	}

	response, err := alertCli.ListAlertRecipients(request)

	if err != nil {
		fmt.Println(err.Error())
	} else {
		for i, recipient := range response.Recipients {
			fmt.Println(strconv.Itoa(i) + ". " + recipient.User.Username + " : " + recipient.State)
		}
	}
}