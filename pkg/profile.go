package fctl

type Profile struct {
	MembershipURI  string `json:"membershipURI"`
	BaseServiceURI string `json:"BaseServiceURI"`
	AccessToken    string `json:"accessToken"`
}

type CurrentProfile Profile
