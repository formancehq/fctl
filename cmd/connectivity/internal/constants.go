package internal

const (
	ResourceConnectors        = "connectors"
	ResourceConnectorVersions = "connectorversions"
	ResourceConnections       = "connections"
	VersionAliasLatest        = "latest"
)

var VersionAliases = []string{VersionAliasLatest, "stable", "rc", "beta", "alpha"}

type ListOptions struct {
	PageSize int32
	Cursor   string
	Query    string
}
