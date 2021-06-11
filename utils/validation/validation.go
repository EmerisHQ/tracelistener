package validation

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/accounts"

	"github.com/gin-gonic/gin/binding"

	"github.com/go-playground/validator/v10"
)

// MissingFields returns a slice of strings containing the names of the fields marked as required and not provided,
// extrapolated by err as validator.ValidationErrors.
// If fieldName is true, the JSON tag name will be provided instead of the struct field name.
// If err is not of the validator.ValidationErrors kind, a nil slice will be returned.
func MissingFields(err error, fieldName bool) []string {
	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		return nil
	}

	var missingFields []string
	for _, e := range ve {
		switch e.Tag() {
		case "required":
			field := e.Field()
			if fieldName {
				field = e.ActualTag()
			}

			missingFields = append(missingFields, field)
		}
	}

	return missingFields
}

// MissingFieldsErr formats the output of MissingFields in an error.
// If err is not of the validator.ValidationErrors kind, the original error will be returned.
func MissingFieldsErr(err error, fieldName bool) error {
	f := MissingFields(err, fieldName)
	if f == nil {
		return err
	}

	return fmt.Errorf("missing fields: %v", strings.Join(f, ","))
}

func jsonTag(fld reflect.StructField) string {
	name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]

	if name == "-" {
		return ""
	}

	return name
}

// JSONFields adds a function to retrieve JSON tag names on StructValidator
// errors.
// It is mostly used alongside gin.
func JSONFields(structValidator binding.StructValidator) {
	if v, ok := structValidator.Engine().(*validator.Validate); ok {
		v.RegisterTagNameFunc(jsonTag)
	}
}

func DerivationPath(structValidator binding.StructValidator) {
	if v, ok := structValidator.Engine().(*validator.Validate); ok {
		if err := v.RegisterValidation("derivationpath", func(fl validator.FieldLevel) bool {
			path, ok := fl.Field().Interface().(string)
			if !ok {
				return false
			}

			if err := validateDerivationPath(path); err != nil {
				return false
			}

			return true
		}); err != nil {
			panic(err)
		}
	}
}
func validateDerivationPath(dp string) error {
	if _, err := accounts.ParseDerivationPath(dp); err != nil {
		return err
	}

	return nil
}
