package orders

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
		c.connectorIDFlag, c.referenceFlag, c.directionFlag, c.statusFlag,
		c.typeFlag, c.sourceAssetFlag, c.destinationAssetFlag,
	} {
		cmd.Flags().String(f, "", "")
	}

	if got := c.buildQuery(cmd); got != nil {
		t.Fatalf("no flags set: want nil query body, got %v", got)
	}

	for flag, val := range map[string]string{c.statusFlag: "FILLED", c.sourceAssetFlag: "USD/2"} {
		if err := cmd.Flags().Set(flag, val); err != nil {
			t.Fatalf("set --%s: %v", flag, err)
		}
	}

	// Keys must be the snake_case storage columns; the endpoint rejects anything else.
	want := map[string]any{"$and": []map[string]any{
		{"$match": map[string]any{"status": "FILLED"}},
		{"$match": map[string]any{"source_asset": "USD/2"}},
	}}
	if got := c.buildQuery(cmd); !reflect.DeepEqual(got, want) {
		t.Fatalf("buildQuery mismatch:\n got=%v\nwant=%v", got, want)
	}
}

func TestToOrder(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	got := toOrder(paymentsmodels.V3Order{
		ID:                   "ord_1",
		ConnectorID:          "conn_1",
		Provider:             "coinbaseprime",
		Reference:            "ref-1",
		CreatedAt:            now,
		UpdatedAt:            now,
		V3OrderDirectionEnum: paymentsmodels.V3OrderDirectionEnum("BUY"),
		V3OrderStatusEnum:    paymentsmodels.V3OrderStatusEnum("FILLED"),
		V3OrderTypeEnum:      paymentsmodels.V3OrderTypeEnum("LIMIT"),
		V3TimeInForceEnum:    paymentsmodels.V3TimeInForceEnum("GOOD_UNTIL_CANCELLED"),
		SourceAsset:          "USD/2",
		DestinationAsset:     "BTC/8",
		BaseQuantityOrdered:  big.NewInt(100),
		V3Metadata:           map[string]string{"k": "v"},
		Adjustments: []paymentsmodels.V3OrderAdjustment{{
			ID:                "adj_1",
			V3OrderStatusEnum: paymentsmodels.V3OrderStatusEnum("PARTIALLY_FILLED"),
			V3Metadata:        map[string]string{"a": "b"},
		}},
	})

	// The enums are typed strings on the SDK side and must surface as plain strings.
	for _, tc := range []struct{ name, got, want string }{
		{"ID", got.ID, "ord_1"},
		{"Direction", got.Direction, "BUY"},
		{"Status", got.Status, "FILLED"},
		{"Type", got.Type, "LIMIT"},
		{"TimeInForce", got.TimeInForce, "GOOD_UNTIL_CANCELLED"},
	} {
		if tc.got != tc.want {
			t.Errorf("%s = %q, want %q", tc.name, tc.got, tc.want)
		}
	}
	if got.BaseQuantityOrdered.Cmp(big.NewInt(100)) != 0 {
		t.Errorf("BaseQuantityOrdered = %v, want 100", got.BaseQuantityOrdered)
	}
	if !reflect.DeepEqual(got.Metadata, map[string]string{"k": "v"}) {
		t.Errorf("Metadata = %v", got.Metadata)
	}
	if len(got.Adjustments) != 1 || got.Adjustments[0].Status != "PARTIALLY_FILLED" {
		t.Fatalf("adjustments not mapped: %+v", got.Adjustments)
	}
	if !reflect.DeepEqual(got.Adjustments[0].Metadata, map[string]string{"a": "b"}) {
		t.Errorf("adjustment metadata = %v", got.Adjustments[0].Metadata)
	}
}
