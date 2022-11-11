package config

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
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

func Get() (*Config, error) {
	return GetConfigManager().Load()
}

func GetConfigManager() *ConfigManager {
	return NewConfigManager(viper.GetString(FileFlag))
}

func GetCurrentProfileName(config *Config) string {
	if profile := viper.GetString(ProfileFlag); profile != "" {
		return profile
	}
	currentProfileName := config.GetCurrentProfileName()
	if currentProfileName == "" {
		currentProfileName = "default"
	}
	return currentProfileName
}

func GetCurrentProfile(cfg *Config) *Profile {
	return cfg.GetProfileOrDefault(GetCurrentProfileName(cfg), viper.GetString(MembershipUriFlag),
		viper.GetString(BaseServiceUriFlag))
}
