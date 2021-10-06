package moduleprocessor

import (
	"errors"
	"fmt"

	sdkutilities "github.com/allinbits/sdk-service-meta/gen/sdk_utilities"
	"github.com/hashicorp/go-multierror"
)

func unwindErrors(err error) error {
	pe, ok := err.(*sdkutilities.ProcessingError)
	if !ok {
		return err
	}

	var errs []error

	for _, e := range pe.Errors {
		errs = append(errs, fmt.Errorf("payload %d: %s", e.PayloadIndex, e.Value))
	}

	baseErr := errors.New("errors encountered during the processing of payloads")
	return multierror.Append(baseErr, errs...)
}
