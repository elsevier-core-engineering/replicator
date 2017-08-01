package client

import (
	"fmt"
	"time"

	metrics "github.com/armon/go-metrics"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/elsevier-core-engineering/replicator/logging"
)

const awsOperationSuccessful = "Successful"

// DescribeAWSRegion uses the EC2 InstanceMetaData endpoint to discover the AWS
// region in which the instance is running.
func DescribeAWSRegion() (region string, err error) {

	ec2meta := ec2metadata.New(session.New())
	identity, err := ec2meta.GetInstanceIdentityDocument()
	if err != nil {
		return "", err
	}
	return identity.Region, nil
}

// NewAWSAsgService creates a new AWS API Session and ASG service connection for
// use across all calls as required.
func NewAWSAsgService(region string) (Session *autoscaling.AutoScaling) {
	sess := session.Must(session.NewSession())
	svc := autoscaling.New(sess, &aws.Config{Region: aws.String(region)})
	return svc
}

// DescribeScalingGroup returns the AWS ASG information of the specified ASG.
func DescribeScalingGroup(asgName string, svc *autoscaling.AutoScaling) (asg *autoscaling.DescribeAutoScalingGroupsOutput, err error) {

	params := &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{
			aws.String(asgName),
		},
	}
	resp, err := svc.DescribeAutoScalingGroups(params)

	if err != nil {
		return resp, err
	}

	return resp, nil
}

// ScaleOutCluster scales the Nomad worker pool by 1 instance, using the current
// configuration as the basis for undertaking the work.
func ScaleOutCluster(asgName string, nodeCount int, svc *autoscaling.AutoScaling) error {

	// Get the current ASG configuration so that we have the basis on which to
	// update to our new desired state.
	asg, err := DescribeScalingGroup(asgName, svc)
	if err != nil {
		return err
	}

	// Evaluate the desired capacity against the current worker count and the
	// MaxSize. Details of why we do the first check GH-70. If the desired
	// capacity +1 is greater than the MaxSize the AWS call would fail so we bail
	// anyway.
	desiredCap := *asg.AutoScalingGroups[0].DesiredCapacity
	maxSize := *asg.AutoScalingGroups[0].MaxSize

	if desiredCap != int64(nodeCount) {
		return fmt.Errorf("asg desired capacity %v does not match the current "+
			"Nomad worker pool count %v", desiredCap, nodeCount)
	}

	if desiredCap+int64(1) > maxSize {
		return fmt.Errorf("incrementing asg count would violate asg max size of %v",
			maxSize)
	}

	// The DesiredCapacity is incremented by 1, while the TerminationPolicies and
	// AvailabilityZones which are required parameters are copied from the Info
	// received from the initial call to DescribeScalingGroup. These params could
	// be directly referenced within UpdateAutoScalingGroupInput but are here for
	// clarity.
	newDesiredCapacity := *asg.AutoScalingGroups[0].DesiredCapacity + int64(1)
	terminationPolicies := asg.AutoScalingGroups[0].TerminationPolicies
	availabilityZones := asg.AutoScalingGroups[0].AvailabilityZones

	// Setup the Input parameters ready for the AWS API call and then trigger the
	// call which will update the ASG.
	params := &autoscaling.UpdateAutoScalingGroupInput{
		AutoScalingGroupName: aws.String(asgName),
		AvailabilityZones:    availabilityZones,
		DesiredCapacity:      aws.Int64(newDesiredCapacity),
		TerminationPolicies:  terminationPolicies,
	}

	logging.Info("client/aws: cluster scaling (scale-out) will now be initiated")

	// Currently it is assumed that no error received from the API means that the
	// increase in ASG size has been successful, or at least will be. This may
	// want to change in the future.
	_, err = svc.UpdateAutoScalingGroup(params)
	if err != nil {
		return err
	}

	// Setup a ticker to poll the autoscaling group and report when an instance
	// has been successfully launched.
	ticker := time.NewTicker(time.Millisecond * 500)
	timeout := time.Tick(time.Minute * 3)

	logging.Info("client/aws: attempting to verify the autoscaling group " +
		"operation has completed successfully")

	for {
		select {
		case <-timeout:
			logging.Info("client/aws: timeout reached while waiting for the "+
				"autoscaling group operation to complete successfully %v", asgName)
			return nil
		case <-ticker.C:
			asg, err := DescribeScalingGroup(asgName, svc)
			if err != nil {
				logging.Error("client/aws: an error occurred while attempting to "+
					"verify the autoscaling group operation completed successfully: %v",
					err)
			} else {
				if len(asg.AutoScalingGroups[0].Instances) == int(newDesiredCapacity) {
					logging.Info("client/aws: verified the autoscaling operation has " +
						"completed successfully")
					metrics.IncrCounter([]string{"cluster", "aws", "scale_out"}, 1)
					return nil
				}
			}
		}
	}
}

// DetachInstance will detach a specified instance from a specified ASG and
// decrements the desired count of the ASG.
func DetachInstance(asgName, instanceID string, svc *autoscaling.AutoScaling) (err error) {
	// Setup the request parameters for the DetachInstances API call.
	params := &autoscaling.DetachInstancesInput{
		AutoScalingGroupName:           aws.String(asgName),
		ShouldDecrementDesiredCapacity: aws.Bool(true),
		InstanceIds: []*string{
			aws.String(instanceID),
		},
	}

	logging.Info("client/aws: attempting to detach instance %v from "+
		"autoscaling group %v", instanceID, asgName)

	// Detach specified instance from the ASG. Note, this also strips the
	// aws:autoscaling:groupName tag from the instance so it will be hidden
	// from the GetMostRecentInstance method.
	resp, err := svc.DetachInstances(params)
	if err != nil {
		return
	}

	// If the immediate API response does not indicate the detachment has
	// completed successfully, call the checkClusterScalingResult() method which
	// will poll the ASG until it can verify the status.
	if *resp.Activities[0].StatusCode != awsOperationSuccessful {
		err = checkClusterScalingResult(resp.Activities[0].ActivityId, svc)
	}

	return
}

// ScaleInCluster scales the cluster size by -1 by using the DetachInstances
// call to target an instance for removal from the ASG.
func ScaleInCluster(asgName, instanceIP string, nodeCount int, svc *autoscaling.AutoScaling) error {

	instanceID := TranslateIptoID(instanceIP, *svc.Config.Region)

	// Get the current ASG configuration so that we have the basis on which to
	// update to our new desired state.
	asg, err := DescribeScalingGroup(asgName, svc)
	if err != nil {
		return err
	}

	// Evaluate the desired capacity against the current worker count and the
	// MinSize. If the desired capacity -1 is less than the MinSize the AWS call
	// would fail so we bail anyway.
	desiredCap := *asg.AutoScalingGroups[0].DesiredCapacity
	minSize := *asg.AutoScalingGroups[0].MinSize

	if desiredCap != int64(nodeCount) {
		return fmt.Errorf("asg desired capacity %v does not match the current "+
			"Nomad worker pool count %v", desiredCap, nodeCount)
	}

	if desiredCap-int64(1) < minSize {
		return fmt.Errorf("decrementing asg count would violate asg min size of %v",
			minSize)
	}

	// Setup the Input parameters ready for the AWS API call and then trigger the
	// call which will remove the identified instance from the ASG and decrement
	// the capacity to ensure no new instances are launched into the cluster thus
	// preserving the scale-in situation.
	params := &autoscaling.DetachInstancesInput{
		AutoScalingGroupName:           aws.String(asgName),
		ShouldDecrementDesiredCapacity: aws.Bool(true),
		InstanceIds: []*string{
			aws.String(instanceID),
		},
	}

	resp, err := svc.DetachInstances(params)
	if err != nil {
		return err
	}

	// The initial scaling activity StatusCode is available, so we might as well
	// check it before calling checkScalingActivityResult even though its highly
	// unlikely to have already completed.
	if *resp.Activities[0].StatusCode != "Successful" {
		err = checkClusterScalingResult(resp.Activities[0].ActivityId, svc)
	}

	if err != nil {
		return err
	}

	// The instance must now be terminated using the AWS EC2 API.
	err = TerminateInstance(instanceID, *svc.Config.Region)

	if err != nil {
		return fmt.Errorf("an error occurred terminating instance %v", instanceID)
	}

	metrics.IncrCounter([]string{"cluster", "aws", "scale_in"}, 1)
	return nil
}

type mostRecentInstance struct {
	InstanceID       string
	InstanceIP       string
	LaunchTime       time.Time
	MostRecentLaunch time.Time
}

// GetMostRecentInstance identifies the most recently launched instance in a
// specified autoscaling group.
func GetMostRecentInstance(autoscalingGroup, region string) (node string, err error) {
	// Setup struct to track most recent instance information
	instanceTracking := &mostRecentInstance{}

	// Calculate instance launch threshold.
	launchThreshold := time.Now().Add(-90 * time.Second)

	// Setup a ticker to poll the health status of the specified worker node
	// and retry up to a specified timeout.
	ticker := time.NewTicker(time.Second * 10)
	timeout := time.Tick(time.Minute * 5)

	// Setup AWS EC2 API Session
	sess := session.Must(session.NewSession())
	svc := ec2.New(sess, &aws.Config{Region: aws.String(region)})

	// Setup query parameters to find instances that are associated with the
	// specified autoscaling group and are in a running or pending state.
	params := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("tag:aws:autoscaling:groupName"),
				Values: []*string{
					aws.String(autoscalingGroup),
				},
			},
			{
				Name: aws.String("instance-state-name"),
				Values: []*string{
					aws.String("running"),
					aws.String("pending"),
				},
			},
		},
	}

	logging.Info("client/aws: determining most recently launched worker node")

	for {
		select {
		case <-timeout:
			err = fmt.Errorf("client/aws: timeout reached while attempting to " +
				"determine the most recently launched node.")
			logging.Error("%v", err)
			return
		case <-ticker.C:
			// Query the AWS API for worker nodes.
			resp, err := svc.DescribeInstances(params)
			if err != nil {
				logging.Error("client/aws: an error occurred while attempting to "+
					"query instances: %v", err)
				continue
			}

			// If our query returns no instances, raise an error.
			if len(resp.Reservations) == 0 {
				logging.Error("client/aws: failed to retrieve a list of EC2 instances "+
					"in autoscaling group %v", autoscalingGroup)
				continue
			}

			// Iterate over and determine the most recent instance.
			for _, res := range resp.Reservations {
				for _, instance := range res.Instances {
					logging.Debug("client/aws: discovered worker node %v which was "+
						"launched on %v", *instance.InstanceId, instance.LaunchTime)

					if instance.LaunchTime.After(instanceTracking.MostRecentLaunch) {
						instanceTracking.MostRecentLaunch = *instance.LaunchTime
						instanceTracking.InstanceIP = *instance.PrivateIpAddress
						instanceTracking.InstanceID = *instance.InstanceId
					}
				}
			}

			// If the most recent node was launched within the last 90 seconds,
			// we've found what we were looking for otherwise, pause and recheck.
			if instanceTracking.MostRecentLaunch.After(launchThreshold) {
				logging.Debug("client/aws: instance %v is the newest worker node",
					instanceTracking.InstanceID)
				return instanceTracking.InstanceIP, nil
			}

			logging.Debug("client/aws: instance %v is the most recent worker "+
				"node discovered but its launch time %v is not within the last 90 "+
				"seconds. Pausing and will recheck nodes.",
				instanceTracking.InstanceID, instanceTracking.MostRecentLaunch)

		}

	}
}

// checkClusterScalingResult is used to poll the scaling activity and check for
// a successful completion.
func checkClusterScalingResult(activityID *string, svc *autoscaling.AutoScaling) error {

	// Setup our timeout and ticker value. TODO: add a backoff for every call we
	// make where the scaling event has not completed successfully.
	ticker := time.NewTicker(time.Second * time.Duration(10))
	timeOut := time.Tick(time.Minute * 3)

	for {
		select {
		case <-timeOut:
			return fmt.Errorf("timeout %v reached on checking scaling activity "+
				"success", timeOut)
		case <-ticker.C:

			// Make a call to the AWS API every tick to ensure we get the latest Info
			// about the scaling activity status.
			params := &autoscaling.DescribeScalingActivitiesInput{
				ActivityIds: []*string{
					aws.String(*activityID),
				},
			}
			resp, err := svc.DescribeScalingActivities(params)
			if err != nil {
				return err
			}

			// Fail fast; if the scaling activity is in a failed or cancelled state
			// we exit. The final check is to see whether or not we have got the
			// Successful state which indicates a completed scaling activity.
			if *resp.Activities[0].StatusCode == "Failed" || *resp.Activities[0].StatusCode == "Cancelled" {
				return fmt.Errorf("scaling activity %v was unsuccessful ", activityID)
			}
			if *resp.Activities[0].StatusCode == awsOperationSuccessful {
				return nil
			}
		}
	}
}

// TerminateInstance will terminate the supplied EC2 instance and confirm
// successful termination by polling the instance state until the terminated
// status is reached.
func TerminateInstance(instanceID, region string) error {

	// Setup the session and the EC2 service link to use for this operation.
	sess := session.Must(session.NewSession())
	svc := ec2.New(sess, &aws.Config{Region: aws.String(region)})

	tparams := &ec2.TerminateInstancesInput{
		InstanceIds: []*string{
			aws.String(instanceID),
		},
		DryRun: aws.Bool(false),
	}

	// Call the API to terminate the instance.
	logging.Info("client/aws: terminating instance %s", instanceID)
	if _, err := svc.TerminateInstances(tparams); err != nil {
		return err
	}

	// Setup our timeout and ticker value. TODO: add a backoff for every call we
	// make where the termination has not completed successfully.
	ticker := time.NewTicker(time.Second * time.Duration(10))
	timeOut := time.Tick(time.Minute * 3)

	logging.Info("client/aws: confirming successful termination of %s",
		instanceID)

	for {
		select {
		case <-timeOut:
			return fmt.Errorf("timeout %v reached on checking instance %s "+
				"termination", timeOut, instanceID)
		case <-ticker.C:

			// Setup the parameters to call the InstanceStatus endpoint so that we
			// can discover the status of the terminating instance.
			params := &ec2.DescribeInstanceStatusInput{
				DryRun:              aws.Bool(false),
				IncludeAllInstances: aws.Bool(true),
				InstanceIds: []*string{
					aws.String(instanceID),
				},
			}

			resp, err := svc.DescribeInstanceStatus(params)
			if err != nil {
				return err
			}

			if *resp.InstanceStatuses[0].InstanceState.Name == "terminated" {
				logging.Info("client/aws: successful termination of %s confirmed",
					instanceID)

				metrics.IncrCounter([]string{"cluster", "aws", "instance_terminations"}, 1)

				return nil
			}
		}
	}
}

// TranslateIptoID translates the IP address of a node to the EC2 instance ID.
func TranslateIptoID(ip, region string) (id string) {
	sess := session.Must(session.NewSession())
	svc := ec2.New(sess, &aws.Config{Region: aws.String(region)})

	params := &ec2.DescribeInstancesInput{
		DryRun: aws.Bool(false),
		Filters: []*ec2.Filter{
			{
				Name: aws.String("private-ip-address"),
				Values: []*string{
					aws.String(ip),
				},
			},
		},
	}
	resp, err := svc.DescribeInstances(params)

	if err != nil {
		logging.Error("client/aws: unable to convert nomad instance IP to AWS ID: %v", err)
		return
	}

	return *resp.Reservations[0].Instances[0].InstanceId
}
