package configs

type MetaSourceConfig struct {
	HttpServer *HTTPServerConfig `json:"http_server" mapstructure:"http_server"`

	FetchSource       *FetchSourceConfig       `json:"fetch_source" mapstructure:"fetch_source"`
	AcceptEventSource *AcceptEventSourceConfig `json:"accept_event_source" mapstructure:"accept_event_source"`
	KubeSource        *KubeSourceConfig        `json:"kube_source" mapstructure:"kube_source"`

	Exporter *ExporterConfig `json:"exporter" mapstructure:"exporter"`
	Querier  *QuerierConfig  `json:"querier" mapstructure:"querier"`
}

type FetchSourceConfig struct {
	// SourceConfig
	SourceAddr   string `json:"source_addr" mapstructure:"source_addr"`
	FetchedTypes string `json:"fetched_types" mapstructure:"fetched_types"`
}

type AcceptEventSourceConfig struct {
	EnableAcceptServer bool `json:"enable_accept_server" mapstructure:"enable_accept_server"`

	// Deprecated use EnableAcceptServer instead
	AcceptEventPort int `json:"accept_event_port" mapstructure:"accept_event_port"`
}

type KubeSourceConfig struct {
	KubeAuthType   string `json:"kube_auth_type" mapstructure:"kube_auth_type"`
	KubeAuthConfig string `json:"kube_auth_config" mapstructure:"kube_auth_config"`
	ClusterID      string `json:"cluster_id" mapstructure:"cluster_id"`

	IsEndpointsNeeded bool `json:"is_endpoints_needed" mapstructure:"is_endpoints_needed"`
}

type ExporterConfig struct {
	// ExportConfig
	RemoteWriteAddr   string `json:"remote_write_addr" mapstructure:"remote_write_addr"`
	EnableFetchServer bool   `json:"enable_fetch_server" mapstructure:"enable_fetch_server"`

	// Deprecated use EnableFetchServer instead
	FetchServerPort int `json:"fetch_server_port" mapstructure:"fetch_server_port"`
}

type QuerierConfig struct {
	EnableQueryServer bool `json:"enable_query_server" mapstructure:"enable_query_server"`
	IsSingleCluster   bool `json:"is_single_cluster" mapstructure:"is_single_cluster"`

	// Deprecated
	QueryServerPort int `json:"query_server_port" mapstructure:"query_server_port"`
}

type HTTPServerConfig struct {
	Port int `json:"port" mapstructure:"port"`
}
