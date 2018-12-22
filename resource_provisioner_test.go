package main

import (
	"github.com/hashicorp/terraform/terraform"
	"testing"
)

func TestProvisioner(t *testing.T) {
	p := Provisioner()

	t.Run("empty-config", func(t *testing.T) {
		c := new(terraform.ResourceConfig)
		c.Config = make(map[string]interface{})
		if _, errs := p.Validate(c); errs == nil || len(errs) < 1 {
			t.Error("did not see expected validation failure")
			return
		}
	})

	t.Run("good-config", func(t *testing.T) {
		m := make(map[string]interface{})
		m[schemaAsgName] = "test-asg"

		c := &terraform.ResourceConfig{Config: m}

		if _, errs := p.Validate(c); errs != nil || len(errs) > 0 {
			t.Error(errs)
			return
		}
	})

	t.Run("missing-asg-name", func(t *testing.T) {
		m := make(map[string]interface{})
		m[schemaRegion] = "test-region"

		c := &terraform.ResourceConfig{Config: m}

		if _, errs := p.Validate(c); errs == nil || len(errs) < 1 {
			t.Error("did not see expected validation failure")
			return
		}
	})

	t.Run("access-key-missing-secret", func(t *testing.T) {
		m := make(map[string]interface{})
		m[schemaAsgName] = "test-asg"
		m[schemaAccessKey] = "AKIAMOCK"

		c := &terraform.ResourceConfig{Config: m}

		if _, errs := p.Validate(c); errs == nil || len(errs) < 1 {
			t.Error("did not see expected validation failure")
			return
		}
	})

	t.Run("bad-pause-duration", func(t *testing.T) {
		m := make(map[string]interface{})
		m[schemaRegion] = "test-region"
		m[schemaAsgName] = "test-asg"
		m[schemaPauseTime] = "abc"

		c := &terraform.ResourceConfig{Config: m}

		if _, errs := p.Validate(c); errs == nil || len(errs) < 1 {
			t.Error("did not see expected validation failure")
			return
		}
	})

	t.Run("bad-new-duration", func(t *testing.T) {
		m := make(map[string]interface{})
		m[schemaRegion] = "test-region"
		m[schemaAsgName] = "test-asg"
		m[schemaASGNewTime] = "abc"

		c := &terraform.ResourceConfig{Config: m}

		if _, errs := p.Validate(c); errs == nil || len(errs) < 1 {
			t.Error("did not see expected validation failure")
			return
		}
	})

	t.Run("valid-duration", func(t *testing.T) {
		m := make(map[string]interface{})
		m[schemaAsgName] = "test-asg"
		m[schemaASGNewTime] = "60m"

		c := &terraform.ResourceConfig{Config: m}

		if _, errs := p.Validate(c); errs != nil || len(errs) > 0 {
			t.Error(errs)
			return
		}
	})
}
