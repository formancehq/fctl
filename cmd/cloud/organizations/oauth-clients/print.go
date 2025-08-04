package oauth_clients

import (
	"fmt"
	"io"

	"github.com/formancehq/fctl/membershipclient"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/pterm/pterm"
)

func onCreateShow(writer io.Writer, client membershipclient.OrganizationClient) error {
	data := [][]string{
		{"Client ID", fmt.Sprintf("organization_%s", client.Id)},
		{"Name", client.Name},
		{"Secret", *client.Secret.Clear},
		{"Secret last digits", client.Secret.LastDigits},
		{"Description", client.Description},
		{"CreatedAt", client.CreatedAt.String()},
	}
	return pterm.DefaultTable.WithHasHeader().WithWriter(writer).WithData(data).Render()
}

func showOrganizationClient(writer io.Writer, client membershipclient.OrganizationClient) error {
	data := [][]string{
		{"Client ID", fmt.Sprintf("organization_%s", client.Id)},
		{"Name", client.Name},
		{"Secret last digits", client.Secret.LastDigits},
		{"Description", client.Description},
		{"CreatedAt", client.CreatedAt.String()},
		{"UpdatedAt", client.UpdatedAt.String()},
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
		{"Client ID", "Name", "Secret last digits", "Description", "CreatedAt", "UpdatedAt"},
	}
	for _, client := range clientsCursor.Data {
		data = append(data, []string{
			fmt.Sprintf("organization_%s", client.Id),
			client.Name,
			client.Secret.LastDigits,
			client.Description,
			client.CreatedAt.String(),
			client.UpdatedAt.String(),
		})
	}
	return pterm.DefaultTable.WithHasHeader().WithWriter(writer).WithData(data).Render()
}
