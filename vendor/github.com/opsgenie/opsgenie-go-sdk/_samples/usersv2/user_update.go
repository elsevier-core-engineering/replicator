package main

import (
	"fmt"

	"github.com/opsgenie/opsgenie-go-sdk/_samples/constants"
	ogcli "github.com/opsgenie/opsgenie-go-sdk/client"
	"github.com/opsgenie/opsgenie-go-sdk/userv2"
)

func main() {
	cli := new(ogcli.OpsGenieClient)
	cli.SetAPIKey(constants.APIKey)

	userCli, _ := cli.UserV2()

	request := userv2.UpdateUserRequest{
		Identifier: &userv2.Identifier{
			Username: "user@company.com",
		},
		FullName:      "Lex Luthor",
		Role:          userv2.UserRole{Name: userv2.AdminRoleId},
		SkypeUsername: "lex.luthor",
		Tags:          []string{"updated"},
		Details:       map[string][]string{"test": {"updated"}},
		Locale:        "de_CH",
		Timezone:      "US/Arizona",
		UserAddress: userv2.UserAddress{
			City:  "Phoenix",
			State: "Arizona",
		},
	}

	response, err := userCli.Update(request)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(response)
	}
}
