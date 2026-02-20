package oauth_clients

import (
	"fmt"
	"io"

	"github.com/pterm/pterm"

	"github.com/formancehq/fctl/v3/internal/membershipclient/models/components"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

func onCreateShow(writer io.Writer, client components.OrganizationClient) error {
	data := [][]string{
		{"Client ID", fmt.Sprintf("organization_%s", client.GetID())},
		{"Name", client.GetName()},
		{"Secret", func() string {
			secret := client.GetSecret()
			if clear := secret.GetClear(); clear != nil {
				return *clear
			}
			return ""
		}()},
		{"Secret last digits", func() string {
			secret := client.GetSecret()
			return secret.GetLastDigits()
		}()},
		{"Description", client.GetDescription()},
		{"CreatedAt", client.GetCreatedAt().String()},
	}
	return pterm.DefaultTable.WithHasHeader().WithWriter(writer).WithData(data).Render()
}

func showOrganizationClient(writer io.Writer, client components.OrganizationClient) error {
	data := [][]string{
		{"Client ID", fmt.Sprintf("organization_%s", client.GetID())},
		{"Name", client.GetName()},
		{"Secret last digits", func() string {
			secret := client.GetSecret()
			return secret.GetLastDigits()
		}()},
		{"Description", client.GetDescription()},
		{"CreatedAt", client.GetCreatedAt().String()},
		{"UpdatedAt", client.GetUpdatedAt().String()},
	}
	return pterm.DefaultTable.WithHasHeader().WithWriter(writer).WithData(data).Render()
}

func showOrganizationClients(writer io.Writer, clientsCursor components.ReadOrganizationClientsResponseData) error {
	cursor := fctl.Cursor{
		HasMore:  clientsCursor.GetHasMore(),
		PageSize: clientsCursor.GetPageSize(),
		Next:     clientsCursor.GetNext(),
		Previous: clientsCursor.GetPrevious(),
	}

	if err := fctl.RenderCursor(writer, cursor); err != nil {
		return err
	}

	data := [][]string{
		{"Client ID", "Name", "Secret last digits", "Description", "CreatedAt", "UpdatedAt"},
	}
	for _, client := range clientsCursor.GetData() {
		secret := client.GetSecret()
		data = append(data, []string{
			fmt.Sprintf("organization_%s", client.GetID()),
			client.GetName(),
			secret.GetLastDigits(),
			client.GetDescription(),
			client.GetCreatedAt().String(),
			client.GetUpdatedAt().String(),
		})
	}
	return pterm.DefaultTable.WithHasHeader().WithWriter(writer).WithData(data).Render()
}
