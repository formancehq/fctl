package internal

import (
	"encoding/json"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestParseSubjectPreservesWireDiscriminators(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  string
	}{
		{"account=world", `{"type":"ACCOUNT","identifier":"world"}`},
		{"wallet=id:wallet-1", `{"type":"WALLET","identifier":"wallet-1","balance":"main"}`},
		{"wallet=id:wallet-1/reserve", `{"type":"WALLET","identifier":"wallet-1","balance":"reserve"}`},
	} {
		t.Run(tc.input, func(t *testing.T) {
			subject, err := ParseSubject(tc.input, &cobra.Command{}, nil)
			require.NoError(t, err)
			data, err := json.Marshal(subject)
			require.NoError(t, err)
			require.JSONEq(t, tc.want, string(data))
		})
	}
}
