package domain

// Service represents an ECS service (value object).
// A service is identified by its name within a cluster.
type Service struct {
	name string
}

// NewService creates a new Service value object.
func NewService(name string) (Service, error) {
	if name == "" {
		return Service{}, ErrInvalidService
	}
	return Service{name: name}, nil
}

// Name returns the service name.
func (s Service) Name() string {
	return s.name
}

// String returns the service name.
func (s Service) String() string {
	return s.name
}

// Equal checks if two services are equal.
func (s Service) Equal(other Service) bool {
	return s.name == other.name
}

// MarshalText implements encoding.TextMarshaler.
func (s Service) MarshalText() ([]byte, error) {
	return []byte(s.name), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (s *Service) UnmarshalText(text []byte) error {
	var err error
	*s, err = NewService(string(text))
	return err
}
