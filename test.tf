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
