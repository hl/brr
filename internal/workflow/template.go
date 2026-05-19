package workflow

import (
	"fmt"
	"os"
)

func InitTemplate(name, template string) error {
	if err := validateName(name); err != nil {
		return err
	}
	if template != "ship" {
		return fmt.Errorf("unknown workflow template %q (available: ship)", template)
	}
	target := workflowPath(name)
	if err := rejectUnsafeExisting(target); err != nil {
		return err
	}
	if err := ensureWorkflowDir(); err != nil {
		return fmt.Errorf("creating .brr/workflows: %w", err)
	}
	if _, err := os.Lstat(target); err == nil {
		return fmt.Errorf("%s already exists", target)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("checking %s: %w", target, err)
	}
	return atomicWriteRegularFile(target, []byte(ShipTemplate), 0o644)
}

const ShipTemplate = `version: 2
description: "requirements -> verified, reviewed code"

defaults:
  profile: claude
  max: 3

cycle:
  target: build
  max: 3

stages:
  - id: spec
    type: agent
    prompt: spec
    max: 3

  - id: plan
    type: agent
    prompt: plan
    max: 5

  - id: build
    type: agent
    prompt: build
    max: 100

  - id: check
    type: command
    command: ["make", "check"]

  - id: verify
    type: agent
    prompt: verify
    max: 3

  - id: review
    type: agent
    prompt: review
    max: 1
`
