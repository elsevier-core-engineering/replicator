package main

import (
	"fmt"

	"github.com/opsgenie/opsgenie-go-sdk/alerts"
	ogcli "github.com/opsgenie/opsgenie-go-sdk/client"
	samples "github.com/opsgenie/opsgenie-go-sdk/_samples"
	"github.com/opsgenie/opsgenie-go-sdk/_samples/constants"
)

func main(){
	// initialize the client
	cli := new (ogcli.OpsGenieClient)
	cli.SetAPIKey(constants.APIKey)

	alertCli, cliErr := cli.Alert()

	if cliErr != nil {
		panic(cliErr)
	}

	// create the alert
	req := alerts.CreateAlertRequest{Message: samples.RandStringWithPrefix("Test", 8)}
	response, alertErr := alertCli.Create(req)

	if alertErr != nil {
		panic(alertErr)
	}

	fmt.Printf("message: %s\n", response.Message)
	fmt.Printf("alert id: %s\n", response.AlertID)
	fmt.Printf("status: %s\n", response.Status)
	fmt.Printf("code: %d\n", response.Code)

	// acknowledge the alert

	ackReq := alerts.AcknowledgeAlertRequest{ ID: response.AlertID }
	ackResponse, ackErr := alertCli.Acknowledge(ackReq)
	if ackErr != nil {
		panic(ackErr)
	}

	fmt.Printf("status: %s\n", ackResponse.Status)
	fmt.Printf("code: %d\n", ackResponse.Code)

	//unacknowledge the alert

	unAckReq := alerts.UnAcknowledgeAlertRequest{ ID: response.AlertID}
	unAckResponse, unAckErr := alertCli.UnAcknowledge(unAckReq)
	if unAckErr != nil {
		panic(unAckErr)
	}

	fmt.Printf("status: %s\n", unAckResponse.Status)
	fmt.Printf("code: %d\n", unAckResponse.Code)
	fmt.Printf("took: %d\n", unAckResponse.Took)
}
