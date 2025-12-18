package versions

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"

	fctl "github.com/formancehq/fctl/pkg"
)

type PaymentMajorVersion int

const (
	V0 PaymentMajorVersion = iota
	V1
	V2
	V3
)

type Version struct {
	Major PaymentMajorVersion
	Minor int
	Raw   string
}

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

	paymentVersion, err := computePaymentVersion(version)
	if err != nil {
		return err
	}

	controller.SetVersion(*paymentVersion)
	return nil
}

func computePaymentVersion(rawVersion string) (*Version, error) {
	semverVersion := semver.MajorMinor(rawVersion)
	if semverVersion == "" {
		// we assume the version is a commit id
		// thus corresponds to the latest possible version
		return &Version{Major: V3, Minor: math.MaxInt, Raw: rawVersion}, nil
	}

	parts := strings.Split(semver.Canonical(semverVersion), ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("expected both major and minor for version string: %s", rawVersion)
	}

	var major PaymentMajorVersion
	minor, _ := strconv.Atoi(parts[1])

	switch parts[0] {
	case "v0", "v1", "v2":
		major = V1
	case "v3":
		major = V3
	default:
		return nil, fmt.Errorf("invalid major version string: %s", rawVersion)
	}

	return &Version{
		Major: major,
		Minor: minor,
		Raw:   rawVersion,
	}, nil
}
