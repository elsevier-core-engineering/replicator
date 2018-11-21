package aws

import (
	"github.com/aws/aws-sdk-go/aws"
)

// awsConfig returns a standard new Config pointer for use during service creation
func awsConfig(region string) *aws.Config {
	return aws.NewConfig().
		WithMaxRetries(5).
		WithRegion(region).
		WithLogLevel(aws.LogDebugWithRequestRetries | aws.LogDebugWithRequestErrors)
}
