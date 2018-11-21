package main

import(
	ogcli "github.com/opsgenie/opsgenie-go-sdk/client"
	"github.com/opsgenie/opsgenie-go-sdk/_samples/constants"
	contacts "github.com/opsgenie/opsgenie-go-sdk/contact"
	"fmt"
)

func main(){
	cli := new(ogcli.OpsGenieClient)
	cli.SetAPIKey(constants.APIKey)

	contactCli, cliErr := cli.Contact()
	if cliErr != nil {
		panic(cliErr)
	}

	contactReq := contacts.CreateContactRequest{ Method: "sms", To: "1-9999999999", Username: "fazilet@test.com"}

	contactResp, contactErr := contactCli.Create(contactReq)
	if contactErr != nil {
		panic(contactErr)
	}

	contactEnableReq := contacts.EnableContactRequest{ Id: contactResp.Id, Username: "fazilet@test.com"}
	contactEnableResp, contactEnableErr := contactCli.Enable(contactEnableReq)
	if contactEnableErr != nil {
		panic(contactEnableErr)
	}

	fmt.Printf("status: %s\n", contactEnableResp.Status)
}

