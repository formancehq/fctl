package internal

const (
	ResourceConnectors        = "connectors"
	ResourceConnectorVersions = "connectorversions"
	ResourceConnections       = "connections"
)

type ListOptions struct {
	PageSize int32
	Cursor   string
	Query    string
}
