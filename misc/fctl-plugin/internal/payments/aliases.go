package payments

var aliases = map[string][]string{"accounts": {"acc", "a", "ac", "account"}, "bank-accounts": {"bank_accounts", "bacc", "ba", "bac", "baccount"}, "payments": {"p"}, "pools": {"pool"},
	"transfer-initiation": {"transfer_initiation", "payment_initiations", "payment-initiations", "ti"}, "transfer-initiation update-status": {"update_status", "u"}, "connectors": {"c", "co", "con"},
	"connectors get-config": {"getconfig", "getconf", "gc", "get", "g"}, "connectors update-config": {"uc"}, "connectors list-available": {"la"}, "connectors schedules": {"sch"},
	"connectors schedules instances": {"inst"}, "connectors install": {"i"}, "connectors uninstall": {"u", "un"},
	"bank-accounts forward": {"fo", "f"}, "bank-accounts update-metadata": {"um", "update-meta"}, "payments set-metadata": {"sm", "set-meta"},
	"pools add-account": {"add", "a"}, "pools remove-account": {"remove", "rm"}, "pools update-query": {"uq"},
	"transfer-initiation approve": {"a"}, "transfer-initiation reject": {"rj"}, "transfer-initiation retry": {"r"}, "transfer-initiation reverse": {"re"}, "orders": {"o"}, "conversions": {"cv"}, "tasks": {"t"}}
