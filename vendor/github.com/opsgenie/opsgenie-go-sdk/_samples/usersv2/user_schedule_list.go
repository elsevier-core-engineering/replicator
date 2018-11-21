package main

import (
	"fmt"

	"github.com/opsgenie/opsgenie-go-sdk/_samples/constants"
	ogcli "github.com/opsgenie/opsgenie-go-sdk/client"
	"github.com/opsgenie/opsgenie-go-sdk/userv2"
)

func main() {
	cli := new(ogcli.OpsGenieClient)
	cli.SetAPIKey(constants.APIKey)

	userCli, _ := cli.UserV2()

	request := userv2.ListUserSchedulesRequest{
		Identifier: &userv2.Identifier{Username: "user@company.com"},
	}

	response, err := userCli.ListSchedules(request)
	if err != nil {
		fmt.Println(err)
	} else {
		for _, schedule := range response.Schedules {
			fmt.Println(schedule.Name)
		}
	}
}
