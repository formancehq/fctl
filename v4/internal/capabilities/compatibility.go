package capabilities

var DefaultComponentCompatibility = ComponentCompatibility{
	{Product: "ledger", Range: ">=1.0.0 <2.0.0", APIVersions: []APIVersion{"v1"}},
	{Product: "ledger", Range: ">=2.0.0 <3.0.0", APIVersions: []APIVersion{"v1", "v2"}},
	{Product: "ledger", Range: ">=3.0.0", APIVersions: []APIVersion{"v1", "v2", "v3"}},
	{Product: "payments", Range: ">=1.0.0 <3.0.0", APIVersions: []APIVersion{"v1"}},
	{Product: "payments", Range: ">=3.0.0", APIVersions: []APIVersion{"v1", "v3"}},
	{Product: "orchestration", Range: ">=1.0.0 <2.0.0", APIVersions: []APIVersion{"v1"}},
	{Product: "orchestration", Range: ">=2.0.0", APIVersions: []APIVersion{"v1", "v2"}},
	{Product: "auth", Range: ">=0.0.0", APIVersions: []APIVersion{"v1"}},
	{Product: "wallets", Range: ">=0.0.0", APIVersions: []APIVersion{"v1"}},
	{Product: "webhooks", Range: ">=0.0.0", APIVersions: []APIVersion{"v1"}},
	{Product: "reconciliation", Range: ">=0.0.0", APIVersions: []APIVersion{"v1"}},
}
