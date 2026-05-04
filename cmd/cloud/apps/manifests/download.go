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
		fctl.WithShortDescription("Download a manifest's transpiled tar.gz archive of its latest version"),
		fctl.WithStringFlag("id", "", "Manifest ID"),
		fctl.WithStringFlag("out", "", "Output file path (required: gzip is binary)"),
		fctl.WithRunE(runManifestDownload),
	)
}

func runManifestDownload(cmd *cobra.Command, _ []string) error {
	id := fctl.GetString(cmd, "id")
	if id == "" {
		return fmt.Errorf("id is required")
	}

	out := fctl.GetString(cmd, "out")
	if out == "" {
		return fmt.Errorf("--out is required (gzip is binary and cannot be written to a TTY)")
	}

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

	resp, err := apiClient.ReadManifest(
		cmd.Context(),
		id,
		nil,
		operations.WithAcceptHeaderOverride(operations.AcceptHeaderEnumApplicationGzip),
	)
	if err != nil {
		return err
	}

	if resp.ResponseStream == nil {
		return fmt.Errorf("server returned no gzip payload for manifest %s", id)
	}
	defer resp.ResponseStream.Close()

	f, err := os.Create(out) // #nosec G304 -- user-specified output path on a CLI flag
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer func() { _ = f.Close() }()

	if _, err := io.Copy(f, resp.ResponseStream); err != nil {
		return fmt.Errorf("write manifest archive: %w", err)
	}
	return f.Close()
}
