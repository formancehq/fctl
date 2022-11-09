package fctl

import (
	"context"
	"fmt"
	"io"

	membershipclient "github.com/numary/membership-api/client"
	"github.com/pkg/errors"
)

func FindOrganizationId(ctx context.Context) (string, error) {
	organization := OrganizationFromContext(ctx)
	if organization == "" {
		apiClient, err := NewMembershipClientFromContext(ctx)
		if err != nil {
			return "", err
		}
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
		organization = organizations.Data[0].Id
	}
	return organization, nil
}

func FindStackId(ctx context.Context, organization string) (string, error) {
	stack := StackFromContext(ctx)
	if stack == "" {
		apiClient, err := NewMembershipClientFromContext(ctx)
		if err != nil {
			return "", err
		}
		stacks, _, err := apiClient.DefaultApi.ListStacks(ctx, organization).Execute()
		if err != nil {
			return "", errors.Wrap(err, "listing stacks")
		}
		if len(stacks.Data) == 0 {
			return "", errors.New("no stacks found")
		}
		if len(stacks.Data) > 1 {
			return "", errors.New("found more than one stack and no stack specified")
		}
		stack = stacks.Data[0].Id
	}
	return stack, nil
}

func FindDefaultStackAndOrganizationId(ctx context.Context) (string, string, error) {
	organization, err := FindOrganizationId(ctx)
	if err != nil {
		return "", "", err
	}

	stack, err := FindStackId(ctx, organization)
	if err != nil {
		return "", "", err
	}
	return organization, stack, nil
}

func PrintOrganization(out io.Writer, o membershipclient.Organization) {
	fmt.Fprintf(out, "Name: %s\r\n", o.Name)
	fmt.Fprintf(out, "Owner ID: %s\r\n", o.OwnerId)
}
