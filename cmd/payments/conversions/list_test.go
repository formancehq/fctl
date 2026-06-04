package conversions

import (
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/spf13/cobra"

	paymentsmodels "github.com/formancehq/formance-sdk-go/v4/pkg/models/payments"
)

func TestBuildQuery(t *testing.T) {
	c := NewListController()
	cmd := &cobra.Command{}
	for _, f := range []string{
		c.connectorIDFlag, c.referenceFlag, c.statusFlag,
		c.sourceAssetFlag, c.destinationAssetFlag,
	} {
		cmd.Flags().String(f, "", "")
	}

	if got := c.buildQuery(cmd); got != nil {
		t.Fatalf("no flags set: want nil query body, got %v", got)
	}

	for flag, val := range map[string]string{c.statusFlag: "COMPLETED", c.sourceAssetFlag: "USD/2"} {
		if err := cmd.Flags().Set(flag, val); err != nil {
			t.Fatalf("set --%s: %v", flag, err)
		}
	}

	// Keys must be the snake_case storage columns; the endpoint rejects anything else.
	want := map[string]any{"$and": []map[string]any{
		{"$match": map[string]any{"status": "COMPLETED"}},
		{"$match": map[string]any{"source_asset": "USD/2"}},
	}}
	if got := c.buildQuery(cmd); !reflect.DeepEqual(got, want) {
		t.Fatalf("buildQuery mismatch:\n got=%v\nwant=%v", got, want)
	}
}

func TestToConversion(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	got := toConversion(paymentsmodels.V3Conversion{
		ID:                     "cv_1",
		ConnectorID:            "conn_1",
		Provider:               "coinbaseprime",
		Reference:              "ref-1",
		CreatedAt:              now,
		UpdatedAt:              now,
		V3ConversionStatusEnum: paymentsmodels.V3ConversionStatusEnum("COMPLETED"),
		SourceAsset:            "USD/2",
		DestinationAsset:       "USDC/6",
		SourceAmount:           big.NewInt(1000),
		DestinationAmount:      big.NewInt(999),
		V3Metadata:             map[string]string{"k": "v"},
	})

	if got.ID != "cv_1" {
		t.Errorf("ID = %q", got.ID)
	}
	// Status is a typed-string enum on the SDK and must surface as a plain string.
	if got.Status != "COMPLETED" {
		t.Errorf("Status = %q, want COMPLETED", got.Status)
	}
	if got.SourceAmount.Cmp(big.NewInt(1000)) != 0 || got.DestinationAmount.Cmp(big.NewInt(999)) != 0 {
		t.Errorf("amounts mismatch: src=%v dst=%v", got.SourceAmount, got.DestinationAmount)
	}
	if !reflect.DeepEqual(got.Metadata, map[string]string{"k": "v"}) {
		t.Errorf("Metadata = %v", got.Metadata)
	}
}
