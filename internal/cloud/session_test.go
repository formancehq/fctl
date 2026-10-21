package cloud

import (
	"encoding/json"
	"slices"
	"testing"

	"golang.org/x/oauth2"
)

func TestApplicationSessionCloneAndJSON(t *testing.T) {
	child := &Session{Application: "deploy", Options: Options{Organization: "org"}, IDToken: "identity",
		MembershipToken: &oauth2.Token{AccessToken: "access", RefreshToken: "refresh"},
		StackToken:      &oauth2.Token{AccessToken: "stack"}, allowedScopes: []string{"apps:Read"},
		Applications: map[string]*Session{"nested": {}}, Targets: map[string]*Session{"nested": {}}, Organizations: map[string]*Session{"nested": {}}}
	root := &Session{Applications: map[string]*Session{"org/deploy": child, "empty": nil}}
	cloned := copyRoot(root)
	copyChild := cloned.Applications["org/deploy"]
	copyChild.MembershipToken.RefreshToken = "rotated"
	copyChild.StackToken.AccessToken = "changed"
	copyChild.allowedScopes[0] = "apps:Write"
	delete(cloned.Applications, "empty")
	if child.MembershipToken.RefreshToken != "refresh" || child.StackToken.AccessToken != "stack" || child.allowedScopes[0] != "apps:Read" || len(root.Applications) != 2 {
		t.Fatal("application clone mutated original credentials, scopes or map")
	}
	if copyChild.Applications != nil || copyChild.Targets != nil || copyChild.Organizations != nil {
		t.Fatal("leaf session retained nested authentication caches")
	}
	restored := roundTripSession(t, cloned)
	if restored.Applications["org/deploy"].Application != "deploy" || restored.Applications["org/deploy"].MembershipToken.RefreshToken != "rotated" || restored.Applications["org/deploy"].allowedScopes != nil {
		t.Fatal("application serialization lost alias/rotation or persisted private guard")
	}
}

func TestApplicationClaimsDecode(t *testing.T) {
	var claims identityClaims
	if err := json.Unmarshal([]byte(`{"org":[{"id":"org","applications":[{"id":"app-id","name":"Apps deploy","alias":"deploy","scopes":["apps:Read","apps:Write"]}]}]}`), &claims); err != nil {
		t.Fatal(err)
	}
	if len(claims.Organizations) != 1 || len(claims.Organizations[0].Applications) != 1 {
		t.Fatal("application claim was not decoded")
	}
	app := claims.Organizations[0].Applications[0]
	if app.ID != "app-id" || app.Name != "Apps deploy" || app.Alias != "deploy" || !slices.Equal(app.Scopes, []string{"apps:Read", "apps:Write"}) {
		t.Fatal("application claim fields or scopes were lost")
	}
}
