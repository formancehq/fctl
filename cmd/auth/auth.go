// Package auth exposes the public Auth API through its generated SDK.
package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	sdk "github.com/formancehq/auth/pkg/client"
	"github.com/formancehq/auth/pkg/client/models/components"
	"github.com/formancehq/auth/pkg/client/models/operations"

	"github.com/formancehq/fctl/v4/internal/command"
)

type response interface {
	GetHTTPMeta() components.HTTPMetadata
}
type operation func(*cobra.Command, *sdk.V1, []string) (response, error)

// NewCommand builds Auth commands using the runtime's endpoint and authenticated HTTP client.
func NewCommand(r command.Runtime) *cobra.Command {
	root := &cobra.Command{Use: "auth", Short: "Manage Auth clients, secrets and users", SilenceUsage: true}
	add := func(parent *cobra.Command, use, help string, nargs int, destructive bool, op operation) *cobra.Command {
		return addCommand(r, parent, use, help, nargs, destructive, op)
	}
	add(root, "info", "Show Auth server information", 0, false, func(cmd *cobra.Command, s *sdk.V1, _ []string) (response, error) {
		return s.GetServerInfo(cmd.Context())
	})
	add(root, "discovery", "Show OpenID Connect discovery configuration", 0, false, func(cmd *cobra.Command, s *sdk.V1, _ []string) (response, error) {
		return s.GetOIDCWellKnowns(cmd.Context())
	})
	clients := &cobra.Command{Use: "clients", Short: "Manage OAuth2 clients"}
	root.AddCommand(clients)
	add(clients, "list", "List OAuth2 clients", 0, false, func(cmd *cobra.Command, s *sdk.V1, _ []string) (response, error) { return s.ListClients(cmd.Context()) })
	add(clients, "show CLIENT", "Show a client and its secret metadata", 1, false, func(cmd *cobra.Command, s *sdk.V1, args []string) (response, error) {
		return s.ReadClient(cmd.Context(), operations.ReadClientRequest{ClientID: url.PathEscape(args[0])})
	})
	create := add(clients, "create", "Create an OAuth2 client from --data JSON", 0, false, createClient)
	bindData(create, false)
	update := add(clients, "update CLIENT", "Update client options, preserving omitted fields", 1, true, updateClient)
	bindData(update, true)
	add(clients, "delete CLIENT", "Delete an OAuth2 client", 1, true, func(cmd *cobra.Command, s *sdk.V1, args []string) (response, error) {
		return s.DeleteClient(cmd.Context(), operations.DeleteClientRequest{ClientID: url.PathEscape(args[0])})
	})
	secrets := &cobra.Command{Use: "secrets", Short: "Manage client secrets (clear values are returned only at creation)"}
	clients.AddCommand(secrets)
	listSecrets := &cobra.Command{Use: "list CLIENT", Short: "List secret metadata for a client", Args: identifierArgs(1), RunE: func(cmd *cobra.Command, args []string) error { return listClientSecrets(r, cmd, args) }}
	secrets.AddCommand(listSecrets)
	secretCreate := add(secrets, "create CLIENT", "Create a secret; save the returned clear value", 1, false, createSecret)
	secretCreate.Flags().String("data", "", "JSON object with name and optional metadata; inline, @file, or - for stdin")
	if err := secretCreate.MarkFlagRequired("data"); err != nil {
		panic(err)
	}
	add(secrets, "delete CLIENT SECRET", "Delete a client secret", 2, true, func(cmd *cobra.Command, s *sdk.V1, args []string) (response, error) {
		return s.DeleteSecret(cmd.Context(), operations.DeleteSecretRequest{ClientID: url.PathEscape(args[0]), SecretID: url.PathEscape(args[1])})
	})
	users := &cobra.Command{Use: "users", Short: "Read Auth users"}
	root.AddCommand(users)
	add(users, "list", "List users", 0, false, func(cmd *cobra.Command, s *sdk.V1, _ []string) (response, error) { return s.ListUsers(cmd.Context()) })
	add(users, "show USER", "Show a user", 1, false, func(cmd *cobra.Command, s *sdk.V1, args []string) (response, error) {
		return s.ReadUser(cmd.Context(), operations.ReadUserRequest{UserID: url.PathEscape(args[0])})
	})
	return root
}

func bindData(cmd *cobra.Command, update bool) {
	cmd.Flags().String("data", "", "JSON client options; inline, @file, or - for stdin")
	if err := cmd.MarkFlagRequired("data"); err != nil {
		panic(err)
	}
	cmd.Long = cmd.Short + ".\nOptions: name, description, public, trusted, redirectUris, postLogoutRedirectUris, scopes, metadata."
	if update {
		cmd.Long += "\nOmitted options are preserved. Explicit false, empty arrays and empty objects clear options. Requires --confirm."
	} else {
		cmd.Long += "\nThe name field is required. Create a secret separately with clients secrets create."
	}
}
func readData(cmd *cobra.Command) (json.RawMessage, error) {
	value, err := cmd.Flags().GetString("data")
	if err != nil {
		return nil, err
	}
	return command.ReadBody(cmd, value)
}
func decodeObject(data json.RawMessage, target any) error {
	if len(bytes.TrimSpace(data)) == 0 || bytes.TrimSpace(data)[0] != '{' {
		return fmt.Errorf("--data must be a JSON object")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid --data: %w", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	for key, value := range fields {
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return fmt.Errorf("--data field %s cannot be null; use false, an empty string, array or object", key)
		}
	}
	return nil
}
func responseBody(res response) (json.RawMessage, error) {
	if res == nil {
		return nil, fmt.Errorf("auth SDK returned no response")
	}
	httpRes := res.GetHTTPMeta().Response
	if httpRes == nil {
		return nil, fmt.Errorf("auth SDK returned no HTTP response")
	}
	if httpRes.Body == nil {
		return json.RawMessage("null"), nil
	}
	body, err := io.ReadAll(io.LimitReader(httpRes.Body, 32<<20+1))
	closeErr := httpRes.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("read Auth response: %w", err)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close Auth response: %w", closeErr)
	}
	if len(body) > 32<<20 {
		return nil, fmt.Errorf("auth response exceeds 32 MiB")
	}
	if httpRes.StatusCode < http.StatusOK || httpRes.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("auth HTTP %d", httpRes.StatusCode)
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return json.RawMessage("null"), nil
	}
	if !json.Valid(body) {
		return nil, fmt.Errorf("auth returned invalid JSON")
	}
	return body, nil
}
func mergeOptions(current, patch json.RawMessage) (json.RawMessage, error) {
	var envelope struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(current, &envelope); err != nil {
		return nil, err
	}
	if envelope.Data == nil {
		return nil, fmt.Errorf("auth response is missing client data")
	}
	fields := make(map[string]json.RawMessage)
	for _, key := range []string{"name", "description", "public", "trusted", "redirectUris", "postLogoutRedirectUris", "scopes", "metadata"} {
		if value, ok := envelope.Data[key]; ok && !bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			fields[key] = value
		}
	}
	var changes map[string]json.RawMessage
	if err := json.Unmarshal(patch, &changes); err != nil {
		return nil, err
	}
	maps.Copy(fields, changes)
	return json.Marshal(fields)
}

func createClient(cmd *cobra.Command, s *sdk.V1, _ []string) (response, error) {
	data, err := readData(cmd)
	if err != nil {
		return nil, err
	}
	var req components.CreateClientRequest
	if err = decodeObject(data, &req); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Name) == "" {
		return nil, fmt.Errorf("--data requires a non-empty name")
	}
	return s.CreateClient(cmd.Context(), &req)
}

func updateClient(cmd *cobra.Command, s *sdk.V1, args []string) (response, error) {
	data, err := readData(cmd)
	if err != nil {
		return nil, err
	}
	var patch components.UpdateClientRequest
	if err = decodeObject(data, &patch); err != nil {
		return nil, err
	}
	current, err := s.ReadClient(cmd.Context(), operations.ReadClientRequest{ClientID: url.PathEscape(args[0])})
	if err != nil {
		return nil, err
	}
	body, err := responseBody(current)
	if err != nil {
		return nil, err
	}
	merged, err := mergeOptions(body, data)
	if err != nil {
		return nil, err
	}
	var req components.UpdateClientRequest
	if err = decodeObject(merged, &req); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Name) == "" {
		return nil, fmt.Errorf("--data requires a non-empty name")
	}
	return s.UpdateClient(cmd.Context(), operations.UpdateClientRequest{ClientID: url.PathEscape(args[0]), UpdateClientRequest: &req})
}

func createSecret(cmd *cobra.Command, s *sdk.V1, args []string) (response, error) {
	data, err := readData(cmd)
	if err != nil {
		return nil, err
	}
	var req components.CreateSecretRequest
	if err = decodeObject(data, &req); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Name) == "" {
		return nil, fmt.Errorf("--data requires a non-empty name")
	}
	return s.CreateSecret(cmd.Context(), operations.CreateSecretRequest{ClientID: url.PathEscape(args[0]), CreateSecretRequest: &req})
}

func listClientSecrets(r command.Runtime, cmd *cobra.Command, args []string) error {
	s, err := newSDK(r, cmd)
	if err != nil {
		return err
	}
	res, err := s.Auth.V1.ReadClient(cmd.Context(), operations.ReadClientRequest{ClientID: url.PathEscape(args[0])})
	if err != nil {
		return err
	}
	body, err := responseBody(res)
	if err != nil {
		return err
	}
	var envelope struct {
		Data *struct {
			Secrets json.RawMessage `json:"secrets"`
		} `json:"data"`
	}
	if err = json.Unmarshal(body, &envelope); err != nil {
		return err
	}
	if envelope.Data == nil {
		return fmt.Errorf("auth response is missing client data")
	}
	if len(envelope.Data.Secrets) == 0 || bytes.Equal(envelope.Data.Secrets, []byte("null")) {
		envelope.Data.Secrets = json.RawMessage("[]")
	}
	return command.WriteJSON(cmd.OutOrStdout(), envelope.Data.Secrets)
}
func addCommand(r command.Runtime, parent *cobra.Command, use, help string, nargs int, destructive bool, op operation) *cobra.Command {
	cmd := &cobra.Command{Use: use, Short: help, Args: identifierArgs(nargs)}
	if destructive {
		cmd.Flags().Bool("confirm", false, "Confirm this destructive operation")
	}
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if destructive {
			confirmed, err := cmd.Flags().GetBool("confirm")
			if err != nil {
				return err
			}
			if !confirmed {
				return fmt.Errorf("%s requires --confirm; review the target and run again with --confirm", cmd.CommandPath())
			}
		}
		client, err := newSDK(r, cmd)
		if err != nil {
			return err
		}
		res, err := op(cmd, client.Auth.V1, args)
		if err != nil {
			return err
		}
		body, err := responseBody(res)
		if err != nil {
			return err
		}
		return command.WriteJSON(cmd.OutOrStdout(), body)
	}
	parent.AddCommand(cmd)
	return cmd
}

func newSDK(r command.Runtime, cmd *cobra.Command) (*sdk.Formance, error) {
	if r.Client == nil {
		return nil, fmt.Errorf("auth connection is not configured")
	}
	conn, err := r.Client(cmd.Context(), "auth")
	if err != nil {
		return nil, err
	}
	if conn == nil || conn.HTTPClient() == nil {
		return nil, fmt.Errorf("auth connection has no HTTP client")
	}
	return sdk.New(sdk.WithServerURL(conn.Endpoint()), sdk.WithClient(conn.HTTPClient())), nil
}

// The SDK interpolates path parameters without escaping; validate IDs before resolution.
func identifierArgs(count int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(count)(cmd, args); err != nil {
			return err
		}
		for _, id := range args {
			if strings.TrimSpace(id) == "" || id == "." || id == ".." {
				return fmt.Errorf("identifier must be non-empty and cannot be a dot or double dot")
			}
		}
		return nil
	}
}
