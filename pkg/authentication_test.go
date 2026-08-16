package fctl

import (
	"reflect"
	"testing"

	"github.com/formancehq/go-libs/v4/oidc"
)

func TestAccessTokenMissingScopes(t *testing.T) {
	token := AccessToken{
		TokenWithClaims: TokenWithClaims[AccessTokenClaims]{
			Claims: AccessTokenClaims{
				Scopes: oidc.SpaceDelimitedArray{
					"organization:CreateStack",
					"organization:ReadRegion",
				},
			},
		},
	}

	missingScopes := token.MissingScopes(
		"organization:CreateStack",
		"organization:ListRegions",
		"organization:ReadRegion",
	)

	expected := []string{"organization:ListRegions"}
	if !reflect.DeepEqual(missingScopes, expected) {
		t.Fatalf("expected missing scopes %v, got %v", expected, missingScopes)
	}

	if token.HasScopes("organization:CreateStack", "organization:ListRegions") {
		t.Fatal("expected HasScopes to return false for missing scope")
	}

	if !token.HasScopes("organization:CreateStack", "organization:ReadRegion") {
		t.Fatal("expected HasScopes to return true when all scopes are present")
	}
}
