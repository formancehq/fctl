package oauth_clients

import (
	"fmt"
	"io"

	"github.com/formancehq/fctl/membershipclient"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/pterm/pterm"
)

func showOrganizationClient(writer io.Writer, client membershipclient.OrganizationClient) error {
	data := [][]string{
		{"Client ID", fmt.Sprintf("organization_%s", client.Id)},
		{"Client Secret", func() string {
			if client.Secret.Clear == nil {
				return ""
			}
			return *client.Secret.Clear
		}()},
		{"Description", client.Description},
		{"CreatedAt", client.CreatedAt.String()},
	}
	return pterm.DefaultTable.WithHasHeader().WithWriter(writer).WithData(data).Render()
}

func showOrganizationClients(writer io.Writer, clientsCursor membershipclient.ReadOrganizationClientsResponseData) error {
	cursor := fctl.Cursor{
		HasMore:  clientsCursor.HasMore,
		PageSize: clientsCursor.PageSize,
		Next:     clientsCursor.Next,
		Previous: clientsCursor.Previous,
	}

	if err := fctl.RenderCursor(writer, cursor); err != nil {
		return err
	}

	data := [][]string{
		{"Client ID", "Client Secret", "Description", "CreatedAt"},
	}
	for _, client := range clientsCursor.Data {
		data = append(data, []string{
			fmt.Sprintf("organization_%s", client.Id),
			func() string {
				if client.Secret.Clear == nil {
					return ""
				}
				return *client.Secret.Clear
			}(),
			client.Description,
			client.CreatedAt.String(),
		})
	}
	return pterm.DefaultTable.WithHasHeader().WithWriter(writer).WithData(data).Render()
}
