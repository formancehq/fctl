package manifests

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/operations"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

func NewDownload() *cobra.Command {
	return fctl.NewCommand("download",
		fctl.WithShortDescription("Download a manifest version's raw YAML content"),
		fctl.WithStringFlag("id", "", "Manifest ID"),
		fctl.WithStringFlag("version", "latest", "Version number or \"latest\""),
		fctl.WithStringFlag("out", "", "Output file path (defaults to stdout)"),
		fctl.WithRunE(runManifestDownload),
	)
}

func runManifestDownload(cmd *cobra.Command, _ []string) error {
	id := fctl.GetString(cmd, "id")
	if id == "" {
		return fmt.Errorf("id is required")
	}

	version := fctl.GetString(cmd, "version")
	if version == "" {
		return fmt.Errorf("version is required")
	}

	out := fctl.GetString(cmd, "out")

	cmd.SilenceUsage = true

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return err
	}

	_, apiClient, err := fctl.NewAppDeployClientFromFlags(
		cmd,
		relyingParty,
		fctl.NewPTermDialog(),
		profileName,
		*profile,
	)
	if err != nil {
		return err
	}

	resp, err := apiClient.ReadManifestVersion(
		cmd.Context(),
		id,
		version,
		operations.WithAcceptHeaderOverride(operations.AcceptHeaderEnumApplicationXYaml),
	)
	if err != nil {
		return err
	}

	if resp.ResponseStream == nil {
		return fmt.Errorf("server returned no YAML payload for manifest %s version %s", id, version)
	}
	defer resp.ResponseStream.Close()

	if out == "" {
		if _, err := io.Copy(cmd.OutOrStdout(), resp.ResponseStream); err != nil {
			return fmt.Errorf("write manifest payload: %w", err)
		}
		return nil
	}

	f, err := os.Create(out) // #nosec G304 -- user-specified output path on a CLI flag
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer func() { _ = f.Close() }()

	if _, err := io.Copy(f, resp.ResponseStream); err != nil {
		return fmt.Errorf("write manifest payload: %w", err)
	}
	return f.Close()
}
