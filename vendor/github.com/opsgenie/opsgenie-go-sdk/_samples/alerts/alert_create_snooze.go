package main

import (
	"fmt"

	"github.com/opsgenie/opsgenie-go-sdk/alerts"
	ogcli "github.com/opsgenie/opsgenie-go-sdk/client"
	samples "github.com/opsgenie/opsgenie-go-sdk/_samples"
	"github.com/opsgenie/opsgenie-go-sdk/_samples/constants"
)

func main(){

	cli := new (ogcli.OpsGenieClient)
	cli.SetAPIKey(constants.APIKey)

	alertCli, cliErr := cli.Alert()
	if cliErr != nil{
		panic(cliErr)
	}

	//create the alert
	createReq := alerts.CreateAlertRequest{Message: samples.RandStringWithPrefix("Test", 8)}
	createResp, createErr := alertCli.Create(createReq)
	if createErr != nil{
		panic(createErr)
	}

	fmt.Printf("message: %s\n", createResp.Message)
	fmt.Printf("alertID: %s\n", createResp.AlertID)
	fmt.Printf("status: %s\n", createResp.Status)
	fmt.Printf("code: %d\n", createResp.Code)

	//snooze the alert
	snoozeReq := alerts.SnoozeAlertRequest{ID: createResp.AlertID, EndDate: constants.EndDate}
	snoozeResp, snoozeErr := alertCli.Snooze(snoozeReq)
	if snoozeErr != nil{
		panic(snoozeErr)
	}

	fmt.Printf("status: %s\n", snoozeResp.Status)
	fmt.Printf("code: %s\n", snoozeResp.Code)
}
