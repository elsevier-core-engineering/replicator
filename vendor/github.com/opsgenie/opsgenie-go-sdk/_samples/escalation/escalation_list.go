package main

import (
	"fmt"
	ogcli "github.com/opsgenie/opsgenie-go-sdk/client"
	esc "github.com/opsgenie/opsgenie-go-sdk/escalation"
	"github.com/opsgenie/opsgenie-go-sdk/_samples/constants"
)

func main() {
	cli := new(ogcli.OpsGenieClient)
	cli.SetAPIKey(constants.APIKey)

	escCli, cliErr := cli.Escalation()

	if cliErr != nil {
		panic(cliErr)
	}

	req := esc.ListEscalationsRequest{}
	response, escErr := escCli.List(req)

	if escErr != nil {
		panic(escErr)
	}

	for _, esc := range response.Escalations {
		fmt.Printf("Id: %s\n", esc.Id)
		fmt.Printf("Name: %s\n", esc.Name)
		fmt.Printf("Team: %s\n", esc.Team)
		fmt.Printf("Rules:\n")
		for _, rule := range esc.Rules {
			fmt.Printf("Delay: %d\n", rule.Delay)
			fmt.Printf("Notify: %s\n", rule.Notify)
			fmt.Printf("NotifyType: %s\n", rule.NotifyType)
			fmt.Printf("NotifyCondition: %s\n", rule.NotifyCondition)
			fmt.Printf("\n")
		}
	}
}