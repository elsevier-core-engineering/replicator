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

	fmt.Printf("Heartbeat created\n")
	fmt.Printf("-----------------\n")
	fmt.Printf("name: %s\n", response.Name)
	fmt.Printf("status: %s\n", response.Status)
	fmt.Printf("code: %d\n", response.Code)

	// list the HBs
	listReq := hb.ListHeartbeatsRequest{}
	listResp, listErr := hbCli.List(listReq)
	if listErr != nil {
		panic(listErr)
	}

	fmt.Println()
	fmt.Printf("Heartbeats\n")
	fmt.Printf("-----------------\n")
	beats := listResp.Heartbeats
	for _, beat := range beats {
		fmt.Printf("Name: %s\n", beat.Name)
		fmt.Printf("Status %s\n", beat.Status)
		fmt.Printf("Description: %s\n", beat.Description)
		fmt.Printf("Enabled: %t\n", beat.Enabled)
		fmt.Printf("Last Heartbeat: %d\n", beat.LastHeartbeat)
		fmt.Printf("Interval: %d\n", beat.Interval)
		fmt.Printf("Interval Unit: %s\n", beat.IntervalUnit)
		fmt.Printf("Expired: %t\n", beat.Expired)
		fmt.Printf("-----------------\n")
	}
}
