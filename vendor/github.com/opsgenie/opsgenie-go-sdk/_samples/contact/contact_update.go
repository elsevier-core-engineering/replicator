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

	contactUpdateReq := contacts.UpdateContactRequest{ Username: "fazilet@test.com", Id: contactResp.Id, To: "1-8888888888"}
	contactUpdateResp, contactUpdateErr := contactCli.Update(contactUpdateReq)
	if contactUpdateErr != nil {
		panic(contactUpdateErr)
	}

	fmt.Printf("id: %s\n", contactUpdateResp.Id)
	fmt.Printf("status: %s\n", contactUpdateResp.Status)
	fmt.Printf("code: %d\n", contactUpdateResp.Code)
}