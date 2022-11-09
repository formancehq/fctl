package fctl

import (
	"context"
	"fmt"
	"io"

	membershipclient "github.com/numary/membership-api/client"
	"github.com/pkg/errors"
)

func FindOrganizationID(ctx context.Context, apiClient *membershipclient.APIClient) (string, error) {
	organizations, _, err := apiClient.DefaultApi.ListOrganizations(ctx).Execute()
	if err != nil {
		return "", errors.Wrap(err, "listing organizations")
	}
	if len(organizations.Data) == 0 {
		return "", errors.New("no organizations found")
	}
	if len(organizations.Data) > 1 {
		return "", errors.New("found more than one organization and no organization specified")
	}
	return organizations.Data[0].Id, nil
}

func FindStackID(ctx context.Context, apiClient *membershipclient.APIClient, organizationID string) (string, error) {
	stacks, _, err := apiClient.DefaultApi.ListStacks(ctx, organizationID).Execute()
	if err != nil {
		return "", errors.Wrap(err, "listing stacks")
	}
	if len(stacks.Data) == 0 {
		return "", errors.New("no stacks found")
	}
	if len(stacks.Data) > 1 {
		return "", errors.New("found more than one stack and no stack specified")
	}
	return stacks.Data[0].Id, nil
}

func PrintOrganization(out io.Writer, o membershipclient.Organization) {
	fmt.Fprintf(out, "Name: %s\r\n", o.Name)
	fmt.Fprintf(out, "Owner ID: %s\r\n", o.OwnerId)
}
