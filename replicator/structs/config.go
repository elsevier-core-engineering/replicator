package structs

import (
	"net"

	"github.com/d3sw/replicator/notifier"
)

// Config is the main configuration struct used to configure the replicator
// application.
type Config struct {
	// BindAddress is the TCP IP address which Replicator will bind to for both
	// HTTP and RPC.
	BindAddress string `mapstructure:"bind_address"`

	// ClusterScalingDisable is a global parameter that can be used to disable
	// Replicator from undertaking any cluster scaling evaluations.
	ClusterScalingDisable bool `mapstructure:"cluster_scaling_disable"`

	// ClusterScalingInterval is the period in seconds at which the ticker will
	// run.
	ClusterScalingInterval int `mapstructure:"cluster_scaling_interval"`

	// Consul is the location of the Consul instance or cluster endpoint to query
	// (may be an IP address or FQDN) with port.
	Consul string `mapstructure:"consul"`

	// ConsulClient provides a client to interact with the Consul API.
	ConsulClient ConsulClient

	// ConsulKeyRoot is the Consul key root location where Replicator stores
	// and fetches critical information from.
	ConsulKeyRoot string `mapstructure:"consul_key_root"`

	// ConsulToken is the Consul ACL token used to access KeyValues from a
	// secure Consul installation.
	ConsulToken string `mapstructure:"consul_token"`

	// HTTPPort is the port to which the HPPT API will bind and listen.
	HTTPPort string `mapstructure:"http_port"`

	// JobScalingDisable is a global parameter that can be used to disable
	// Replicator from undertaking any job scaling evaluations.
	JobScalingDisable bool `mapstructure:"job_scaling_disable"`

	// JobScalingInterval is the period in seconds at which the ticker will
	// run.
	JobScalingInterval int `mapstructure:"job_scaling_interval"`

	// LogLevel is the level at which the application should log from.
	LogLevel string `mapstructure:"log_level"`

	// Nomad is the location of the Nomad instance or cluster endpoint to query
	// (may be an IP address or FQDN) with port.
	Nomad string `mapstructure:"nomad"`

	// Nomad ACL token.
	NomadToken string `mapstructure:"nomad_token"`

	// NomadTLSServerName, if set, is used to set the SNI host when connecting via
	// TLS.
	NomadTLSServerName string `mapstructure:"nomad_tls_server_name"`

	// NomadClient provides a client to interact with the Nomad API.
	NomadClient NomadClient

	// Notification contains Replicators notification configuration params and
	// initialized backends.
	Notification *Notification `mapstructure:"notification"`

	RPCAddr      *net.TCPAddr
	RPCAdvertise *net.TCPAddr

	// RPCPort is the RPC port on which Replicator will run and advertise.
	RPCPort int `mapstructure:"rpc_port"`

	// ScalingConcurrency sets the maximum number of concurrent scaling
	// operations allowed for both job and worker pool scaling.
	ScalingConcurrency int `mapstructure:"scaling_concurrency"`

	// Telemetry is the configuration struct that controls the telemetry settings.
	Telemetry *Telemetry `mapstructure:"telemetry"`
}

// Telemetry is the struct that control the telemetry configuration. If a value
// is present then telemetry is enabled. Currently statsd is only supported for
// sending telemetry.
type Telemetry struct {
	// StatsdAddress specifies the address of a statsd server to forward metrics
	// to and should include the port.
	StatsdAddress string `mapstructure:"statsd_address"`
}

// Notification is the control struct for Replicator notifications.
type Notification struct {
	// ClusterIdentifier is a friendly name which is used when sending
	// notifications for easy human identification.
	ClusterIdentifier string `mapstructure:"cluster_identifier"`

	// PagerDutyServiceKey is the PD integration key for the Events API v1.
	PagerDutyServiceKey string `mapstructure:"pagerduty_service_key"`

	// OpsGenieAPIKey is the OpsGenie integration key for the Alerts API v2.
	OpsGenieAPIKey string `mapstructure:"opsgenie_service_key"`

	// Notifiers is where our initialize notification backends are stored so they
	// can be used on the fly when required.
	Notifiers []notifier.Notifier
}

// Merge merges two configurations.
func (c *Config) Merge(b *Config) *Config {
	config := *c

	if b.BindAddress != "" {
		config.BindAddress = b.BindAddress
	}

	if b.Nomad != "" {
		config.Nomad = b.Nomad
	}

	if b.NomadToken != "" {
		config.NomadToken = b.NomadToken
	}
	if b.NomadTLSServerName != "" {
		config.NomadTLSServerName = b.NomadTLSServerName
	}

	if b.Consul != "" {
		config.Consul = b.Consul
	}

	if b.ConsulToken != "" {
		config.ConsulToken = b.ConsulToken
	}

	if b.ConsulKeyRoot != "" {
		config.ConsulKeyRoot = b.ConsulKeyRoot
	}

	if b.LogLevel != "" {
		config.LogLevel = b.LogLevel
	}

	if b.ClusterScalingInterval > 0 {
		config.ClusterScalingInterval = b.ClusterScalingInterval
	}

	if b.JobScalingInterval > 0 {
		config.JobScalingInterval = b.JobScalingInterval
	}

	if b.ClusterScalingDisable {
		config.ClusterScalingDisable = b.ClusterScalingDisable
	}

	if b.JobScalingDisable {
		config.JobScalingDisable = b.JobScalingDisable
	}

	if b.HTTPPort != "" {
		config.HTTPPort = b.HTTPPort
	}

	if b.RPCPort > 0 {
		config.RPCPort = b.RPCPort
	}

	if b.ScalingConcurrency > 0 {
		config.ScalingConcurrency = b.ScalingConcurrency
	}

	// Apply the Telemetry config
	if config.Telemetry == nil && b.Telemetry != nil {
		telemetry := *b.Telemetry
		config.Telemetry = &telemetry
	} else if b.Telemetry != nil {
		config.Telemetry = config.Telemetry.Merge(b.Telemetry)
	}

	// Apply the Notification config
	if config.Notification == nil && b.Notification != nil {
		notification := *b.Notification
		config.Notification = &notification
	} else if b.Notification != nil {
		config.Notification = config.Notification.Merge(b.Notification)
	}

	return &config
}

// Merge is used to merge two Telemetry configurations together.
func (t *Telemetry) Merge(b *Telemetry) *Telemetry {
	config := *t

	if b.StatsdAddress != "" {
		config.StatsdAddress = b.StatsdAddress
	}

	return &config
}

// Merge is used to merge two Notification configurations together.
func (n *Notification) Merge(b *Notification) *Notification {
	config := *n

	if b.ClusterIdentifier != "" {
		config.ClusterIdentifier = b.ClusterIdentifier
	}

	if b.PagerDutyServiceKey != "" {
		config.PagerDutyServiceKey = b.PagerDutyServiceKey
	}

	if b.OpsGenieAPIKey != "" {
		config.OpsGenieAPIKey = b.OpsGenieAPIKey
	}

	return &config
}
