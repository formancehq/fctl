package connectivityclient

import (
	"context"
	"net/http"
	"time"
)

const (
	ResourceConnectors         = "connectors"
	ResourceConnectorVersions  = "connectorversions"
	ResourceConnectorInstances = "connectorinstances"
)

// VersionAliasLatest resolves to the newest Validated version across all
// channels; the channel aliases resolve to that channel's head with exactly
// the rules installation uses. Reserved by the API: never valid semver.
const VersionAliasLatest = "latest"

var VersionAliases = []string{VersionAliasLatest, "stable", "rc", "beta", "alpha"}

type ObjectMeta struct {
	Name              *string           `json:"name,omitempty"`
	Namespace         *string           `json:"namespace,omitempty"`
	ResourceVersion   *string           `json:"resourceVersion,omitempty"`
	UID               *string           `json:"uid,omitempty"`
	CreationTimestamp *time.Time        `json:"creationTimestamp,omitempty"`
	Labels            map[string]string `json:"labels,omitempty"`
	Annotations       map[string]string `json:"annotations,omitempty"`
}

type KeyRef struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

type EnvValue struct {
	Value        *string `json:"value,omitempty"`
	SecretRef    *KeyRef `json:"secretRef,omitempty"`
	ConfigMapRef *KeyRef `json:"configMapRef,omitempty"`
}

type FileMount struct {
	Path         string  `json:"path"`
	Value        *string `json:"value,omitempty"`
	SecretRef    *KeyRef `json:"secretRef,omitempty"`
	ConfigMapRef *KeyRef `json:"configMapRef,omitempty"`
	Mode         *int32  `json:"mode,omitempty"`
}

type ConnectorBranding struct {
	DisplayName *string `json:"displayName,omitempty"`
	AccentColor *string `json:"accentColor,omitempty"`
	LogoSvg     *string `json:"logoSvg,omitempty"`
	LogoSvgDark *string `json:"logoSvgDark,omitempty"`
}

type ConnectorSpec struct {
	DisplayName   *string            `json:"displayName,omitempty"`
	Description   *string            `json:"description,omitempty"`
	ImageURL      *string            `json:"imageUrl,omitempty"`
	Catalog       *string            `json:"catalog,omitempty"`
	Tags          []string           `json:"tags,omitempty"`
	Tagline       *string            `json:"tagline,omitempty"`
	Branding      *ConnectorBranding `json:"branding,omitempty"`
	LatestVersion *string            `json:"latestVersion,omitempty"`
}

type ConnectorStatus struct {
	Phase   *string `json:"phase,omitempty"`
	Message *string `json:"message,omitempty"`
}

type Connector struct {
	Metadata ObjectMeta       `json:"metadata"`
	Spec     ConnectorSpec    `json:"spec"`
	Status   *ConnectorStatus `json:"status,omitempty"`
}

type ConnectorList struct {
	Items    []Connector `json:"items"`
	PageSize int32       `json:"pageSize"`
	HasMore  bool        `json:"hasMore"`
	Next     string      `json:"next,omitempty"`
}

type ConnectorVersionSummary struct {
	Version     string     `json:"version"`
	Image       string     `json:"image"`
	Digest      *string    `json:"digest,omitempty"`
	ReleaseDate *time.Time `json:"releaseDate,omitempty"`
}

type ConnectorVersion struct {
	Version            string         `json:"version"`
	Image              string         `json:"image"`
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

type ConnectorInstanceConfig struct {
	Env   map[string]EnvValue `json:"env,omitempty"`
	Files []FileMount         `json:"files,omitempty"`
}

type ConnectorInstanceSpec struct {
	Connector       string                   `json:"connector"`
	Version         *string                  `json:"version,omitempty"`
	Channel         *string                  `json:"channel,omitempty"`
	ConnectivityRef *string                  `json:"connectivityRef,omitempty"`
	Ledger          string                   `json:"ledger"`
	StartSequence   *int64                   `json:"startSequence,omitempty"`
	PollInterval    *string                  `json:"pollInterval,omitempty"`
	Config          *ConnectorInstanceConfig `json:"config,omitempty"`
}

type ConnectorInstanceStatus struct {
	Phase                *string `json:"phase,omitempty"`
	State                *string `json:"state,omitempty"`
	ConnectorAddress     *string `json:"connectorAddress,omitempty"`
	ResolvedImage        *string `json:"resolvedImage,omitempty"`
	ResolvedConnectorRef *string `json:"resolvedConnectorRef,omitempty"`
	ResolvedVersion      *string `json:"resolvedVersion,omitempty"`
	ResolvedDigest       *string `json:"resolvedDigest,omitempty"`
	CurrentSequence      *int64  `json:"currentSequence,omitempty"`
	SourceTipSequence    *int64  `json:"sourceTipSequence,omitempty"`
	LastError            *string `json:"lastError,omitempty"`
	Message              *string `json:"message,omitempty"`
}

type ConnectorInstance struct {
	Metadata ObjectMeta               `json:"metadata"`
	Spec     ConnectorInstanceSpec    `json:"spec"`
	Status   *ConnectorInstanceStatus `json:"status,omitempty"`
}

type ConnectorInstanceList struct {
	Items    []ConnectorInstance `json:"items"`
	PageSize int32               `json:"pageSize"`
	HasMore  bool                `json:"hasMore"`
	Next     string              `json:"next,omitempty"`
}

type ConnectorInstanceCreate struct {
	Name        string                `json:"name"`
	Labels      map[string]string     `json:"labels,omitempty"`
	Annotations map[string]string     `json:"annotations,omitempty"`
	Spec        ConnectorInstanceSpec `json:"spec"`
}

type ConnectorInstancePatch map[string]any

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
	ListConnectors(context.Context, ListOptions) (*ConnectorList, error)
	GetConnectorFacets(context.Context, string) (*FacetDistribution, error)
	GetQueryCapabilities(context.Context) (*QueryCapabilities, error)
	GetConnector(context.Context, string) (*Connector, error)
	ListConnectorVersions(context.Context, string, ListOptions) (*ConnectorVersionList, error)
	GetConnectorVersion(context.Context, string, string) (*ConnectorVersion, error)
	ListConnectorInstances(context.Context, ListOptions) (*ConnectorInstanceList, error)
	CreateConnectorInstance(context.Context, ConnectorInstanceCreate) (*ConnectorInstance, error)
	GetConnectorInstance(context.Context, string) (*ConnectorInstance, error)
	PatchConnectorInstance(context.Context, string, ConnectorInstancePatch) (*ConnectorInstance, error)
	DeleteConnectorInstance(context.Context, string) error
}

func New(stackURI string, httpClient *http.Client) Client {
	return &client{stackURI: stackURI, httpClient: httpClient}
}
