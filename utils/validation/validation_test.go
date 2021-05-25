package validation_test

import (
	"fmt"
	"reflect"
	"testing"

	ut "github.com/go-playground/universal-translator"

	"github.com/go-playground/validator/v10"

	"github.com/stretchr/testify/require"

	"github.com/allinbits/demeris-backend/utils/validation"
)

type e struct{}

func (e e) Tag() string {
	return "required"
}

func (e e) ActualTag() string {
	return "ActualTag"
}

func (e e) Namespace() string {
	panic("implement me")
}

func (e e) StructNamespace() string {
	panic("implement me")
}

func (e e) Field() string {
	return "Field"
}

func (e e) StructField() string {
	panic("implement me")
}

func (e e) Value() interface{} {
	panic("implement me")
}

func (e e) Param() string {
	panic("implement me")
}

func (e e) Kind() reflect.Kind {
	panic("implement me")
}

func (e e) Type() reflect.Type {
	panic("implement me")
}

func (e e) Translate(ut ut.Translator) string {
	panic("implement me")
}

func (e e) Error() string {
	panic("implement me")
}

func TestMissingFields(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		fieldName bool
		want      []string
	}{
		{
			"not validation error",
			fmt.Errorf("not validation"),
			false,
			nil,
		},
		{
			"validation error",
			validator.ValidationErrors{
				e{},
			},
			false,
			[]string{"Field"},
		},
		{
			"validation error with field name",
			validator.ValidationErrors{
				e{},
			},
			true,
			[]string{"ActualTag"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, validation.MissingFields(tt.err, tt.fieldName))
		})
	}
}

func TestMissingFieldsErr(t *testing.T) {

	tests := []struct {
		name      string
		err       error
		fieldName bool
		want      error
	}{
		{
			"not validation error",
			fmt.Errorf("not validation"),
			false,
			fmt.Errorf("not validation"),
		},
		{
			"validation error",
			validator.ValidationErrors{
				e{},
			},
			false,
			fmt.Errorf("missing fields: Field"),
		},
		{
			"validation error with field name",
			validator.ValidationErrors{
				e{},
			},
			true,
			fmt.Errorf("missing fields: ActualTag"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, validation.MissingFieldsErr(tt.err, tt.fieldName))
		})
	}
}
