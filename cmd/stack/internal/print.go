package internal

import (
	"fmt"
	"io"

	"github.com/formancehq/fctl/internal/membershipclient/models/components"

	fctl "github.com/formancehq/fctl/pkg"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"
	"github.com/iancoleman/strcase"
	"github.com/pterm/pterm"
)

func PrintStackInformation(out io.Writer, stack *components.Stack, versions *shared.GetVersionsResponse) error {
	err := printInformation(out, stack)

	if err != nil {
		return err
	}

	if versions != nil {
		err = printVersion(out, stack.URI, versions)

		if err != nil {
			return err
		}
	}

	err = printMetadata(out, stack)
	if err != nil {
		return err
	}

	return nil
}

func printInformation(out io.Writer, stack *components.Stack) error {

	fctl.Section.WithWriter(out).Println("Information")
	tableData := pterm.TableData{}
	tableData = append(tableData, []string{pterm.LightCyan("ID"), stack.ID, ""})
	tableData = append(tableData, []string{pterm.LightCyan("Name"), stack.Name, ""})
	tableData = append(tableData, []string{pterm.LightCyan("Region"), stack.RegionID, ""})
	tableData = append(tableData, []string{pterm.LightCyan("Status"), string(stack.State), ""})
	tableData = append(tableData, []string{pterm.LightCyan("Effective status"), string(stack.Status), ""})

	if stack.AuditEnabled != nil {
		tableData = append(tableData, []string{pterm.LightCyan("Audit enabled"), fctl.BoolPointerToString(stack.AuditEnabled), ""})
	}

	return pterm.DefaultTable.
		WithWriter(out).
		WithData(tableData).
		Render()
}

func printVersion(out io.Writer, url string, versions *shared.GetVersionsResponse) error {
	fctl.Println()
	fctl.Section.WithWriter(out).Println("Versions")

	tableData := pterm.TableData{}

	for _, service := range versions.Versions {
		tableData = append(tableData, []string{pterm.LightCyan(strcase.ToCamel(service.Name)), service.Version,
			fmt.Sprintf("%s/api/%s", url, service.Name)})
	}

	return pterm.DefaultTable.
		WithWriter(out).
		WithData(tableData).
		Render()
}

func printMetadata(out io.Writer, stack *components.Stack) error {
	fctl.Println()
	fctl.Section.WithWriter(out).Println("Metadata")

	tableData := pterm.TableData{}

	if stack.Metadata != nil {
		for k, v := range stack.Metadata {
			tableData = append(tableData, []string{pterm.LightCyan(k), v})
		}
	}

	return pterm.DefaultTable.
		WithWriter(out).
		WithData(tableData).
		Render()
}
