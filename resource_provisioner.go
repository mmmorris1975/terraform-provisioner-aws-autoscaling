package main

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"time"
)

const (
	schemaAsgName    = "asg_name"
	schemaRegion     = "region"
	schemaAccessKey  = "access_key"
	schemaSecretKey  = "secret_key"
	schemaToken      = "token"
	schemaProfile    = "profile"
	schemaBatchSize  = "batch_size"
	schemaMIIS       = "min_instances_in_service"
	schemaPauseTime  = "pause_time"
	schemaASGNewTime = "asg_new_time"
)

// define the provisioner schema and actions
func Provisioner() terraform.ResourceProvisioner {
	return &schema.Provisioner{
		Schema: map[string]*schema.Schema{
			schemaAsgName: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the AutoScaling Group to manage",
			},
			schemaRegion: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The AWS region, if not specified look up value from environment or profile",
			},
			schemaAccessKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The AWS access key, if not specified look up value from environment or profile",
			},
			schemaSecretKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The AWS secret key, if not specified look up value from environment or profile",
			},
			schemaToken: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The AWS session token, if not specified look up value from environment or profile",
			},
			schemaProfile: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The AWS profile name as set in the shared configuration file",
			},
			schemaBatchSize: {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     1,
				Description: "The maximum number of instances that the provisioner updates in a single pass",
			},
			schemaMIIS: {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     0,
				Description: "The minimum number of instances that must be in service within the Auto Scaling group while the provisioner updates old instances",
			},
			schemaPauseTime: {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "0s",
				Description:  "The amount of time the provisioner pauses after making a change to a batch of instances.  Format is golang duration string",
				ValidateFunc: validateDuration,
			},
			schemaASGNewTime: {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "2m",
				Description:  "The amount of time after the ASG creation date that the provisioner will consider the ASG new and not execute.  Format is golang duration string",
				ValidateFunc: validateDuration,
			},
		},

		ApplyFunc: applyFn,
	}
}

func validateDuration(val interface{}, k string) ([]string, []error) {
	_, err := time.ParseDuration(val.(string))
	if err != nil {
		return nil, []error{fmt.Errorf("%s: %s", k, err)}
	}
	return nil, nil
}

func applyFn(ctx context.Context) error {
	out := ctx.Value(schema.ProvOutputKey).(terraform.UIOutput)       // never nil
	cfg := ctx.Value(schema.ProvConfigDataKey).(*schema.ResourceData) // never nil

	h, err := newAsgHandler(cfg)
	if err != nil {
		return err
	}

	asg, err := h.describeAsg()
	if err != nil {
		return err
	}

	// bail out if it looks like this is a new ASG (simple, and possibly unreliable, test)
	if time.Since(*asg.CreatedTime) < h.freshTime {
		out.Output("AutoScalingGroup appears to be new, skipping provisioning")
		return nil
	}

	if h.batchSize > *asg.DesiredCapacity || h.batchSize < 1 {
		h.batchSize = *asg.DesiredCapacity
	}

	ch := make(chan error)
	go h.terminateInstances(ch)
	for e := range ch {
		out.Output(fmt.Sprintf("WARNING: %v", e))
	}

	// TODO support in-place update of ASG instance properties (tag changes)
	// This is going to be tricky, since state.Attributes (the trigger vars) values can only be strings (not lists)
	// The ASG resource allows the user to make tags as individual tag{} blocks, or a tags[], but it doesn't appear
	// that TF provides a common representation/aggregation of the tags.

	return nil
}
