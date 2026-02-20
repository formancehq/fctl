package views

import (
	"io"

	"github.com/pterm/pterm"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func PrintSecrets(out io.Writer, secrets []shared.ClientSecret) error {
	fctl.Section.WithWriter(out).Println("Secrets :")

	return pterm.DefaultTable.
		WithWriter(out).
		WithHasHeader(true).
		WithData(fctl.Prepend(
			fctl.Map(secrets, func(secret shared.ClientSecret) []string {
				return []string{
					secret.ID, secret.Name, secret.LastDigits,
				}
			}),
			[]string{"ID", "Name", "Last digits"},
		)).
		Render()
}
