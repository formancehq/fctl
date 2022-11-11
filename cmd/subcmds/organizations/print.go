package organizations

import (
	"fmt"
	"io"

	"github.com/formancehq/fctl/membershipclient"
)

func PrintOrganization(out io.Writer, o membershipclient.Organization) {
	fmt.Fprintf(out, "Name: %s\r\n", o.Name)
	fmt.Fprintf(out, "Owner ID: %s\r\n", o.OwnerId)
}
