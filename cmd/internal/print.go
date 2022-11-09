package internal

import (
	"fmt"
	"io"
	"time"

	"github.com/formancehq/auth/authclient"
	ledgerclient "github.com/numary/ledger/client"
	membershipclient "github.com/numary/membership-api/client"
)

func PrintStackInformation(out io.Writer, profile *Profile, stack *membershipclient.Stack) error {
	baseUrlStr := profile.ServicesBaseUrl(stack.OrganizationId, stack.Id).String()

	fmt.Fprintf(out, "Your dashboard will be reachable on: %s\r\n", baseUrlStr)
	fmt.Fprintln(out, "You can access your sandbox apis using following urls :")
	fmt.Fprintf(out, "Ledger: %s/api/ledger\r\n", baseUrlStr)
	fmt.Fprintf(out, "Payments: %s/api/payments\n", baseUrlStr)
	fmt.Fprintf(out, "Search: %s/api/search\n", baseUrlStr)
	fmt.Fprintf(out, "Auth: %s/api/auth\n", baseUrlStr)

	return nil
}

func PrintOrganization(out io.Writer, o membershipclient.Organization) {
	fmt.Fprintf(out, "Name: %s\r\n", o.Name)
	fmt.Fprintf(out, "Owner ID: %s\r\n", o.OwnerId)
}

func PrintLedgerTransaction(out io.Writer, tx ledgerclient.Transaction) {
	fmt.Fprintf(out, "Date: %s\r\n", tx.Timestamp.Format(time.RFC3339))
	if tx.Reference != nil && *tx.Reference != "" {
		fmt.Fprintf(out, "Reference: %s\r\n", *tx.Reference)
	}
	fmt.Fprintln(out, "Pre commit volumes:")
	for account, v := range *tx.PreCommitVolumes {
		fmt.Fprintf(out, "\tAddress: %s\r\n", account)
		for asset, volumes := range v {
			fmt.Fprintf(out, "\t\tAsset: %s\t\tInput: %f\tOutput: %f\tBalance: %f\r\n",
				asset, volumes.Input, volumes.Output, *volumes.Balance)
		}
	}
	fmt.Fprintln(out, "Post commit volumes:")
	for account, v := range *tx.PostCommitVolumes {
		fmt.Fprintf(out, "\tAddress: %s\r\n", account)
		for asset, volumes := range v {
			fmt.Fprintf(out, "\t\tAsset: %s\t\tInput: %f\tOutput: %f\tBalance: %f\r\n",
				asset, volumes.Input, volumes.Output, *volumes.Balance)
		}
	}
	if len(tx.Metadata) > 0 {
		fmt.Fprintln(out, "Metadata:")
		for k, v := range tx.Metadata {
			fmt.Fprintf(out, "\t- %s: %s\r\n", k, v)
		}
	}
}

func PrintAuthClient(out io.Writer, c authclient.Client) {
	fmt.Fprintf(out, "ID: %s\r\n", c.Id)
	fmt.Fprintf(out, "Name: %s\r\n", c.Name)
	if c.Public != nil && *c.Public {
		fmt.Fprintf(out, "Public: yes\r\n")
	}
	if c.Trusted != nil && *c.Trusted {
		fmt.Fprintf(out, "Trusted: yes\r\n")
	}
	if len(c.Secrets) > 0 {
		fmt.Fprintf(out, "Secrets: \r\n")
		for _, secret := range c.Secrets {
			fmt.Fprintf(out, "\t -\r\n")
			fmt.Fprintf(out, "\t\tID: %s\r\n", secret.Id)
			fmt.Fprintf(out, "\t\tName: %s\r\n", secret.Name)
			fmt.Fprintf(out, "\t\tLastDigits: %s\r\n", secret.LastDigits)
		}
	}
	if len(c.RedirectUris) > 0 {
		fmt.Fprintf(out, "Redirect URIS: \r\n")
		for _, redirectUri := range c.RedirectUris {
			fmt.Fprintf(out, "\t- %s\r\n", redirectUri)
		}
	}
	if len(c.PostLogoutRedirectUris) > 0 {
		fmt.Fprintf(out, "Post Logout Redirect URIS: \r\n")
		for _, postLogoutRedirectUri := range c.PostLogoutRedirectUris {
			fmt.Fprintf(out, "\t- %s\r\n", postLogoutRedirectUri)
		}
	}
	if len(c.Scopes) > 0 {
		fmt.Fprintf(out, "Scopes: \r\n")
		for _, scope := range c.Scopes {
			fmt.Fprintf(out, "\t- %s\r\n", scope)
		}
	}
}

func PrintAuthClientSecret(out io.Writer, c *authclient.Secret) {
	fmt.Fprintf(out, "Name: %s\r\n", c.Name)
	fmt.Fprintf(out, "ID: %s\r\n", c.Id)
	fmt.Fprintf(out, "Last Digits: %s\r\n", c.LastDigits)
	fmt.Fprintf(out, "Clear: %s\r\n", c.Clear)
}
