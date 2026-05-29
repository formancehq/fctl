package views

import (
	"io"

	"github.com/pterm/pterm"

	"github.com/formancehq/formance-sdk-go/v4/pkg/models/auth"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func PrintSecrets(out io.Writer, secrets []auth.ClientSecret) error {
	fctl.Section.WithWriter(out).Println("Secrets :")

	return pterm.DefaultTable.
		WithWriter(out).
		WithHasHeader(true).
		WithData(fctl.Prepend(
			fctl.Map(secrets, func(secret auth.ClientSecret) []string {
				return []string{
					secret.ID, secret.Name, secret.LastDigits,
				}
			}),
			[]string{"ID", "Name", "Last digits"},
		)).
		Render()
}
