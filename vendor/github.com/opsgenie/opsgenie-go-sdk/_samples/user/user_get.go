package main

import (
	"fmt"

	ogcli "github.com/opsgenie/opsgenie-go-sdk/client"
	"github.com/opsgenie/opsgenie-go-sdk/user"
	"github.com/opsgenie/opsgenie-go-sdk/_samples/constants"
)

func main() {
	cli := new(ogcli.OpsGenieClient)
	cli.SetAPIKey(constants.APIKey)

	userCli, cliErr := cli.User()

	if cliErr != nil {
		panic(cliErr)
	}

	req := user.GetUserRequest{Username: ""}
	response, userErr := userCli.Get(req)

	if userErr != nil {
		panic(userErr)
	}

	fmt.Printf("Id: %s\n", response.Id)
	fmt.Printf("Username: %s\n", response.Username)
	fmt.Printf("Fullname: %s\n", response.Fullname)
	fmt.Printf("Timezone: %s\n", response.Timezone)
	fmt.Printf("Locale: %s\n", response.Locale)
	fmt.Printf("State: %s\n", response.State)
	fmt.Printf("Escalations: %v\n", response.Escalations)
	fmt.Printf("Schedules: %v\n", response.Schedules)
	fmt.Printf("Role: %v\n", response.Role)
	fmt.Printf("Groups: %v\n", response.Groups)
	fmt.Printf("Contacts: %v\n", response.Contacts)
	fmt.Printf("SkypeUserName: %s\n", response.SkypeUsername)
}
