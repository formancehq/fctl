package legacy

import "testing"

func TestCompatibility(t *testing.T) {
	for _, test := range []struct {
		service, version, line string
		want                   bool
	}{
		{"ledger", "v2.4.15", "v3.2", true},
		{"ledger", "v2.4.0-beta.1", "v3.2-rc", true},
		{"ledger", "3.0.0-alpha.1", "", false},
		{"ledger", "2.4.15", "v4.0-beta", false},
		{"auth", "2.5.1", "v3.2", true},
		{"auth", "2.5.1", "v4.0-beta", false},
		{"payments", "3.4.8", "v3.2", true},
		{"wallets", "2.2.1", "", true},
		{"unknown", "2.0.0", "v3.2", false},
		{"ledger", "commit-sha", "v3.2", false},
		{"ledger", "02.4.15", "v3.2", false},
		{"ledger", "2.bad", "v3.2", false},
	} {
		t.Run(test.service+"/"+test.version+"/"+test.line, func(t *testing.T) {
			if got := Supported(test.service, test.version, test.line); got != test.want {
				t.Fatalf("got %v want %v", got, test.want)
			}
		})
	}
}
