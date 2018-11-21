package main

import (
	"fmt"

	ogcli "github.com/opsgenie/opsgenie-go-sdk/client"
	"github.com/opsgenie/opsgenie-go-sdk/userv2"
	"github.com/opsgenie/opsgenie-go-sdk/_samples/constants"
)

func main() {
	cli := new(ogcli.OpsGenieClient)
	cli.SetAPIKey(constants.APIKey)

	userCli, _ := cli.UserV2()

	userRole := &userv2.UserRole{
		Name: userv2.OwnerRoleId,
	}

	request := userv2.CreateUserRequest{
		UserName:      "user2@company.com",
		FullName:      "User Name",
		Role:          userRole,
		SkypeUsername: "user.name",
		UserAddress: userv2.UserAddress{
			Country: "US",
			State:   "Indiana",
			City:    "Terre Haute",
			Line:    "567 Stratford Park",
			ZipCode: "47802",
		},
		Tags:              []string{"advanced", "marked"},
		Details:           map[string][]string {"detail1key": {"detail1dvalue1", "detail1value2"}},
		Timezone:          "Europe/Istanbul",
		Locale:            "en_US",
		DisableInvitation: true,
	}

	response, err := userCli.Create(request)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("Id: %s, username: %#v\n", response.User.ID, response.User.Name)
	}
}
