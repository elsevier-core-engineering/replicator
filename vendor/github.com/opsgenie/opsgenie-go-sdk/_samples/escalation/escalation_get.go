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

	req := esc.GetEscalationRequest{Name:""}
	response, escErr := escCli.Get(req)

	if escErr != nil {
		panic(escErr)
	}

	fmt.Printf("Id: %s\n", response.Id)
	fmt.Printf("Name: %s\n", response.Name)
	fmt.Printf("Team: %s\n", response.Team)
	fmt.Printf("Rules:\n")
	for _, rule := range response.Rules {
		fmt.Printf("Delay: %d\n", rule.Delay)
		fmt.Printf("Notify: %s\n", rule.Notify)
		fmt.Printf("NotifyType: %s\n", rule.NotifyType)
		fmt.Printf("NotifyCondition: %s\n", rule.NotifyCondition)
		fmt.Printf("\n")
	}

}