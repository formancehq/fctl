package connectivityclient

import (
	"context"
	"net/http"
	"time"
)

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

type InstanceConfig struct {
	Env   map[string]EnvValue `json:"env,omitempty"`
	Files []FileMount         `json:"files,omitempty"`
}

type VersionEntry struct {
	Version string  `json:"version"`
	Digest  *string `json:"digest,omitempty"`
	Image   *string `json:"image,omitempty"`
}

type PluginSpec struct {
	Image          string          `json:"image"`
	Version        *string         `json:"version,omitempty"`
	Description    *string         `json:"description,omitempty"`
	DocsURL        *string         `json:"docsURL,omitempty"`
	Capabilities   []string        `json:"capabilities,omitempty"`
	ConfigSchema   map[string]any  `json:"configSchema,omitempty"`
	DefaultVersion *string         `json:"defaultVersion,omitempty"`
	Versions       []VersionEntry  `json:"versions,omitempty"`
	Defaults       *InstanceConfig `json:"defaults,omitempty"`
}

type PluginStatus struct {
	Phase   *string `json:"phase,omitempty"`
	Message *string `json:"message,omitempty"`
}

type Plugin struct {
	Metadata ObjectMeta    `json:"metadata"`
	Spec     PluginSpec    `json:"spec"`
	Status   *PluginStatus `json:"status,omitempty"`
}

type PluginList struct {
	Items    []Plugin `json:"items"`
	Continue string   `json:"continue,omitempty"`
}

type InstanceSpec struct {
	Plugin          string          `json:"plugin"`
	Version         *string         `json:"version,omitempty"`
	ConnectivityRef *string         `json:"connectivityRef,omitempty"`
	Ledger          string          `json:"ledger"`
	StartSequence   *int64          `json:"startSequence,omitempty"`
	PollInterval    *string         `json:"pollInterval,omitempty"`
	Config          *InstanceConfig `json:"config,omitempty"`
}

type InstanceStatus struct {
	Phase             *string `json:"phase,omitempty"`
	State             *string `json:"state,omitempty"`
	PluginAddress     *string `json:"pluginAddress,omitempty"`
	ResolvedImage     *string `json:"resolvedImage,omitempty"`
	CurrentSequence   *int64  `json:"currentSequence,omitempty"`
	SourceTipSequence *int64  `json:"sourceTipSequence,omitempty"`
	LastError         *string `json:"lastError,omitempty"`
	Message           *string `json:"message,omitempty"`
}

type Instance struct {
	Metadata ObjectMeta      `json:"metadata"`
	Spec     InstanceSpec    `json:"spec"`
	Status   *InstanceStatus `json:"status,omitempty"`
}

type InstanceList struct {
	Items    []Instance `json:"items"`
	Continue string     `json:"continue,omitempty"`
}

type InstanceCreate struct {
	Name        string            `json:"name"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Spec        InstanceSpec      `json:"spec"`
}

type InstancePatch map[string]any

type ListOptions struct {
	Limit    int32
	Continue string
	Plugin   string
}

type APIError struct {
	StatusCode int
	Code       string
	Message    string
	Details    map[string]any
}

type Client interface {
	ListPlugins(context.Context, ListOptions) (*PluginList, error)
	GetPlugin(context.Context, string) (*Plugin, error)
	ListInstances(context.Context, ListOptions) (*InstanceList, error)
	CreateInstance(context.Context, InstanceCreate) (*Instance, error)
	GetInstance(context.Context, string) (*Instance, error)
	PatchInstance(context.Context, string, InstancePatch) (*Instance, error)
	DeleteInstance(context.Context, string) error
}

func New(stackURI string, httpClient *http.Client) Client {
	return &client{stackURI: stackURI, httpClient: httpClient}
}
