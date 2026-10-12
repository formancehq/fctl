package pluginsdk

import "testing"

func TestServiceVersion(t *testing.T) {
	for _, test := range []struct {
		body, want string
		bad        bool
	}{
		{`{"version":"v3.0.0-beta.10"}`, "3.0.0-beta.10", false},
		{`{"data":{"version":"v2.4.15"}}`, "2.4.15", false},
		{`{"data":{"version":"2.4.15"},"version":"v2.4.15"}`, "2.4.15", false},
		{`{"data":{"version":"2.4.15"},"version":"3.0.0"}`, "", true},
		{`{"data":{}}`, "", true},
		{`{"version":" 3.0.0"}`, "", true},
		{`{"version":"3.0.0"} {}`, "", true},
	} {
		t.Run(test.body, func(t *testing.T) {
			got, err := ServiceVersion([]byte(test.body))
			if (err != nil) != test.bad || got != test.want {
				t.Fatalf("got %q, %v; want %q bad=%v", got, err, test.want, test.bad)
			}
		})
	}
}
