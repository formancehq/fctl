package fctl

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/formancehq/go-libs/v3/oidc/client"
)

type ErrInvalidAuthentication struct {
	err error
}

func (e ErrInvalidAuthentication) Error() string {
	return e.err.Error()
}

func (e ErrInvalidAuthentication) Unwrap() error {
	return e.err
}

func (e ErrInvalidAuthentication) Is(err error) bool {
	_, ok := err.(*ErrInvalidAuthentication)
	return ok
}

func IsInvalidAuthentication(err error) bool {
	return errors.Is(err, &ErrInvalidAuthentication{})
}

func newErrInvalidAuthentication(err error) *ErrInvalidAuthentication {
	return &ErrInvalidAuthentication{
		err: err,
	}
}

const AuthClient = "fctl"

type Profile struct {
	MembershipURI string  `json:"membershipURI"`
	RootTokens    *Tokens `json:"rootTokens"`

	DefaultOrganization string `json:"defaultOrganization"`
	DefaultStack        string `json:"defaultStack"`
}

func (p *Profile) UpdateRootToken(tokens *Tokens) {
	p.RootTokens = tokens
}

func (p *Profile) GetMembershipURI() string {
	return p.MembershipURI
}

func (p *Profile) GetDefaultOrganization() string {
	return p.DefaultOrganization
}

func (p *Profile) GetDefaultStack() string {
	return p.DefaultStack
}

func (p *Profile) GetRootToken() (*AccessToken, error) {
	return &p.RootTokens.Access, nil
}

func (p *Profile) GetClaims() (AccessTokenClaims, error) {
	return p.RootTokens.Access.Claims, nil
}

func (p *Profile) SetDefaultOrganization(o string) {
	p.DefaultOrganization = o
}

func (p *Profile) IsConnected() bool {
	return p.RootTokens != nil
}

func LoadCurrentProfile(cmd *cobra.Command, cfg Config) (*Profile, string, error) {
	currentProfileName := GetCurrentProfileName(cmd, cfg)
	profile, err := LoadProfile(cmd, currentProfileName)
	if errors.Is(err, &fs.PathError{}) || errors.Is(err, os.ErrNotExist) {
		return &Profile{
			MembershipURI: DefaultMembershipURI,
		}, currentProfileName, nil
	}

	return profile, currentProfileName, err
}

func LoadAndAuthenticateCurrentProfileWithConfig(cmd *cobra.Command, cfg Config) (*Profile, string, client.RelyingParty, error) {

	profile, profileName, err := LoadCurrentProfile(cmd, cfg)
	if err != nil {
		return nil, "", nil, err
	}

	relyingParty, err := GetAuthRelyingParty(cmd.Context(), GetHttpClient(cmd), profile.GetMembershipURI())
	if err != nil {
		return nil, "", nil, err
	}

	if !profile.IsConnected() {
		return nil, "", nil, newErrInvalidAuthentication(errors.New("not authenticated, please run 'fctl login'"))
	}

	return profile, profileName, relyingParty, nil
}

func LoadAndAuthenticateCurrentProfile(cmd *cobra.Command) (*Config, *Profile, string, client.RelyingParty, error) {
	cfg, err := LoadConfig(cmd)
	if err != nil {
		return nil, nil, "", nil, err
	}

	profile, profileName, relyingParty, err := LoadAndAuthenticateCurrentProfileWithConfig(cmd, *cfg)
	if err != nil {
		return nil, nil, "", nil, err
	}

	return cfg, profile, profileName, relyingParty, nil
}

type CurrentProfile Profile

func ListProfiles(cmd *cobra.Command, filters ...func(string) bool) ([]string, error) {

	ret := make([]string, 0)

	dir, err := os.ReadDir(GetFilePath(cmd, "profiles"))
	if err != nil {
		return nil, err
	}

l:
	for _, d := range dir {
		if d.IsDir() {
			name := d.Name()
			for _, filter := range filters {
				if !filter(name) {
					continue l
				}
			}

			ret = append(ret, name)
		}
	}

	sort.Strings(ret)
	return ret, nil
}

func LoadProfile(cmd *cobra.Command, name string) (*Profile, error) {
	return ReadJSONFile[Profile](cmd, filepath.Join("profiles", name, "profile.json"))
}

func WriteProfile(cmd *cobra.Command, name string, profile Profile) error {
	profileDir := GetFilePath(cmd, filepath.Join("profiles", name))
	if err := os.MkdirAll(profileDir, 0700); err != nil {
		return err
	}

	return WriteJSONFile(filepath.Join(profileDir, "profile.json"), profile)
}

func DeleteProfile(cmd *cobra.Command, name string) error {
	profileDir := GetFilePath(cmd, filepath.Join("profiles", name))
	return os.RemoveAll(profileDir)
}

func RenameProfile(cmd *cobra.Command, oldName, newName string) error {
	oldProfileDir := GetFilePath(cmd, filepath.Join("profiles", oldName))
	newProfileDir := GetFilePath(cmd, filepath.Join("profiles", newName))
	return os.Rename(oldProfileDir, newProfileDir)
}

func ResetProfile(cmd *cobra.Command, name string) error {
	profile, err := LoadProfile(cmd, name)
	if err != nil {
		return err
	}
	profile.MembershipURI = DefaultMembershipURI

	return WriteProfile(cmd, name, *profile)
}

func WriteOrganizationToken(cmd *cobra.Command, profileName string, token AccessToken) error {
	profileDir := GetFilePath(cmd, filepath.Join("profiles", profileName, "organizations", token.Claims.OrganizationID))
	if err := os.MkdirAll(profileDir, 0700); err != nil {
		return err
	}

	return WriteJSONFile(filepath.Join(profileDir, "accesses.json"), token)
}

func ReadOrganizationToken(cmd *cobra.Command, profileName, organizationID string) (*AccessToken, error) {
	ret, err := ReadJSONFile[AccessToken](cmd, filepath.Join("profiles", profileName, "organizations", organizationID, "accesses.json"))
	if err != nil {
		if errors.Is(err, &fs.PathError{}) || errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	return ret, nil
}

func WriteStackToken(cmd *cobra.Command, profileName, stackID string, token AccessToken) error {
	profileDir := GetFilePath(cmd, filepath.Join("profiles", profileName, "organizations", token.Claims.OrganizationID, "stacks", stackID))
	if err := os.MkdirAll(profileDir, 0700); err != nil {
		return err
	}

	return WriteJSONFile(filepath.Join(profileDir, "accesses.json"), token)
}

func ReadStackToken(cmd *cobra.Command, profileName, organizationID, stackID string) (*AccessToken, error) {
	ret, err := ReadJSONFile[AccessToken](cmd, filepath.Join("profiles", profileName, "organizations", organizationID, "stacks", stackID, "accesses.json"))
	if err != nil {
		if errors.Is(err, &fs.PathError{}) || errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	return ret, nil
}

func WriteAppToken(cmd *cobra.Command, profileName, appAlias string, token AccessToken) error {
	profileDir := GetFilePath(cmd, filepath.Join("profiles", profileName, "organizations", token.Claims.OrganizationID, "apps", appAlias))
	if err := os.MkdirAll(profileDir, 0700); err != nil {
		return err
	}

	return WriteJSONFile(filepath.Join(profileDir, "accesses.json"), token)
}

func ReadAppToken(cmd *cobra.Command, profileName, organizationID, appAlias string) (*AccessToken, error) {
	ret, err := ReadJSONFile[AccessToken](cmd, filepath.Join("profiles", profileName, "organizations", organizationID, "apps", appAlias, "accesses.json"))
	if err != nil {
		if errors.Is(err, &fs.PathError{}) || errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	return ret, nil
}
