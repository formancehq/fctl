package deployments

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
		fctl.WithShortDescription("Download a deployment's resolved manifest (yaml) or full configuration archive (gzip)"),
		fctl.WithStringFlag("id", "", "Deployment ID"),
		fctl.WithStringFlag("format", "yaml", "Response format: yaml | gzip"),
		fctl.WithStringFlag("out", "", "Output file path (required for gzip; defaults to stdout for yaml)"),
		fctl.WithRunE(runDeploymentDownload),
	)
}

func runDeploymentDownload(cmd *cobra.Command, _ []string) error {
	id := fctl.GetString(cmd, "id")
	if id == "" {
		return fmt.Errorf("id is required")
	}

	format := fctl.GetString(cmd, "format")
	out := fctl.GetString(cmd, "out")

	var accept operations.AcceptHeaderEnum
	switch format {
	case "yaml":
		accept = operations.AcceptHeaderEnumApplicationYaml
	case "gzip":
		accept = operations.AcceptHeaderEnumApplicationGzip
		if out == "" {
			return fmt.Errorf("--out is required when --format=gzip (binary content cannot be written to a TTY)")
		}
	default:
		return fmt.Errorf("invalid --format %q (expected: yaml | gzip)", format)
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

	resp, err := apiClient.ReadDeployment(
		cmd.Context(),
		id,
		nil,
		operations.WithAcceptHeaderOverride(accept),
	)
	if err != nil {
		return err
	}

	var stream io.ReadCloser
	switch accept {
	case operations.AcceptHeaderEnumApplicationYaml:
		stream = resp.TwoHundredApplicationYamlResponseStream
	case operations.AcceptHeaderEnumApplicationGzip:
		stream = resp.TwoHundredApplicationGzipResponseStream
	}
	if stream == nil {
		return fmt.Errorf("server returned no %s payload for deployment %s", accept, id)
	}
	defer stream.Close()

	if out == "" {
		if _, err := io.Copy(cmd.OutOrStdout(), stream); err != nil {
			return fmt.Errorf("write deployment payload: %w", err)
		}
		return nil
	}

	f, err := os.Create(out) // #nosec G304 -- user-specified output path on a CLI flag
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer func() { _ = f.Close() }()

	if _, err := io.Copy(f, stream); err != nil {
		return fmt.Errorf("write deployment payload: %w", err)
	}
	return f.Close()
}
