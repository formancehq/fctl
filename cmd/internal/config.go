package internal

import (
	"encoding/json"
	"errors"
	"os"
	"path"
)

const (
	DefaultBaseUri       = "https://sandbox.formance.cloud"
	DefaultMemberShipUri = "https://app.formance.cloud/api"
)

type persistedConfig struct {
	CurrentProfile string              `json:"currentProfile"`
	Profiles       map[string]*Profile `json:"profiles"`
}

type Config struct {
	currentProfile string
	profiles       map[string]*Profile
	manager        *ConfigManager
}

func (c *Config) MarshalJSON() ([]byte, error) {
	return json.Marshal(persistedConfig{
		CurrentProfile: c.currentProfile,
		Profiles:       c.profiles,
	})
}

func (c *Config) UnmarshalJSON(data []byte) error {
	cfg := &persistedConfig{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return err
	}
	*c = Config{
		currentProfile: cfg.CurrentProfile,
		profiles:       cfg.Profiles,
	}
	return nil
}

func (c *Config) GetProfile(name string) *Profile {
	p := c.profiles[name]
	if p != nil {
		p.config = c
	}
	return p
}

func (c *Config) GetProfileOrDefault(name string, membershipUri, baseServiceUri string) *Profile {
	p := c.GetProfile(name)
	if p == nil {
		if c.profiles == nil {
			c.profiles = map[string]*Profile{}
		}
		f := &Profile{
			membershipURI:  membershipUri,
			baseServiceURI: baseServiceUri,
			config:         c,
		}
		c.profiles[name] = f
		return f
	}
	return p
}

func (c *Config) DeleteProfile(s string) error {
	_, ok := c.profiles[s]
	if !ok {
		return errors.New("not found")
	}
	delete(c.profiles, s)
	return nil
}

func (c *Config) Persist() error {
	return c.manager.UpdateConfig(c)
}

func (c *Config) SetCurrentProfile(name string, profile *Profile) {
	c.profiles[name] = profile
	c.currentProfile = name
}

func (c *Config) SetProfile(name string, profile *Profile) {
	c.profiles[name] = profile
}

func (c *Config) GetProfiles() map[string]*Profile {
	return c.profiles
}

func (c *Config) GetCurrentProfileName() string {
	return c.currentProfile
}

func (c *Config) SetCurrentProfileName(s string) {
	c.currentProfile = s
}

type ConfigManager struct {
	configFilePath string
}

func (m *ConfigManager) Load() (*Config, error) {

	f, err := os.Open(m.configFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{
				profiles: map[string]*Profile{},
				manager:  m,
			}, nil
		}
		return nil, err
	}
	defer f.Close()

	cfg := &Config{}
	if err := json.NewDecoder(f).Decode(cfg); err != nil {
		return nil, err
	}
	cfg.manager = m
	if cfg.profiles == nil {
		cfg.profiles = map[string]*Profile{}
	}

	return cfg, nil
}

func (m *ConfigManager) UpdateConfig(config *Config) error {
	if err := os.MkdirAll(path.Dir(m.configFilePath), 0700); err != nil {
		return err
	}

	f, err := os.OpenFile(m.configFilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(config); err != nil {
		return err
	}
	return nil
}

func NewConfigManager(configFilePath string) *ConfigManager {
	return &ConfigManager{
		configFilePath: configFilePath,
	}
}
