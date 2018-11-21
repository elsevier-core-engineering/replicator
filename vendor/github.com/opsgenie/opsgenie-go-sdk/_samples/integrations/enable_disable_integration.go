package main

import (
	"fmt"

	ogcli "github.com/opsgenie/opsgenie-go-sdk/client"
	itg "github.com/opsgenie/opsgenie-go-sdk/integration"
	"github.com/opsgenie/opsgenie-go-sdk/_samples/constants"
)

func main() {
	cli := new(ogcli.OpsGenieClient)
	cli.SetAPIKey(constants.APIKey)

	integrationCli, cliErr := cli.Integration()

	if cliErr != nil {
		panic(cliErr)
	}
	//disable integration
	disableReq := itg.DisableIntegrationRequest{Name: constants.IntegrationName}
	_, itgError := integrationCli.Disable(disableReq)

	if itgError != nil {
		panic(itgError)
	}
	fmt.Printf("Integration disabled successfuly\n")

	//enable integration
	enableReq := itg.EnableIntegrationRequest{Name: constants.IntegrationName}
	_, itgError = integrationCli.Enable(enableReq)

	if itgError != nil {
		panic(itgError)
	}
	fmt.Printf("Integration enabled successfuly\n")
}
