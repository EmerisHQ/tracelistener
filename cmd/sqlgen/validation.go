package main

import (
	"fmt"
	"unicode"
)

const (
	maxNameLen = 59
)

func validateName(name string) error {
	if len(name) == 0 {
		return fmt.Errorf("cannot be empty")
	}

	for i, r := range name {
		if i == 0 && !unicode.IsLetter(r) && r != '_' {
			return fmt.Errorf("must start with a letter or underscore")
		}

		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return fmt.Errorf("must be alphanumeric with underscores")
		}
	}

	if len(name) >= maxNameLen {
		return fmt.Errorf("exceed maximum length of %d chars", maxNameLen)
	}

	return nil
}
