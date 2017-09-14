package aws

import (
	"fmt"
	"time"

	metrics "github.com/armon/go-metrics"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/elsevier-core-engineering/replicator/logging"
	"github.com/elsevier-core-engineering/replicator/replicator/structs"
)

// describeScalingGroup returns the current configuration of a worker pool
// autoscaling group.
func describeScalingGroup(asgName string,
	svc *autoscaling.AutoScaling) (
	asg *autoscaling.DescribeAutoScalingGroupsOutput, err error) {

	params := &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{
			aws.String(asgName),
		},
	}
	resp, err := svc.DescribeAutoScalingGroups(params)

	// If we failed to get exactly one ASG, raise an error.
	if len(resp.AutoScalingGroups) != 1 {
		err = fmt.Errorf("the attempt to retrieve the current worker pool "+
			"autoscaling group configuration expected exaclty one result got %v",
			len(asg.AutoScalingGroups))
	}

	return resp, err
}

// getMostRecentInstance monitors a worker pool autoscaling group after a
// scale out operation to identify the newly launched instance.
func getMostRecentInstance(asg, region string) (node string, err error) {
	// Setup struct to track most recent instance information
	instanceTracking := &structs.MostRecentNode{}

	// Calculate instance launch threshold.
	launchThreshold := time.Now().Add(-90 * time.Second)

	// Setup a ticker to poll the autoscaling group for a recently
	// launched instance and retry up to a specified timeout.
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
					aws.String(asg),
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

	logging.Info("provider/aws: determining most recently launched instance "+
		"in autoscaling group %v", asg)

	for {
		select {
		case <-timeout:
			err = fmt.Errorf("provider/aws: timeout reached while attempting to "+
				"determine the most recently launched instance in autoscaling "+
				"group %v", asg)
			logging.Error("%v", err)
			return
		case <-ticker.C:
			// Query the AWS API for worker nodes.
			resp, err := svc.DescribeInstances(params)
			if err != nil {
				logging.Error("provider/aws: an error occurred while attempting to "+
					"retrieve EC2 instance details: %v", err)
				continue
			}

			// If our query returns no instances, raise an error.
			if len(resp.Reservations) == 0 {
				logging.Error("provider/aws: failed to retrieve a list of EC2 "+
					"instances in autoscaling group %v", asg)
				continue
			}

			// Iterate over and determine the most recent instance.
			for _, res := range resp.Reservations {
				for _, instance := range res.Instances {
					logging.Debug("provider/aws: discovered worker node %v which was "+
						"launched on %v", *instance.InstanceId, instance.LaunchTime)

					if instance.LaunchTime.After(instanceTracking.MostRecentLaunch) {
						instanceTracking.MostRecentLaunch = *instance.LaunchTime
						instanceTracking.InstanceIP = *instance.PrivateIpAddress
						instanceTracking.InstanceID = *instance.InstanceId
					}
				}
			}

			// If the most recent node was launched after our launch threshold
			// we've found what we were looking for otherwise, pause and recheck.
			if instanceTracking.MostRecentLaunch.After(launchThreshold) {
				logging.Info("provider/aws: instance %v is the newest instance in "+
					"autoscaling group %v and was launched after the threshold %v",
					instanceTracking.InstanceID, asg, launchThreshold)
				return instanceTracking.InstanceIP, nil
			}

			logging.Debug("provider/aws: instance %v is the most recent instance "+
				"launched in autoscaling group %v but its launch time %v is not "+
				"after the launch threshold %v", instanceTracking.InstanceID, asg,
				instanceTracking.MostRecentLaunch, launchThreshold)
		}

	}
}

// detachInstance is used to detach a specified node from a worker pool
// autoscaling group and automatically decrements the desired count.
func detachInstance(asg, instanceID string,
	svc *autoscaling.AutoScaling) (err error) {

	// Setup the request parameters for the DetachInstances API call.
	params := &autoscaling.DetachInstancesInput{
		AutoScalingGroupName:           aws.String(asg),
		ShouldDecrementDesiredCapacity: aws.Bool(true),
		InstanceIds: []*string{
			aws.String(instanceID),
		},
	}

	logging.Info("provider/aws: attempting to detach instance %v from "+
		"autoscaling group %v", instanceID, asg)

	// Detach specified instance from the ASG. Note, this also strips the
	// aws:autoscaling:groupName tag from the instance so it will be hidden
	// from the getMostRecentInstance method.
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
	if err == nil {
		logging.Info("provider/aws: successfully detached instance %v from "+
			"autoscaling group %v", instanceID, asg)
	}

	return
}

// checkClusterScalingResult is used to poll a worker pool autoscaling group
// to monitor a specified scaling activity for successful completion.
func checkClusterScalingResult(activityID *string,
	svc *autoscaling.AutoScaling) error {

	// Setup our timeout and ticker value.
	ticker := time.NewTicker(time.Second * time.Duration(10))
	timeOut := time.Tick(time.Minute * 3)

	for {
		select {
		case <-timeOut:
			return fmt.Errorf("timeout reached while attempting to verify scaling "+
				"activity %v completed successfully", activityID)

		case <-ticker.C:
			params := &autoscaling.DescribeScalingActivitiesInput{
				ActivityIds: []*string{
					aws.String(*activityID),
				},
			}

			// Check the status of the scaling activity.
			resp, err := svc.DescribeScalingActivities(params)
			if err != nil {
				return err
			}

			if *resp.Activities[0].StatusCode == awsOperationFailed ||
				*resp.Activities[0].StatusCode == awsOperationCancelled {

				return fmt.Errorf("scaling activity %v failed to complete "+
					"successfully", activityID)
			}

			if *resp.Activities[0].StatusCode == awsOperationSuccessful {
				return nil
			}
		}
	}
}

// verifyAsgUpdate validates that a scale out operation against a worker
// pool autoscaling group has completed successfully.
func verifyAsgUpdate(workerPool string, capacity int64,
	svc *autoscaling.AutoScaling) error {

	// Setup a ticker to poll the autoscaling group and report when an instance
	// has been successfully launched.
	ticker := time.NewTicker(time.Millisecond * 500)
	timeout := time.Tick(time.Minute * 3)

	logging.Info("provider/aws: attempting to verify the autoscaling group "+
		"scaling operation for worker pool %v has completed successfully",
		workerPool)

	for {
		select {

		case <-timeout:
			return fmt.Errorf("timeout reached while attempting to verify the "+
				"autoscaling group scaling operation for worker pool %v completed "+
				"successfully", workerPool)

		case <-ticker.C:
			asg, err := describeScalingGroup(workerPool, svc)
			if err != nil {
				logging.Error("provider/aws: an error occurred while attempting to "+
					"verify the autoscaling group operation for worker pool %v: %v",
					workerPool, err)

			} else {
				if int64(len(asg.AutoScalingGroups[0].Instances)) == capacity {
					logging.Info("provider/aws: verified the autoscaling operation "+
						"for worker pool %v has completed successfully", workerPool)

					metrics.IncrCounter([]string{"cluster", "aws", "scale_out"}, 1)
					return nil
				}
			}
		}
	}
}
