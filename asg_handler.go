package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/hashicorp/terraform/helper/schema"
	"time"
)

type asgHandler struct {
	client    *autoscaling.AutoScaling
	asgName   string
	activeLC  string
	batchSize int64
	minInSvc  int64
	pauseTime time.Duration
	freshTime time.Duration
}

func newAsgHandler(cfg *schema.ResourceData) (*asgHandler, error) {
	awsOpts := session.Options{SharedConfigState: session.SharedConfigEnable}
	awsCfg := aws.Config{}

	ak := cfg.Get(SchemaAccessKey).(string)
	if len(ak) > 0 {
		sk := cfg.Get(SchemaSecretKey).(string)
		st := cfg.Get(SchemaToken).(string)
		awsCfg.Credentials = credentials.NewStaticCredentials(ak, sk, st)
	}

	p := cfg.Get(SchemaProfile).(string)
	if len(p) > 0 {
		awsOpts.Profile = p
	}

	r := cfg.Get(SchemaRegion).(string)
	if len(r) > 0 {
		awsCfg.Region = &r
	}

	awsOpts.Config = awsCfg
	s, err := session.NewSessionWithOptions(awsOpts)
	if err != nil {
		return nil, err
	}

	// previously passed validation, so we'll skip error checking
	pauseTime, _ := time.ParseDuration(cfg.Get(SchemaPauseTime).(string))
	freshTime, _ := time.ParseDuration(cfg.Get(SchemaASGNewTime).(string))

	h := asgHandler{
		client:    autoscaling.New(s),
		asgName:   cfg.Get(SchemaAsgName).(string),
		batchSize: int64(cfg.Get(SchemaBatchSize).(int)),
		minInSvc:  int64(cfg.Get(SchemaMIIS).(int)),
		pauseTime: pauseTime,
		freshTime: freshTime,
	}

	return &h, nil
}

func (h *asgHandler) describeAsg() (*autoscaling.Group, error) {
	i := autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{aws.String(h.asgName)},
	}

	o, err := h.client.DescribeAutoScalingGroups(&i)
	if err != nil {
		return nil, err
	}

	asg := o.AutoScalingGroups[0]
	h.activeLC = *asg.LaunchConfigurationName

	return asg, nil
}

func (h *asgHandler) terminateInstances(ch chan<- error) {
	defer close(ch)
	batchSize := h.batchSize

	// handle min_instances_in_service, this is over-simplified
	if h.batchSize-h.minInSvc < 1 {
		batchSize = 1
	} else {
		h.batchSize -= h.minInSvc
	}

	i := &autoscaling.DescribeAutoScalingInstancesInput{MaxRecords: &batchSize}
	for {
		r, err := h.client.DescribeAutoScalingInstances(i)
		if err != nil {
			ch <- err
			return
		}

		for _, v := range r.AutoScalingInstances {
			// i.LaunchConfigurationName may be nil when launch config instance was started with was deleted
			if v.LaunchConfigurationName == nil || *v.LaunchConfigurationName != h.activeLC {
				i := &autoscaling.TerminateInstanceInAutoScalingGroupInput{
					InstanceId:                     v.InstanceId,
					ShouldDecrementDesiredCapacity: aws.Bool(false),
				}

				_, err := h.client.TerminateInstanceInAutoScalingGroup(i)
				if err != nil {
					ch <- err
				}
			}
		}

		if r.NextToken == nil {
			break
		} else {
			i.NextToken = r.NextToken
			time.Sleep(h.pauseTime)
		}
	}
}
