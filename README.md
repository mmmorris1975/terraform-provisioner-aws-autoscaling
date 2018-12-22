AWS Auto Scaling Terraform Provisioner
===
[![Go Report Card](https://goreportcard.com/badge/github.com/mmmorris1975/terraform-provisioner-aws-autoscaling)](https://goreportcard.com/report/github.com/mmmorris1975/terraform-provisioner-aws-autoscaling)

Replace AWS AutoScaling instances similar to the AWS CloudFormation 
[Update Policy](https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-updatepolicy.html) feature.

Overview
---
Changes to attributes of AutoScaling Groups (ASG) which affect the existing EC2 instances are normally not applied at the
time the ASG properties are updated, and will only get applied to new instances.  Sometimes it is desirable
to implement those changes immediately, and this provisioner plugin will update (replace) the EC2 instances in the ASG
during a `terraform apply`.

One thing to note is that this may significantly increase the execution time for `terraform apply`, depending on how the
update parameters are configured, the size of the ASG, and the provisioning time for new instances.

Installation
---
As this is not a built-in provisioner, nor officially supplied by Hashicorp, you must manually install this plugin to 
the correct location for it to work.  The preferred location will be to place the binary in your local `~/.terraform.d/plugins`
directory (`%APPDATA%\terraform.d\plugins` for you Windows-using types), however terraform can look other places.  See the
[terraform documentation](https://www.terraform.io/docs/extend/how-terraform-works.html#plugin-locations) for more info.

Configuration
---
Configuration properties of the provider are not made visible to the provisioner plugin, so it is necessary to repeat
any pertinent provider configuration in the provisioner block in order to properly initialize the AWS client. This
means that at the very least, the `region` parameter will need to be specified for both the provider and provisioner.

The following table describes the configuration properties supported by this provisioner.

| Property | Required | Default | Comment |
:----------|:---------|:--------|:--------|
| asg_name | true     |         | The name of the AutoScaling Group to manage |
| region   | true     |         | The AWS region |
| access_key | false  |         | The AWS access key, if not specified use the SDK default credential lookup chain |
| secret_key | false  |         | The AWS secret key, if not specified use the SDK default credential lookup chain |
| token    | false    |         | The AWS session token, if not specified use the SDK default credential lookup chain |
| profile  | false    |         | The AWS profile name as set in the shared configuration file |
| batch_size | false  | 1       | The maximum number of instances that the provisioner updates in a single pass |
| min_instances_in_service | false | 0 | The minimum number of instances that must be in service within the Auto Scaling group while the provisioner updates old instances |
| pause_time | false  | 0s      | The amount of time the provisioner pauses after making a change to a batch of instances.  Format is golang duration string |
| asg_new_time | false | 2m     | The amount of time after the ASG creation date that the provisioner will consider the ASG new and not execute.  Format is golang duration string |


Usage and Examples
---
While it would be desirable to place the provisioner inside the `aws_autoscaling_group` resource, Terraform provisioners
only execute during resource creation (or destruction), and not for non-destructive resource property updates (ex. `launch_configuration` changes).
This means that for this provisioner to provide any value, you'll need to set it up using a `null_resource` with triggers
based on the autoscaling group property changes you wish to fire the provisioner for.

Example terraform setup:

```hcl-terraform
variable "region" { default = "us-east-2" }
variable "instance_type" { default = "t2.micro" }
variable "vpc_id" {}

provider "aws" {
  region = "${var.region}"
}

data "aws_subnet_ids" "example" {
  vpc_id = "${var.vpc_id}"
}

data "aws_ami" "amzn" {
  most_recent = true
  owners      = ["amazon"]

  filter {
    name   = "state"
    values = ["available"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }

  filter {
    name   = "architecture"
    values = ["x86_64"]
  }

  filter {
    name   = "root-device-type"
    values = ["ebs"]
  }

  filter {
    name   = "name"
    values = ["amzn2-ami-minimal-hvm-*"]
  }
}

resource "aws_launch_configuration" "test" {
  name_prefix   = "tf-asg-provisioner-test-"
  image_id      = "${data.aws_ami.amzn.id}"
  instance_type = "${var.instance_type}"

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_autoscaling_group" "test" {
  name     = "tf-asg-provisioner-test"
  max_size = 5
  min_size = 3

  launch_configuration = "${aws_launch_configuration.test.name}"
  health_check_type    = "EC2"
  vpc_zone_identifier  = ["${data.aws_subnet_ids.example.ids}"]
  default_cooldown     = 60
}

resource "null_resource" "test" {
  triggers {
    lc_name = "${aws_autoscaling_group.test.launch_configuration}"
  }

  provisioner "aws-autoscaling" {
    region     = "${var.region}"
    asg_name   = "${aws_autoscaling_group.test.name}"
    batch_size = 2
    pause_time = "30s"
    min_instances_in_service = 1
  }
}
```

Contributing
---
