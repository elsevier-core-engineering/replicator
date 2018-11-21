package main

import (
	"fmt"
	ogcli "github.com/opsgenie/opsgenie-go-sdk/client"
	sch "github.com/opsgenie/opsgenie-go-sdk/schedule"
	"github.com/opsgenie/opsgenie-go-sdk/_samples/constants"
)

func main() {
	cli := new(ogcli.OpsGenieClient)
	cli.SetAPIKey(constants.APIKey)

	schCli, cliErr := cli.Schedule()

	if cliErr != nil {
		panic(cliErr)
	}

	req := sch.DeleteScheduleRequest{Name:""}
	response, schErr := schCli.Delete(req)

	if schErr != nil {
		panic(schErr)
	}

	fmt.Printf("status: %s\n", response.Status)
	fmt.Printf("code: %d\n", response.Code)
}