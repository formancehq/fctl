package versions

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/deployserverclient/models/operations"
	fctl "github.com/formancehq/fctl/pkg"
)

type ManifestCtrl struct {
	store any
}

var _ fctl.Controller[any] = (*ManifestCtrl)(nil)

func NewManifestCtrl() *ManifestCtrl {
	return &ManifestCtrl{
		store: nil,
	}
}

func NewManifest() *cobra.Command {
	return fctl.NewCommand("show-manifest",
		fctl.WithShortDescription("Manifest versions for an app"),
		fctl.WithStringFlag("id", "", "App ID"),
		fctl.WithController(NewManifestCtrl()),
	)
}

func (c *ManifestCtrl) GetStore() any {
	return c.store
}

func (c *ManifestCtrl) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	cfg, err := fctl.LoadConfig(cmd)
	if err != nil {
		return nil, err
	}

	profile, profileName, err := fctl.LoadCurrentProfile(cmd, *cfg)
	if err != nil {
		return nil, err
	}

	relyingParty, err := fctl.GetAuthRelyingParty(cmd.Context(), fctl.GetHttpClient(cmd), profile.MembershipURI)
	if err != nil {
		return nil, err
	}

	organizationID, err := fctl.ResolveOrganizationID(cmd, *profile)
	if err != nil {
		return nil, err
	}

	apiClient, err := fctl.NewAppDeployClient(
		cmd,
		relyingParty,
		fctl.NewPTermDialog(),
		profileName,
		*profile,
		organizationID,
	)
	if err != nil {
		return nil, err
	}
	id := fctl.GetString(cmd, "id")
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	versions, err := apiClient.ReadVersion(cmd.Context(), id, operations.WithAcceptHeaderOverride(operations.AcceptHeaderEnumApplicationYaml))
	if err != nil {
		return nil, err
	}
	defer versions.TwoHundredApplicationYamlResponseStream.Close()
	data, err := io.ReadAll(versions.TwoHundredApplicationYamlResponseStream)
	if err != nil {
		return nil, err
	}

	c.store = string(data)
	return c, nil
}

func (c *ManifestCtrl) Render(cmd *cobra.Command, args []string) error {
	fmt.Println(c.store)
	return nil
}
