package versions

import (
	"fmt"

	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"

	fctl "github.com/formancehq/fctl/v3/pkg"
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

func GetPaymentsVersion(cmd *cobra.Command, _ []string, controller VersionController) error {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return err
	}

	stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return err
	}
	response, err := stackClient.Payments.V1.PaymentsgetServerInfo(cmd.Context())
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
