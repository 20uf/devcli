package domain

// Container represents a Docker container in an ECS task (value object).
// Containers are identified by their name within a task.
type Container struct {
	name string
}

// NewContainer creates a new Container value object.
func NewContainer(name string) (Container, error) {
	if name == "" {
		return Container{}, ErrInvalidContainer
	}
	return Container{name: name}, nil
}

// Name returns the container name.
func (c Container) Name() string {
	return c.name
}

// String returns the container name.
func (c Container) String() string {
	return c.name
}

// Equal checks if two containers are equal.
func (c Container) Equal(other Container) bool {
	return c.name == other.name
}

// MarshalText implements encoding.TextMarshaler.
func (c Container) MarshalText() ([]byte, error) {
	return []byte(c.name), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (c *Container) UnmarshalText(text []byte) error {
	var err error
	*c, err = NewContainer(string(text))
	return err
}

// IsPreferred returns true if this container matches common development container names.
func (c Container) IsPreferred() bool {
	switch c.name {
	case "php", "app", "web", "api":
		return true
	default:
		return false
	}
}
