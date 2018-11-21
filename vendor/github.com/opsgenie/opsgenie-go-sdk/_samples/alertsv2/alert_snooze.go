package main

import (
	"github.com/opsgenie/opsgenie-go-sdk/_samples/constants"
	"github.com/opsgenie/opsgenie-go-sdk/alertsv2"
	"fmt"
	ogcli "github.com/opsgenie/opsgenie-go-sdk/client"
	"time"
)

func main() {
	cli := new(ogcli.OpsGenieClient)
	cli.SetAPIKey(constants.APIKey)

	alertCli, _ := cli.AlertV2()

	identifier := alertsv2.Identifier{
		TinyID: "2",
	};

	actionRequest := alertsv2.AlertActionRequest{
		Identifier: &identifier,
		User:       "test",
		Source:     "Source",
		Note:       "Note",
	}

	ackRequest := alertsv2.SnoozeRequest{
		AlertActionRequest: actionRequest,
		EndTime: time.Now().AddDate(0, 0, 1),

	}

	response, err := alertCli.Snooze(ackRequest)

	if err != nil {
		fmt.Print(err.Error())
	} else {
		fmt.Println("RequsetId: " + response.RequestID)
	}

}
