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

	contactGetReq := contacts.GetContactRequest{ Id: contactResp.Id, Username: "fazilet@test.com"}
	contactGetResp, contactGetErr := contactCli.Get(contactGetReq)
	if contactGetErr != nil {
		panic(contactGetErr)
	}

	fmt.Printf("disabledReason: %s\n", contactGetResp.DisabledReason)
	fmt.Printf("method: %s\n", contactGetResp.Method)
	fmt.Printf("to: %s\n", contactGetResp.To)
	fmt.Printf("id: %s\n", contactGetResp.Id)
	fmt.Printf("enabled: %t\n", contactGetResp.Enabled)
}
