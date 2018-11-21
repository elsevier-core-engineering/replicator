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

	rules := []esc.Rule{}
	rule := esc.Rule{Delay:4, Notify:"", NotifyCondition: ""}
	rules = append(rules, rule)
	req := esc.UpdateEscalationRequest{Id:"", Name: "", Rules: rules}
	response, escErr := escCli.Update(req)

	if escErr != nil {
		panic(escErr)
	}

	fmt.Printf("status: %s\n", response.Status)
	fmt.Printf("code: %d\n", response.Code)
}
