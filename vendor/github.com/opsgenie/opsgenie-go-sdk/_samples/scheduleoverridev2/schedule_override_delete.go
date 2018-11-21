package main

import (
	"github.com/opsgenie/opsgenie-go-sdk/_samples/constants"
	"github.com/opsgenie/opsgenie-go-sdk/schedulev2"
	"fmt"
	ogcli "github.com/opsgenie/opsgenie-go-sdk/client"
)

func main() {
	cli := new(ogcli.OpsGenieClient)
	cli.SetAPIKey(constants.APIKey)

	scheduleCli, _ := cli.ScheduleOverrideV2()

	identifier := &schedulev2.ScheduleIdentifier{
		Name:"Integration-schedule",
	}

	response, err := scheduleCli.Delete(schedulev2.DeleteScheduleOverrideRequest{
		ScheduleIdentifier :identifier,
		Alias:				"Override Alias",
	})

	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Printf(
			"RequestID: %s",
			response.ResponseMeta.RequestID,
		)
	}
}
