package domain

import (
	"errors"
	"time"
)

// Deployment represents an intended deployment execution (aggregate root).
// It encapsulates all information needed to trigger and track a workflow run.
// This is the entry point for the deployment domain logic.
type Deployment struct {
	id        string
	workflow  Workflow
	inputs    []Input     // All inputs for this deployment
	branch    string      // Branch to run on
	run       *Run        // The actual run (populated after trigger)
	createdAt time.Time
	url       string // GitHub repo URL
}

// NewDeployment creates a new Deployment aggregate.
func NewDeployment(
	id string,
	workflow Workflow,
	branch string,
	url string,
) (Deployment, error) {
	if id == "" {
		return Deployment{}, errors.New("deployment id is required")
	}
	if branch == "" {
		return Deployment{}, errors.New("branch is required")
	}

	return Deployment{
		id:        id,
		workflow:  workflow,
		branch:    branch,
		url:       url,
		inputs:    []Input{},
		createdAt: time.Now(),
	}, nil
}

// ID returns the deployment identifier.
func (d Deployment) ID() string {
	return d.id
}

// Workflow returns the target workflow.
func (d Deployment) Workflow() Workflow {
	return d.workflow
}

// Branch returns the target branch.
func (d Deployment) Branch() string {
	return d.branch
}

// URL returns the GitHub repo URL.
func (d Deployment) URL() string {
	return d.url
}

// CreatedAt returns when the deployment was created.
func (d Deployment) CreatedAt() time.Time {
	return d.createdAt
}

// Inputs returns all deployment inputs.
func (d Deployment) Inputs() []Input {
	return d.inputs
}

// Run returns the actual run (if triggered).
func (d Deployment) Run() *Run {
	return d.run
}

// HasRun checks if a run has been triggered.
func (d Deployment) HasRun() bool {
	return d.run != nil
}

// AddInput adds a typed input to the deployment.
// Validates that input is correct for its type.
func (d *Deployment) AddInput(input Input) error {
	if err := input.Validate(); err != nil {
		return err
	}

	// Check for duplicate keys
	for i, existing := range d.inputs {
		if existing.Key() == input.Key() {
			d.inputs[i] = input // Replace existing
			return nil
		}
	}

	d.inputs = append(d.inputs, input)
	return nil
}

// GetInput retrieves an input by key.
func (d Deployment) GetInput(key string) *Input {
	for _, input := range d.inputs {
		if input.Key() == key {
			return &input
		}
	}
	return nil
}

// SetInputValue updates an input's value by key.
func (d *Deployment) SetInputValue(key string, value string) error {
	for i, input := range d.inputs {
		if input.Key() == key {
			if err := input.SetValue(value); err != nil {
				return err
			}
			d.inputs[i] = input
			return nil
		}
	}
	return ErrInvalidInput
}

// ValidateInputs checks that all required inputs are provided.
func (d Deployment) ValidateInputs() error {
	for _, input := range d.inputs {
		if err := input.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// SetRun associates a run with this deployment.
// Called after the workflow is triggered.
func (d *Deployment) SetRun(run Run) {
	d.run = &run
}

// BuildInputsMap returns all inputs as a key-value map for GitHub API.
func (d Deployment) BuildInputsMap() map[string]string {
	result := make(map[string]string)
	for _, input := range d.inputs {
		result[input.Key()] = input.Value()
	}
	return result
}

// String returns a human-readable representation.
func (d Deployment) String() string {
	if d.run != nil {
		return d.workflow.Name() + " → " + d.run.String()
	}
	return d.workflow.Name() + " (pending)"
}

// Summary returns a detailed description of the deployment.
func (d Deployment) Summary() string {
	summary := "Workflow: " + d.workflow.Name() + "\n"
	summary += "Branch: " + d.branch + "\n"

	if len(d.inputs) > 0 {
		summary += "Inputs:\n"
		for _, input := range d.inputs {
			summary += "  - " + input.String() + "\n"
		}
	}

	if d.run != nil {
		summary += "Run: " + d.run.String() + " (" + string(d.run.Status()) + ")\n"
	}

	return summary
}
