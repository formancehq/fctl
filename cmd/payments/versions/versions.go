package versions

import (
	"fmt"

	fctl "github.com/formancehq/fctl/pkg"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

type Version int

const (
	V0 Version = iota
	V1
	V2
	V3
)

type VersionController interface {
	SetVersion(Version)
}

func GetPaymentsVersion(cmd *cobra.Command, args []string, controller VersionController) error {
	store := fctl.GetStackStore(cmd.Context())
	response, err := store.Client().Payments.V1.PaymentsgetServerInfo(cmd.Context())
	if err != nil {
		return err
	}

	if response.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	version := "v" + *response.PaymentsServerInfo.Version

	if !semver.IsValid(version) {
		// last version for commits
		controller.SetVersion(V3)
		return nil
	}

	res := semver.Compare(version, "v3.0.0-rc.1")
	switch res {
	case 0, 1:
		controller.SetVersion(V3)
	default:
		controller.SetVersion(V1)
	}

	return nil
}
