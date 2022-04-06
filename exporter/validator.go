package exporter

import (
	"fmt"
	"regexp"
	"time"
)

type (
	paramValFunc func(params *Params) error

	ValidationError struct {
		err error
	}
)

var IsAlphaNumeric = regexp.MustCompile(`^[a-zA-Z0-9]*$`).MatchString

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error: %v", e.err)
}

func NewValidationError(err error) ValidationError {
	return ValidationError{err: err}
}

func runParamValidators(p *Params, fns ...paramValFunc) error {
	for _, fn := range fns {
		if err := fn(p); err != nil {
			return err
		}
	}
	return nil
}

func validateSizeLim(p *Params) error {
	if p.SizeLim < 0 || p.SizeLim >= MaxSizeLim {
		return NewValidationError(fmt.Errorf("accepted record file size 1-100MB received %d", p.SizeLim))
	}
	return nil
}

func validateRecordCount(p *Params) error {
	if p.RecordLim < 0 || p.RecordLim > MaxRecordLim {
		return NewValidationError(fmt.Errorf("accepted record count 1-1000000 received %d", p.RecordLim))
	}
	return nil
}

func validateDuration(p *Params) error {
	if p.Duration < 5*time.Second || p.Duration > MaxDuration {
		return NewValidationError(fmt.Errorf("accepted duration 1s-24Hour received %v", p.Duration))
	}
	return nil
}

func validateFileId(p *Params) error {
	if len(p.FileId) > 10 {
		return NewValidationError(fmt.Errorf("accepted max id len 10 received %d", len(p.FileId)))
	}
	if !IsAlphaNumeric(p.FileId) {
		return NewValidationError(fmt.Errorf("accepted characters a-z, A-Z and 0-9, received %s", p.FileId))
	}
	return nil
}

func ValidateParamCombination(p *Params) error {
	// At least one valid param required.
	if p.SizeLim == 0 && p.Duration < 5*time.Second && p.RecordLim == 0 {
		return NewValidationError(fmt.Errorf("invalid param combination"))
	}
	return nil
}
