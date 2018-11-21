package main

import (
	"fmt"

	ogcli "github.com/opsgenie/opsgenie-go-sdk/client"
	"github.com/opsgenie/opsgenie-go-sdk/userv2"
	"github.com/opsgenie/opsgenie-go-sdk/_samples/constants"
)

func main() {
	cli := new(ogcli.OpsGenieClient)
	cli.SetAPIKey(constants.APIKey)

	userCli, _ := cli.UserV2()

	request := userv2.ListUserEscalationsRequest{
		Identifier: &userv2.Identifier{Username: "user@company.com"},
	}

	response, err := userCli.ListEscalations(request)
	if err != nil {
		fmt.Println(err)
	} else {
		for _, escalation := range response.Escalations {
			fmt.Println(escalation)
		}
	}
}
