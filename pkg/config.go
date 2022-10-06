package fctl

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

type Config struct {
	CurrentProfile string              `json:"currentProfile"`
	Profiles       map[string]*Profile `json:"profiles"`
}

func (c *Config) GetProfile(name string) *Profile {
	return c.Profiles[name]
}

func (c *Config) GetProfileOrDefault(name string, f *Profile) *Profile {
	p := c.GetProfile(name)
	if p == nil {
		if c.Profiles == nil {
			c.Profiles = map[string]*Profile{}
		}
		c.Profiles[name] = f
		return f
	}
	return p
}

func (c *Config) DeleteProfile(s string) error {
	_, ok := c.Profiles[s]
	if !ok {
		return errors.New("not found")
	}
	delete(c.Profiles, s)
	return nil
}

type ConfigManager struct {
	configFilePath string
}

func (m *ConfigManager) Load() (*Config, error) {

	f, err := os.Open(m.configFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}
	defer f.Close()

	cfg := &Config{}
	if err := json.NewDecoder(f).Decode(cfg); err != nil {
		return nil, err
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
