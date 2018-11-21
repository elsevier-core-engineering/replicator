package main

import (
	"fmt"

	ogcli "github.com/opsgenie/opsgenie-go-sdk/client"
	"github.com/opsgenie/opsgenie-go-sdk/policy"
	"github.com/opsgenie/opsgenie-go-sdk/_samples/constants"
)

func main() {
	cli := new(ogcli.OpsGenieClient)
	cli.SetAPIKey(constants.APIKey)

	policyCli, cliErr := cli.Policy()

	if cliErr != nil {
		panic(cliErr)
	}
	//disable policy
	disableReq := policy.DisablePolicyRequest{Name: constants.PolicyName}
	_, itgError := policyCli.Disable(disableReq)

	if itgError != nil {
		panic(itgError)
	}
	fmt.Printf("Policy disabled successfuly\n")

	//enable policy
	enableReq := policy.EnablePolicyRequest{Name: constants.PolicyName}
	_, itgError = policyCli.Enable(enableReq)

	if itgError != nil {
		panic(itgError)
	}
	fmt.Printf("Policy enabled successfuly\n")
}
