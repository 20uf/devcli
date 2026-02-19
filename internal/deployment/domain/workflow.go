package domain

// Workflow represents a GitHub Actions workflow (value object).
// Identified by its name/file path (e.g., "deploy.yml").
type Workflow struct {
	name string // e.g., "deploy.yml" or "deploy"
	id   string // e.g., "12345" or empty if just name
}

// NewWorkflow creates a new Workflow value object.
func NewWorkflow(name string) (Workflow, error) {
	if name == "" {
		return Workflow{}, ErrInvalidWorkflow
	}
	return Workflow{name: name}, nil
}

// NewWorkflowWithID creates a Workflow with both name and ID.
func NewWorkflowWithID(name, id string) (Workflow, error) {
	if name == "" {
		return Workflow{}, ErrInvalidWorkflow
	}
	return Workflow{name: name, id: id}, nil
}

// Name returns the workflow name/file.
func (w Workflow) Name() string {
	return w.name
}

// ID returns the workflow ID (may be empty).
func (w Workflow) ID() string {
	return w.id
}

// String returns the workflow name.
func (w Workflow) String() string {
	return w.name
}

// Equal checks if two workflows are equal.
func (w Workflow) Equal(other Workflow) bool {
	return w.name == other.name && w.id == other.id
}

// MarshalText implements encoding.TextMarshaler.
func (w Workflow) MarshalText() ([]byte, error) {
	return []byte(w.name), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (w *Workflow) UnmarshalText(text []byte) error {
	var err error
	*w, err = NewWorkflow(string(text))
	return err
}
