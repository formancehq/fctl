package capabilities

import (
	"strings"
	"testing"
)

func TestParseOpenAPIManifest(t *testing.T) {
	const document = `{
	  "info": {"version": "v-test"},
	  "paths": {
	    "/api/ledger/{ledger}/transactions": {
	      "get": {"operationId": "listTransactions", "tags": ["ledger.v1"]}
	    },
	    "/api/ledger/v2/{ledger}/transactions": {
	      "get": {"operationId": "v2ListTransactions", "tags": ["ledger.v2"]}
	    },
	    "/api/payments/v3/payments": {
	      "get": {"operationId": "v3ListPayments", "tags": ["payments.v3"]}
	    },
	    "/versions": {
	      "get": {"operationId": "getVersions", "tags": []}
	    }
	  }
	}`

	manifest, err := ParseOpenAPIManifest(strings.NewReader(document))
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if manifest.SpecVersion != "v-test" {
		t.Fatalf("expected spec version v-test, got %q", manifest.SpecVersion)
	}

	ledger := manifest.Products["ledger"]
	assertAPIVersions(t, ledger.APIVersions, []APIVersion{"v1", "v2"})
	if ledger.Operations["listTransactions"]["v1"].OperationID != "listTransactions" {
		t.Fatalf("missing ledger v1 listTransactions operation")
	}
	if ledger.Operations["listTransactions"]["v2"].OperationID != "v2ListTransactions" {
		t.Fatalf("missing ledger v2 listTransactions operation")
	}

	payments := manifest.Products["payments"]
	assertAPIVersions(t, payments.APIVersions, []APIVersion{"v3"})
	if payments.Operations["listPayments"]["v3"].Path != "/api/payments/v3/payments" {
		t.Fatalf("missing payments v3 listPayments operation")
	}
	if _, ok := manifest.Products["versions"]; ok {
		t.Fatalf("/versions should not be included as a product")
	}
}

func TestCanonicalFeature(t *testing.T) {
	tests := map[string]string{
		"listTransactions":   "listTransactions",
		"v2ListTransactions": "listTransactions",
		"CreateTransactions": "createTransactions",
	}
	for input, expected := range tests {
		if got := canonicalFeature(input); got != expected {
			t.Fatalf("expected %s -> %s, got %s", input, expected, got)
		}
	}
}
