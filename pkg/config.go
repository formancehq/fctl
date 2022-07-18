package fctl

type Config struct {
	Profiles map[string]Profile `json:"profiles"`
}

type Profile struct {
	Mode        string      `json:"mode"`
	Production  bool        `json:"production"`
	URI         string      `json:"uri"`
	Credentials Credentials `json:"credentials"`
}

type Credentials struct {
	AccessKeyId     string `json:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret"`
}

type CurrentProfile = Profile

type CurrentProfileName = string

type GetCurrentProfile = func() (*CurrentProfile, CurrentProfileName, error)
