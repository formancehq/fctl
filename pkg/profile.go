package fctl

import (
	"github.com/zitadel/oidc/pkg/oidc"
)

type Profile struct {
	MembershipURI  string       `json:"membershipURI"`
	BaseServiceURI string       `json:"baseServiceURI"`
	Tokens         *oidc.Tokens `json:"tokens"`
}

type CurrentProfile Profile
