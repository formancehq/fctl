package internal

import (
	"fmt"
	"io"

	"github.com/numary/membership-api/client"
)

func PrintStackInformation(out io.Writer, profile *Profile, stack *client.Stack) error {
	baseUrl, err := ServicesBaseUrl(*profile, stack.OrganizationId, stack.Id)
	if err != nil {
		return err
	}
	baseUrlStr := baseUrl.String()

	fmt.Fprintf(out, "Your dashboard will be reachable on: %s\r\n", baseUrlStr)
	fmt.Fprintln(out, "You can access your sandbox apis using following urls :")
	fmt.Fprintf(out, "Ledger: %s/api/ledger\r\n", baseUrlStr)
	fmt.Fprintf(out, "Payments: %s/api/payments\n", baseUrlStr)
	fmt.Fprintf(out, "Search: %s/api/search\n", baseUrlStr)
	fmt.Fprintf(out, "Auth: %s/api/auth\n", baseUrlStr)

	return nil
}
