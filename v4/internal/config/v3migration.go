package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/formancehq/fctl/v4/internal/credentials"
)

type V3State struct {
	Config   V3Config
	Profiles map[string]V3Profile
}

type V3Config struct {
	CurrentProfile string `json:"currentProfile" yaml:"currentProfile"`
}

type V3Profile struct {
	MembershipURI       string          `json:"membershipURI" yaml:"membershipURI"`
	RootTokens          json.RawMessage `json:"rootTokens" yaml:"rootTokens"`
	DefaultOrganization string          `json:"defaultOrganization" yaml:"defaultOrganization"`
	DefaultStack        string          `json:"defaultStack" yaml:"defaultStack"`
}

type MigrationPlan struct {
	CurrentContext  string
	Contexts        map[string]Context
	CredentialMoves []CredentialMove
}

type CredentialMove struct {
	Profile string
	Ref     string
	Value   string
}

func LoadV3State(dir string) (V3State, error) {
	if dir == "" {
		return V3State{}, errors.New("v3 config directory is required")
	}

	configBytes, err := os.ReadFile(filepath.Join(dir, "config.yml"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return V3State{}, fmt.Errorf("read v3 config: %w; --from must point to the fctl v3 config directory containing config.yml and profiles/", err)
		}
		return V3State{}, fmt.Errorf("read v3 config: %w", err)
	}
	var v3Config V3Config
	if err := yaml.Unmarshal(configBytes, &v3Config); err != nil {
		return V3State{}, fmt.Errorf("parse v3 config: %w", err)
	}

	profilesDir := filepath.Join(dir, "profiles")
	entries, err := os.ReadDir(profilesDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return V3State{}, fmt.Errorf("read v3 profiles: %w; --from must point to the fctl v3 config directory containing config.yml and profiles/", err)
		}
		return V3State{}, fmt.Errorf("read v3 profiles: %w", err)
	}

	profiles := map[string]V3Profile{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		profileBytes, err := os.ReadFile(filepath.Join(profilesDir, name, "profile.json"))
		if err != nil {
			return V3State{}, fmt.Errorf("read v3 profile %q: %w", name, err)
		}
		var profile V3Profile
		if err := json.Unmarshal(profileBytes, &profile); err != nil {
			return V3State{}, fmt.Errorf("parse v3 profile %q: %w", name, err)
		}
		profiles[name] = profile
	}

	return V3State{Config: v3Config, Profiles: profiles}, nil
}

func PlanV3Migration(state V3State) (MigrationPlan, error) {
	if len(state.Profiles) == 0 {
		return MigrationPlan{}, errors.New("no v3 profiles found")
	}

	plan := MigrationPlan{
		CurrentContext: state.Config.CurrentProfile,
		Contexts:       map[string]Context{},
	}
	for name, profile := range state.Profiles {
		cloudURL := profile.MembershipURI
		if cloudURL == "" {
			cloudURL = DefaultCloudURL
		}

		kind := ContextKindCloud
		if profile.DefaultOrganization != "" && profile.DefaultStack != "" {
			kind = ContextKindCloudStack
		}

		auth := Auth{Method: AuthMethodCloudDevice}
		if len(profile.RootTokens) > 0 && string(profile.RootTokens) != "null" {
			ref := "keyring://formance/fctl-v4/" + name + "/rootTokens"
			auth.TokenRef = ref
			plan.CredentialMoves = append(plan.CredentialMoves, CredentialMove{
				Profile: name,
				Ref:     ref,
				Value:   string(profile.RootTokens),
			})
		}

		plan.Contexts[name] = Context{
			Kind:         kind,
			CloudURL:     cloudURL,
			Organization: profile.DefaultOrganization,
			Stack:        profile.DefaultStack,
			Auth:         auth,
			API: map[string]string{
				"ledger": string(APIPolicyLatestCompatible),
			},
		}
	}
	if plan.CurrentContext == "" {
		if _, ok := plan.Contexts["default"]; ok {
			plan.CurrentContext = "default"
		}
	}
	if plan.CurrentContext != "" {
		if _, ok := plan.Contexts[plan.CurrentContext]; !ok {
			return MigrationPlan{}, fmt.Errorf("current v3 profile %q has no matching profile", plan.CurrentContext)
		}
	}
	return plan, nil
}

func (p MigrationPlan) Config() Config {
	return Config{
		Version:        Version,
		CurrentContext: p.CurrentContext,
		Contexts:       p.Contexts,
	}
}

func WriteMigration(ctx context.Context, path string, plan MigrationPlan, store credentials.Store) error {
	if len(plan.CredentialMoves) > 0 && store == nil {
		return errors.New("credential store is required to migrate v3 tokens")
	}
	for _, move := range plan.CredentialMoves {
		if err := store.Set(ctx, move.Ref, move.Value); err != nil {
			return fmt.Errorf("store credential for profile %q: %w", move.Profile, err)
		}
	}
	return SaveFile(path, plan.Config())
}
