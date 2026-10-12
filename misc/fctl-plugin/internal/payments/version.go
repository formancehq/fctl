package payments

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"
)

var paymentSemver = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:[-+][0-9A-Za-z.+-]+)?$`)

var paymentCommit = regexp.MustCompile(`^[0-9a-fA-F]{7,40}$`)

func paymentsVersion(ctx context.Context, client *httpclient.Client) (major, minor int, err error) {
	info, err := client.Do(ctx, http.MethodGet, httpclient.Path("_info"), nil, nil, nil)
	if err != nil {
		return 0, 0, err
	}
	version, err := pluginsdk.ServiceVersion(info)
	if err != nil {
		return 0, 0, err
	}
	match := paymentSemver.FindStringSubmatch(version)
	if match == nil {
		if paymentCommit.MatchString(version) {
			return 3, int(^uint(0) >> 1), nil
		}
		return 0, 0, fmt.Errorf("unrecognized Payments version %q", version)
	}
	major, err = strconv.Atoi(match[1])
	if err != nil {
		return 0, 0, err
	}
	minor, err = strconv.Atoi(match[2])
	if err != nil {
		return 0, 0, err
	}
	if major > 3 {
		return 0, 0, fmt.Errorf("unsupported Payments major %d", major)
	}
	return major, minor, nil
}

func paymentV3Route(command string, major, minor int) (bool, error) {
	if strings.HasPrefix(command, "bank-accounts ") || command == "pools create" || command == "pools latest-balances" {
		return major == 3, nil
	}
	if strings.HasPrefix(command, "orders ") || strings.HasPrefix(command, "conversions ") {
		if major != 3 || minor < 3 {
			return false, fmt.Errorf("%s requires Payments >= 3.3.0", strings.Fields(command)[0])
		}
		return true, nil
	}
	switch command {
	case "tasks get", "transfer-initiation approve", "transfer-initiation reject":
		if major != 3 {
			return false, fmt.Errorf("%s requires Payments >= 3.0.0", command)
		}
		return true, nil
	case "pools update-query":
		if major != 3 || minor < 1 {
			return false, fmt.Errorf("update-query requires Payments >= 3.1.0")
		}
		return true, nil
	case "transfer-initiation update-status":
		if major == 3 {
			return false, fmt.Errorf("update-status is unavailable on Payments 3; use approve or reject")
		}
	}
	return false, nil
}
