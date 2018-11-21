package main

import (
	"fmt"

	ogcli "github.com/opsgenie/opsgenie-go-sdk/client"
	hb "github.com/opsgenie/opsgenie-go-sdk/heartbeat"
	samples "github.com/opsgenie/opsgenie-go-sdk/_samples"
	"github.com/opsgenie/opsgenie-go-sdk/_samples/constants"
)

func main() {
	cli := new(ogcli.OpsGenieClient)
	cli.SetAPIKey(constants.APIKey)

	hbCli, cliErr := cli.Heartbeat()

	if cliErr != nil {
		panic(cliErr)
	}

	// create the hb
	enabled := true
	req := hb.AddHeartbeatRequest{
		Name: samples.RandStringWithPrefix("Test", 4),
		IntervalUnit:"minutes",
		Enabled: &enabled,
		Interval:5,
		Description:"Heartbeat description"}
	response, hbErr := hbCli.Add(req)

	if hbErr != nil {
		panic(hbErr)
	}

	fmt.Printf("Heartbeat added\n")
	fmt.Printf("---------------\n")
	fmt.Printf("name: %s\n", response.Name)
	fmt.Printf("status: %s\n", response.Status)
	fmt.Printf("code: %d\n", response.Code)

	// send heart beat request
	sendReq := hb.SendHeartbeatRequest{Name: response.Name}
	sendResp, sendErr := hbCli.Send(sendReq)

	if sendErr != nil {
		panic(sendErr)
	}

	fmt.Println()
	fmt.Printf("Heartbeat request sent\n")
	fmt.Printf("----------------------\n")
	fmt.Printf("Heartbeat: %d\n", sendResp.Heartbeat)
	fmt.Printf("Will expire at: %d\n", sendResp.WillExpireAt)
	fmt.Printf("Status: %s\n", sendResp.Status)
	fmt.Printf("Code: %d\n", sendResp.Code)
}
