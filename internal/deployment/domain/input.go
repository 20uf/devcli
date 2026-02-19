package domain

import "fmt"

// InputType represents the type of a workflow input.
type InputType string

const (
	InputTypeString   InputType = "string"
	InputTypeBoolean  InputType = "boolean"
	InputTypeChoice   InputType = "choice"
	InputTypeUnknown  InputType = "unknown"
)

// Input represents a typed workflow input (value object).
// Inputs can be:
// - string: any text value
// - boolean: true/false
// - choice: one of a predefined list
type Input struct {
	key       string
	inputType InputType
	value     string        // The actual value provided by user
	required  bool
	options   []string      // For choice type: allowed values
}

// NewInput creates a new typed Input value object.
func NewInput(key string, inputType InputType, value string, required bool) (Input, error) {
	if key == "" {
		return Input{}, ErrInvalidInput
	}

	return Input{
		key:       key,
		inputType: inputType,
		value:     value,
		required:  required,
	}, nil
}

// NewChoiceInput creates a choice-type input with options.
func NewChoiceInput(key string, value string, options []string, required bool) (Input, error) {
	if key == "" {
		return Input{}, ErrInvalidInput
	}

	input := Input{
		key:       key,
		inputType: InputTypeChoice,
		value:     value,
		required:  required,
		options:   options,
	}

	// Validate that value is in options
	if value != "" {
		if !input.isValidChoice() {
			return Input{}, ErrInputValidationFailed
		}
	}

	return input, nil
}

// Key returns the input key/name.
func (i Input) Key() string {
	return i.key
}

// Type returns the input type.
func (i Input) Type() InputType {
	return i.inputType
}

// Value returns the input value.
func (i Input) Value() string {
	return i.value
}

// IsRequired returns if this input is required.
func (i Input) IsRequired() bool {
	return i.required
}

// Options returns the choice options (only for choice type).
func (i Input) Options() []string {
	return i.options
}

// Validate checks if the input value is valid for its type.
func (i Input) Validate() error {
	if i.required && i.value == "" {
		return ErrMissingRequiredInput
	}

	switch i.inputType {
	case InputTypeBoolean:
		return i.validateBoolean()
	case InputTypeChoice:
		if i.value != "" && !i.isValidChoice() {
			return ErrInputValidationFailed
		}
	case InputTypeString:
		// Any string is valid
	case InputTypeUnknown:
		// Unknown type, just use as string
	}

	return nil
}

// validateBoolean checks if value is a valid boolean.
func (i Input) validateBoolean() error {
	if i.value == "" {
		return nil // Empty is OK for optional booleans
	}
	switch i.value {
	case "true", "false", "yes", "no", "1", "0":
		return nil
	default:
		return ErrInputTypeMismatch
	}
}

// isValidChoice checks if value is in the options list.
func (i Input) isValidChoice() bool {
	for _, opt := range i.options {
		if opt == i.Value() {
			return true
		}
	}
	return false
}

// SetValue updates the input value with validation.
func (i *Input) SetValue(value string) error {
	i.value = value
	return i.Validate()
}

// String returns a human-readable representation.
func (i Input) String() string {
	return fmt.Sprintf("%s=%s (type:%s)", i.key, i.value, i.inputType)
}
