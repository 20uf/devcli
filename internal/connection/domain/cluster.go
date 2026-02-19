package domain

// Cluster represents an ECS cluster (value object).
// Clusters are identified by their name and are immutable.
type Cluster struct {
	name string
}

// NewCluster creates a new Cluster value object.
func NewCluster(name string) (Cluster, error) {
	if name == "" {
		return Cluster{}, ErrInvalidCluster
	}
	return Cluster{name: name}, nil
}

// Name returns the cluster name.
func (c Cluster) Name() string {
	return c.name
}

// String returns the cluster name (satisfies fmt.Stringer).
func (c Cluster) String() string {
	return c.name
}

// Equal checks if two clusters are equal.
func (c Cluster) Equal(other Cluster) bool {
	return c.name == other.name
}

// MarshalText implements encoding.TextMarshaler for serialization.
func (c Cluster) MarshalText() ([]byte, error) {
	return []byte(c.name), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (c *Cluster) UnmarshalText(text []byte) error {
	var err error
	*c, err = NewCluster(string(text))
	return err
}
