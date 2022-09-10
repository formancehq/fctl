package cmd

import (
	"context"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

func findOrganizationId(ctx context.Context) (string, error) {
	organization := viper.GetString(organizationFlag)
	if organization == "" {
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

func findStackId(ctx context.Context, organization string) (string, error) {
	stack := viper.GetString(stackFlag)
	if stack == "" {
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

func findDefaultStackAndOrganizationId(ctx context.Context) (string, string, error) {
	organization, err := findOrganizationId(ctx)
	if err != nil {
		return "", "", err
	}

	stack, err := findStackId(ctx, organization)
	if err != nil {
		return "", "", err
	}
	return organization, stack, nil
}
