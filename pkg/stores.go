package fctl

import (
	"context"
	"fmt"
	"net/http"

	"github.com/formancehq/fctl/internal/deployserverclient"
	"github.com/formancehq/fctl/membershipclient"
	v2 "github.com/formancehq/formance-sdk-go/v3"
	"github.com/spf13/cobra"
)

var (
	storeKey           string = "_stores"
	stackKey                  = "_stack"
	membershipKey      string = "_membership"
	orgKey                    = "_membership_organization"
	membershipStackKey        = "_membership_stack"
	deployServerKey           = struct{}{}
)

func GetOrganizationStore(cmd *cobra.Command) *OrganizationStore {
	return GetStore(cmd.Context(), orgKey).(*OrganizationStore)
}

func GetMembershipStackStore(ctx context.Context) *MembershipStackStore {
	return GetStore(ctx, membershipStackKey).(*MembershipStackStore)
}

func ContextWithOrganizationStore(ctx context.Context, store *OrganizationStore) context.Context {
	return ContextWithStore(ctx, orgKey, store)
}

func ContextWithMembershipStackStore(ctx context.Context, store *MembershipStackStore) context.Context {
	return ContextWithStore(ctx, membershipStackKey, store)
}

func ContextWithStore(ctx context.Context, key string, store interface{}) context.Context {
	var stores map[string]interface{}
	stores, ok := ctx.Value(storeKey).(map[string]interface{})
	if !ok {
		stores = map[string]interface{}{}
	}
	stores[key] = store

	return context.WithValue(ctx, storeKey, stores)
}

func GetStore(ctx context.Context, key string) any {
	stores, ok := ctx.Value(storeKey).(map[string]interface{})
	if !ok {
		return nil
	}
	store, ok := stores[key]
	if !ok {
		return nil
	}
	return store
}

type StackStore struct {
	Config         *Config
	stack          *membershipclient.Stack
	stackClient    *v2.Formance
	organizationId string
}

func (cns StackStore) Client() *v2.Formance {
	return cns.stackClient
}

func (cns StackStore) Stack() *membershipclient.Stack {
	return cns.stack
}

func (cns StackStore) OrganizationId() string {
	return cns.organizationId
}

func StackNode(config *Config, stack *membershipclient.Stack, organization string, stackClient *v2.Formance) *StackStore {
	return &StackStore{
		Config:         config,
		stack:          stack,
		organizationId: organization,
		stackClient:    stackClient,
	}
}

func GetStackStore(ctx context.Context) *StackStore {
	return GetStore(ctx, stackKey).(*StackStore)
}

func ContextWithStackStore(ctx context.Context, store *StackStore) context.Context {
	return ContextWithStore(ctx, stackKey, store)
}

func NewStackStore(cmd *cobra.Command) error {
	cfg, err := GetConfig(cmd)
	if err != nil {
		return err
	}
	apiClient, err := NewMembershipClient(cmd, cfg)
	if err != nil {
		return err
	}
	organizationID, err := ResolveOrganizationID(cmd, cfg, apiClient.DefaultAPI)
	if err != nil {
		return err
	}

	stack, err := ResolveStack(cmd, cfg, organizationID)
	if err != nil {
		return err
	}

	stackClient, err := NewStackClient(cmd, cfg, stack)
	if err != nil {
		return err
	}
	cmd.SetContext(ContextWithStackStore(cmd.Context(), StackNode(cfg, stack, organizationID, stackClient)))
	return nil
}

type MembershipStore struct {
	Config           *Config
	MembershipClient *MembershipClient
}

func (cns MembershipStore) Client() *membershipclient.DefaultAPIService {
	return cns.MembershipClient.DefaultAPI
}

func MembershipNode(config *Config, apiClient *MembershipClient) *MembershipStore {
	return &MembershipStore{
		Config:           config,
		MembershipClient: apiClient,
	}
}

func GetMembershipStore(ctx context.Context) *MembershipStore {
	return GetStore(ctx, membershipKey).(*MembershipStore)
}

func ContextWithMembershipStore(ctx context.Context, store *MembershipStore) context.Context {
	return ContextWithStore(ctx, membershipKey, store)
}

func NewMembershipStore(cmd *cobra.Command) error {
	cfg, err := GetConfig(cmd)
	if err != nil {
		return err
	}

	apiClient, err := NewMembershipClient(cmd, cfg)
	if err != nil {
		return err
	}
	cmd.SetContext(ContextWithMembershipStore(cmd.Context(), MembershipNode(cfg, apiClient)))
	return nil
}

func ContextWithDeployServerStore(ctx context.Context, store *DeployServerStore) context.Context {
	return context.WithValue(ctx, deployServerKey, store)
}

func GetDeployServerStore(ctx context.Context) *DeployServerStore {
	return ctx.Value(deployServerKey).(*DeployServerStore)
}

type DeployServerStore struct {
	Cli *deployserverclient.DeployServer
}

func NewDeployServerStore(cmd *cobra.Command) error {
	rt := NewHTTPTransport(cmd, map[string][]string{})
	sdkOptions := []deployserverclient.SDKOption{
		deployserverclient.WithClient(&http.Client{
			Transport: RoundTripperFn(func(req *http.Request) (*http.Response, error) {
				cfg, err := GetConfig(cmd)
				if err != nil {
					return nil, err
				}

				token, err := cfg.GetCurrentProfile().GetToken(cmd.Context(), GetHttpClient(cmd, map[string][]string{}))
				if err != nil {
					return nil, err
				}
				req.Header.Set("Authorization", "Bearer "+token.AccessToken)
				return rt.RoundTrip(req)
			}),
		}),
	}
	uri := GetString(cmd, FrameworkURIFlag)
	if uri == "" {
		return fmt.Errorf("--%s is required", FrameworkURIFlag)
	} else {
		sdkOptions = append(sdkOptions, deployserverclient.WithServerURL(uri))
	}

	cli := deployserverclient.New(sdkOptions...)

	cmd.SetContext(ContextWithDeployServerStore(cmd.Context(), &DeployServerStore{
		Cli: cli,
	}))
	return nil
}
