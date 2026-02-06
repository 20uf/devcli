package ui

import (
	"github.com/AlecAivazis/survey/v2"
)

// Select displays an interactive selection prompt with fuzzy search.
func Select(label string, options []string) (string, error) {
	var selected string

	prompt := &survey.Select{
		Message:  label,
		Options:  options,
		PageSize: 20,
	}

	err := survey.AskOne(prompt, &selected)
	if err != nil {
		return "", err
	}

	return selected, nil
}
