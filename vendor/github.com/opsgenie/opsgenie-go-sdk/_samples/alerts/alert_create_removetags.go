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

	tags := []string{"tag1", "tag2"}

	//create the alert with tags
	createReq := alerts.CreateAlertRequest{Message: samples.RandStringWithPrefix("Test", 8), Tags: tags}
	createResp, createErr := alertCli.Create(createReq)
	if createErr != nil{
		panic(createErr)
	}

	fmt.Printf("message: %s\n", createResp.Message)
	fmt.Printf("alertID: %s\n", createResp.AlertID)
	fmt.Printf("status: %s\n", createResp.Status)
	fmt.Printf("code: %d\n", createResp.Code)

	//remove the tags from the alert
	removeReq := alerts.RemoveTagsAlertRequest{ID: createResp.AlertID, Tags: tags}
	removeResp, removeErr := alertCli.RemoveTags(removeReq)
	if removeErr != nil{
		panic(removeErr)
	}

	fmt.Printf("status: %s\n", removeResp.Status)
	fmt.Printf("code: %d\n", removeResp.Code)
}
