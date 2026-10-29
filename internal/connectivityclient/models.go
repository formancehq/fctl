package connectivityclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	ResourceConnectors        = "connectors"
	ResourceConnectorVersions = "connectorversions"
	ResourceConnections       = "connections"
)

// VersionAliasLatest resolves to the newest Validated version across all
// channels; the channel aliases resolve to that channel's head with exactly
// the rules installation uses. Reserved by the API: never valid semver.
const VersionAliasLatest = "latest"

var VersionAliases = []string{VersionAliasLatest, "stable", "rc", "beta", "alpha"}

// ServiceInfo identifies the API build; it does not describe service health.
type ServiceInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"buildDate"`
}

func (s *ServiceInfo) UnmarshalJSON(data []byte) error {
	var value struct {
		Version   *string `json:"version"`
		Commit    *string `json:"commit"`
		BuildDate *string `json:"buildDate"`
	}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if value.Version == nil || value.Commit == nil || value.BuildDate == nil {
		return fmt.Errorf("service info version, commit and buildDate are required")
	}
	*s = ServiceInfo{Version: *value.Version, Commit: *value.Commit, BuildDate: *value.BuildDate}
	return nil
}

type ConnectorBranding struct {
	DisplayName *string `json:"displayName,omitempty"`
	AccentColor *string `json:"accentColor,omitempty"`
	LogoSvg     *string `json:"logoSvg,omitempty"`
	LogoSvgDark *string `json:"logoSvgDark,omitempty"`
}

type Connector struct {
	Name          string             `json:"name"`
	CreatedAt     *time.Time         `json:"createdAt,omitempty"`
	DisplayName   *string            `json:"displayName,omitempty"`
	Description   *string            `json:"description,omitempty"`
	Catalog       *string            `json:"catalog,omitempty"`
	Tags          []string           `json:"tags,omitempty"`
	Tagline       *string            `json:"tagline,omitempty"`
	Branding      *ConnectorBranding `json:"branding,omitempty"`
	LatestVersion *string            `json:"latestVersion,omitempty"`
}

type ConnectorList struct {
	Items    []Connector `json:"items"`
	PageSize int32       `json:"pageSize"`
	HasMore  bool        `json:"hasMore"`
	Next     string      `json:"next,omitempty"`
}

type ConnectorVersionSummary struct {
	Version     string     `json:"version"`
	Digest      *string    `json:"digest,omitempty"`
	ReleaseDate *time.Time `json:"releaseDate,omitempty"`
}

type ConnectorVersion struct {
	Version            string         `json:"version"`
	Digest             *string        `json:"digest,omitempty"`
	ReleaseDate        *time.Time     `json:"releaseDate,omitempty"`
	ConfigSchema       map[string]any `json:"configSchema,omitempty"`
	AdditionalMetadata map[string]any `json:"additionalMetadata,omitempty"`
}

type ConnectorVersionList struct {
	Items    []ConnectorVersionSummary `json:"items"`
	PageSize int32                     `json:"pageSize"`
	HasMore  bool                      `json:"hasMore"`
	Next     string                    `json:"next,omitempty"`
}

type FacetDistribution struct {
	Total  int64                       `json:"total"`
	Facets map[string]map[string]int64 `json:"facets"`
}

type QueryFieldCapability struct {
	Operators []string `json:"operators"`
	Enum      []string `json:"enum,omitempty"`
}

type QueryCapabilities struct {
	Resources map[string]map[string]QueryFieldCapability `json:"resources"`
}

// ConnectionConfig groups literal inputs for configuration assembly.
type ConnectionConfig struct {
	Env   map[string]string `json:"env,omitempty"`
	Files map[string]string `json:"files,omitempty"`
}

// ConnectionConfiguration contains only tenant-owned public fields.
// Connector and Ledger may be absent on imported reads. Create requires both;
// PUT preserves omitted identities; PATCH excludes identity fields.
type ConnectionConfiguration struct {
	Connector     string            `json:"connector,omitempty"`
	Version       *string           `json:"version,omitempty"`
	Channel       *string           `json:"channel,omitempty"`
	Ledger        *string           `json:"ledger,omitempty"`
	InitialCursor []byte            `json:"initialCursor,omitempty"`
	PollInterval  *string           `json:"pollInterval,omitempty"`
	Suspend       *bool             `json:"suspend,omitempty"`
	Env           map[string]string `json:"env,omitempty"`
	Files         map[string]string `json:"files,omitempty"`
}

func (s ConnectionConfiguration) ConnectorName() string { return s.Connector }
func (s ConnectionConfiguration) Configuration() *ConnectionConfig {
	return &ConnectionConfig{Env: s.Env, Files: s.Files}
}

type SuspensionSource struct {
	Source string `json:"source"`
	Name   string `json:"name"`
}

func (s *SuspensionSource) UnmarshalJSON(data []byte) error {
	var value struct {
		Source *string `json:"source"`
		Name   *string `json:"name"`
	}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if value.Source == nil || value.Name == nil {
		return fmt.Errorf("suspension source and name are required")
	}
	s.Source, s.Name = *value.Source, *value.Name
	return nil
}

type IngestionProgress struct {
	Current int64 `json:"current"`
}

// UnmarshalJSON distinguishes a missing required progress value from zero.
func (p *IngestionProgress) UnmarshalJSON(data []byte) error {
	var value struct {
		Current *int64 `json:"current"`
	}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if value.Current == nil {
		return fmt.Errorf("ingestion progress.current is required")
	}
	p.Current = *value.Current
	return nil
}

type IngestionStatus struct {
	State     *string            `json:"state,omitempty"`
	Progress  *IngestionProgress `json:"progress,omitempty"`
	LastError *string            `json:"lastError,omitempty"`
}
type StatusError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (s *StatusError) UnmarshalJSON(data []byte) error {
	var value struct {
		Code    *string `json:"code"`
		Message *string `json:"message"`
	}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if value.Code == nil || value.Message == nil {
		return fmt.Errorf("status error code and message are required")
	}
	s.Code, s.Message = *value.Code, *value.Message
	return nil
}

// ConnectionStatus reports recorded outcomes, not live worker health.
type ConnectionStatus struct {
	AppliedVersion *string            `json:"appliedVersion,omitempty"`
	Lifecycle      string             `json:"lifecycle"`
	Ingestion      *IngestionStatus   `json:"ingestion,omitempty"`
	Error          *StatusError       `json:"error,omitempty"`
	SuspendedBy    []SuspensionSource `json:"suspendedBy"`
}

type Connection struct {
	Name          string                  `json:"name"`
	Configuration ConnectionConfiguration `json:"configuration"`
	Status        *ConnectionStatus       `json:"status,omitempty"`
}

// UnmarshalJSON keeps a missing configuration distinct from an imported empty one.
func (c *Connection) UnmarshalJSON(data []byte) error {
	type wire Connection
	var value struct {
		*wire
		Configuration json.RawMessage `json:"configuration"`
	}
	var decoded Connection
	value.wire = (*wire)(&decoded)
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if len(value.Configuration) == 0 || string(value.Configuration) == "null" {
		return fmt.Errorf("connection configuration must be an object")
	}
	if err := json.Unmarshal(value.Configuration, &decoded.Configuration); err != nil {
		return err
	}
	*c = decoded
	return nil
}

type ConnectionList struct {
	Items    []Connection `json:"items"`
	PageSize int32        `json:"pageSize"`
	HasMore  bool         `json:"hasMore"`
	Next     string       `json:"next,omitempty"`
}

type ConnectionCreate struct {
	Name          string                  `json:"name"`
	Configuration ConnectionConfiguration `json:"configuration"`
}

type ConnectionUpdate struct {
	Configuration ConnectionConfiguration `json:"configuration"`
}

type ConnectionPatch map[string]any

type ListOptions struct {
	PageSize int32
	Cursor   string
	Query    string
}

type APIError struct {
	StatusCode int
	Code       string
	Message    string
	Details    map[string]any
}

type Client interface {
	GetInfo(context.Context) (*ServiceInfo, error)
	ListConnectors(context.Context, ListOptions) (*ConnectorList, error)
	GetConnectorFacets(context.Context, string) (*FacetDistribution, error)
	GetQueryCapabilities(context.Context) (*QueryCapabilities, error)
	GetConnector(context.Context, string) (*Connector, error)
	ListConnectorVersions(context.Context, string, ListOptions) (*ConnectorVersionList, error)
	GetConnectorVersion(context.Context, string, string) (*ConnectorVersion, error)
	ListConnections(context.Context, ListOptions) (*ConnectionList, error)
	CreateConnection(context.Context, ConnectionCreate) (*Connection, error)
	GetConnection(context.Context, string) (*Connection, error)
	ReplaceConnection(context.Context, string, ConnectionUpdate) (*Connection, error)
	PatchConnection(context.Context, string, ConnectionPatch) (*Connection, error)
	DeleteConnection(context.Context, string) error
}

func New(stackURI string, httpClient *http.Client) Client {
	return &client{stackURI: stackURI, httpClient: httpClient}
}
