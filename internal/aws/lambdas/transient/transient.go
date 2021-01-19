package transient

import (
	"errors"
	"fmt"
	"strings"
	"wekactl/internal/aws/lambdas/protocol"
)

func Handler(terminateResponse protocol.TerminatedInstancesResponse) error {
	errs := terminateResponse.TransientErrors
	if len(errs) > 0 {
		return errors.New(fmt.Sprintf("the following errors were found:\n%s", strings.Join(errs, "\n")))
	}
	return nil
}
