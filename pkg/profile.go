package fctl

import (
	"golang.org/x/oauth2"
)

type Profile struct {
	MembershipURI  string        `json:"membershipURI"`
	BaseServiceURI string        `json:"baseServiceURI"`
	Token          *oauth2.Token `json:"token"`
}

type CurrentProfile Profile
