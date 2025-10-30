package internal

import (
	"fmt"
	"strconv"

	"github.com/formancehq/fctl/membershipclient"
	"github.com/pterm/pterm"
)

func PrintOrganization(organization *membershipclient.OrganizationExpanded) error {
	pterm.DefaultSection.Println("Organization")

	data := [][]string{
		{"ID", organization.Id},
		{"Name", organization.Name},
		{"Domain", func() string {
			if organization.Domain == nil {
				return ""
			}
			return *organization.Domain
		}()},
		{"Default Policy", func() string {
			if !organization.DefaultPolicyID.IsSet() {
				return "None"
			}
			return fmt.Sprintf("%d", *organization.DefaultPolicyID.Get())
		}()},
	}

	if organization.Owner != nil {
		data = append(data, []string{"Owner", organization.Owner.Email})
	}

	if organization.TotalUsers != nil {
		data = append(data, []string{"Total Users", strconv.Itoa(int(*organization.TotalUsers))})
	}

	if organization.TotalStacks != nil {
		data = append(data, []string{"Total Stacks", strconv.Itoa(int(*organization.TotalStacks))})
	}

	return pterm.DefaultTable.WithHasHeader().WithData(data).Render()
}
