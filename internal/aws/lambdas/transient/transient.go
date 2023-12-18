package transient

import (
	"errors"
	"fmt"
	"github.com/weka/go-cloud-lib/protocol"
	"strings"
)

func Handler(terminateResponse protocol.TerminatedInstancesResponse) error {
	errs := terminateResponse.TransientErrors
	if len(errs) > 0 {
		return errors.New(fmt.Sprintf("the following errors were found:\n%s", strings.Join(errs, "\n")))
	}
	return nil
}
