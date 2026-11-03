package clientcontract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/ledger"
)

func TestBulkVariantsRoundTrip(t *testing.T) {
	fixtures := []string{
		`{"action":"CREATE_TRANSACTION","ik":"create-1","data":{"postings":[{"source":"world","destination":"users:1","amount":9007199254740993,"asset":"USD/2"}],"metadata":{"note":"kept"}}}`,
		`{"action":"ADD_METADATA","ik":"add-1","data":{"targetId":"users:1","targetType":"ACCOUNT","metadata":{"key":"value"}}}`,
		`{"action":"DELETE_METADATA","ik":"delete-1","data":{"targetId":9007199254740993,"targetType":"TRANSACTION","key":"key"}}`,
		`{"action":"REVERT_TRANSACTION","ik":"revert-1","data":{"id":9007199254740993,"force":true,"atEffectiveDate":true,"metadata":{"key":"value"}}}`,
	}
	decode := func(b []byte) any {
		d := json.NewDecoder(bytes.NewReader(b))
		d.UseNumber()
		var v any
		if err := d.Decode(&v); err != nil {
			panic(err)
		}
		return v
	}
	for _, fixture := range fixtures {
		var v ledger.V2BulkElement
		if err := json.Unmarshal([]byte(fixture), &v); err != nil {
			panic(err)
		}
		b, err := json.Marshal(v)
		if err != nil {
			panic(err)
		}
		if !reflect.DeepEqual(decode([]byte(fixture)), decode(b)) {
			panic(fmt.Sprintf("roundtrip changed: %s => %s", fixture, b))
		}
		t.Log(v.Type, "roundtrip PASS")
	}
	var invalid ledger.V2BulkElement
	if json.Unmarshal([]byte(`{"action":"UNKNOWN","data":{}}`), &invalid) == nil {
		panic("unknown action accepted")
	}
	t.Log("unknown discriminator rejected")
}
