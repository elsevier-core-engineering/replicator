package agent

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	metrics "github.com/armon/go-metrics"
	"github.com/d3sw/replicator/command"
	"github.com/d3sw/replicator/command/base"
	"github.com/d3sw/replicator/logging"
	"github.com/d3sw/replicator/notifier"
	"github.com/d3sw/replicator/replicator"
	"github.com/d3sw/replicator/replicator/structs"
	"github.com/d3sw/replicator/version"
)

// Command is the agent command structure used to track passed args as well as
// the CLI meta.
type Command struct {
	args []string
	command.Meta
	httpServer   *HTTPServer
	server       *replicator.Server
	shutdownChan chan struct{}
}

// Run triggers a run of the replicator agent by setting up and parsing the
// configuration and then initiating a new runner.
func (c *Command) Run(args []string) int {

	c.args = args
	conf := c.parseFlags()
	if conf == nil {
		return 1
	}

	err := c.initialzeAgent(conf)
	if err != nil {
		logging.Error("command/agent: unable to initialize agent: %v", err)
		return 1
	}

	cVerb := "enabled"
	jVerb := "enabled"

	if conf.ClusterScalingDisable {
		cVerb = "disabled"
	}
	if conf.JobScalingDisable {
		jVerb = "disabled"
	}

	logging.Info("command/agent: running version %v", version.Get())
	logging.Info("command/agent: starting replicator agent...")
	logging.Info("command/agent: replicator is running with cluster scaling globally %s", cVerb)
	logging.Info("command/agent: replicator is running with job scaling globally %s", jVerb)

	// Create the initial runner with the merged configuration parameters.
	server, err := replicator.NewServer(conf)
	if err != nil {
		logging.Error("command/agent: failed to start Replicator server: %s", err)
		return 1
	}

	c.server = server

	http, err := NewHTTPServer(c, conf)
	if err != nil {
		logging.Error("command/agent: unable to start HTTP server: %s", err)
		return 1
	}

	c.httpServer = http

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	for {
		select {
		case s := <-signalCh:
			switch s {
			case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				logging.Info("command/agent: caught signal %v", s)
				server.Shutdown()
				c.httpServer.Shutdown()
				return 1

			case syscall.SIGHUP:
				logging.Info("command/agent: caught signal %v", s)
				server.Shutdown()
				c.httpServer.Shutdown()

				// Reload the configuration in order to make proper use of SIGHUP.
				config := c.parseFlags()
				if err != nil {
					return 1
				}

				err := c.initialzeAgent(config)
				if err != nil {
					logging.Error("command/agent: unable to initialize agent: %v", err)
					return 1
				}

				server, err := replicator.NewServer(conf)
				if err != nil {
					return 1
				}

				c.server = server

				http, err := NewHTTPServer(c, conf)
				if err != nil {
					logging.Error("command/agent: unable to start HTTP server: %s", err)
					return 1
				}

				c.httpServer = http
			}
		}
	}
}

func (c *Command) parseFlags() *structs.Config {

	var configPath string
	var dev bool

	// An empty new config is setup here to allow us to fill this with any passed
	// cli flags for later merging.
	cliConfig := &structs.Config{
		Telemetry:    &structs.Telemetry{},
		Notification: &structs.Notification{},
	}

	flags := c.Meta.FlagSet("agent", command.FlagSetClient)
	flags.Usage = func() { c.UI.Error(c.Help()) }

	flags.StringVar(&configPath, "config", "", "")
	flags.BoolVar(&dev, "dev", false, "")

	// Top level configuration flags
	flags.StringVar(&cliConfig.BindAddress, "bind-address", "", "")
	flags.StringVar(&cliConfig.Consul, "consul", "", "")
	flags.StringVar(&cliConfig.ConsulKeyRoot, "consul-key-root", "", "")
	flags.StringVar(&cliConfig.ConsulToken, "consul-token", "", "")
	flags.StringVar(&cliConfig.LogLevel, "log-level", "", "")
	flags.StringVar(&cliConfig.Nomad, "nomad", "", "")
	flags.StringVar(&cliConfig.NomadToken, "nomad-token", "", "")
	flags.StringVar(&cliConfig.NomadTLSServerName, "nomad-tls-server-name", "", "")
	flags.IntVar(&cliConfig.ClusterScalingInterval, "cluster-scaling-interval", 0, "")
	flags.IntVar(&cliConfig.JobScalingInterval, "job-scaling-interval", 0, "")
	flags.IntVar(&cliConfig.ScalingConcurrency, "scaling-concurrency", 0, "")
	flags.BoolVar(&cliConfig.ClusterScalingDisable, "cluster-scaling-disable", false, "")
	flags.BoolVar(&cliConfig.JobScalingDisable, "job-scaling-disable", false, "")
	flags.StringVar(&cliConfig.HTTPPort, "http-port", "", "")
	flags.IntVar(&cliConfig.RPCPort, "rpc-port", 0, "")

	// Telemetry configuration flags
	flags.StringVar(&cliConfig.Telemetry.StatsdAddress, "statsd-address", "", "")

	// Notification configuration flags
	flags.StringVar(&cliConfig.Notification.ClusterIdentifier, "cluster-identifier", "", "")
	flags.StringVar(&cliConfig.Notification.PagerDutyServiceKey, "pagerduty-service-key", "", "")
	flags.StringVar(&cliConfig.Notification.OpsGenieAPIKey, "opsgenie-api-key", "", "")

	if err := flags.Parse(c.args); err != nil {
		return nil
	}

	// Depending on the flags provided (if any) we load a default configuration
	// which will be the basis for all merging.
	var config *structs.Config

	if dev {
		config = base.DevConfig()
	} else {
		config = base.DefaultConfig()
	}

	if configPath != "" {
		current, err := base.LoadConfig(configPath)
		if err != nil {
			c.UI.Error(fmt.Sprintf("Error loading configuration from %s: %s", configPath, err))
			return nil
		}

		config = config.Merge(current)
	}

	config = config.Merge(cliConfig)
	return config

}

// setupTelemetry is used to setup Replicators telemetry.
func (c *Command) setupTelemetry(config *structs.Telemetry) error {

	// Setup telemetry to aggregate on 10 second intervals for 1 minute.
	inm := metrics.NewInmemSink(10*time.Second, time.Minute)
	metrics.DefaultInmemSignal(inm)

	var telemetry *structs.Telemetry
	if config == nil {
		telemetry = &structs.Telemetry{}
	} else {
		telemetry = config
	}

	metricsConf := metrics.DefaultConfig("replicator")

	var fanout metrics.FanoutSink

	// Configure the statsd sink
	if telemetry.StatsdAddress != "" {
		sink, err := metrics.NewStatsdSink(telemetry.StatsdAddress)
		if err != nil {
			return err
		}
		fanout = append(fanout, sink)
	}

	// Initialize the global sink
	if len(fanout) > 0 {
		fanout = append(fanout, inm)
		metrics.NewGlobal(metricsConf, fanout)
	} else {
		metricsConf.EnableHostname = false
		metrics.NewGlobal(metricsConf, inm)
	}
	return nil
}

// setupNotifier is used to setup Replicators notifier provider.
func (c *Command) setupNotifier(config *structs.Notification) (err error) {

	// Configure the PagerDuty notifier.
	if config.PagerDutyServiceKey != "" {

		p := make(map[string]string)
		p["PagerDutyServiceKey"] = config.PagerDutyServiceKey
		pd, err := notifier.NewProvider("pagerduty", p)

		if err != nil {
			return err
		}
		config.Notifiers = append(config.Notifiers, pd)
	}

	// Configure the OpsGenie notifier.
	if config.OpsGenieAPIKey != "" {

		ogConfig := make(map[string]string)
		ogConfig["OpsGenieAPIKey"] = config.OpsGenieAPIKey
		og, err := notifier.NewProvider("opsgenie", ogConfig)

		if err != nil {
			return err
		}
		config.Notifiers = append(config.Notifiers, og)
	}

	return nil
}

// initialzeAgent setups up a number of configuration clients which depend on
// the merged configuration.
func (c *Command) initialzeAgent(config *structs.Config) (err error) {

	if config.BindAddress != base.DefaultBindAddr || config.RPCPort != base.DefaultRPCPort {
		config.RPCAddr = &net.TCPAddr{IP: net.ParseIP(config.BindAddress), Port: config.RPCPort}
	}

	// Setup telemetry
	if err = c.setupTelemetry(config.Telemetry); err != nil {
		return
	}

	// Setup notifiers
	if err = c.setupNotifier(config.Notification); err != nil {
		return
	}

	// Setup logging
	logging.SetLevel(config.LogLevel)

	// Setup the Consul and Nomad clients
	if err = base.InitializeClients(config); err != nil {
		return
	}

	return nil
}

// RPC is used to make an RPC call to Replicator.
func (c *Command) RPC(method string, reply interface{}) error {
	return c.server.RPC(method, reply)
}

// Help provides the help information for the agent command.
func (c *Command) Help() string {
	helpText := `
  Usage: replicator agent [options]

    Starts the Replicator agent and runs until an interrupt is received.
    The Replicator agent's configuration primarily comes from the config
    files used. If no config file is passed, a default config will be
    used.

  General Options:

    -bind-address=<addr>
      Specifies which address the Replicator agent will bind to for HTTP API and
      RPC network  services.

    -cluster-scaling-disable
      Passing this flag will disable cluster scaling completly.

    -cluster-scaling-interval=<seconds>
      The time period in seconds between Replicator cluster scaling
      evaluation runs.

    -config=<path>
      The path to either a single config file or a directory of config
      files to use for configuring the Replicator agent. Replicator
      processes configuration files in lexicographic order.

    -consul=<address:port>
      This is the address of the Consul agent. By default, this is
      localhost:8500, which is the default bind and port for a local
      Consul agent. It is not recommended that you communicate directly
      with a Consul server, and instead communicate with the local
      Consul agent. There are many reasons for this, most importantly
      the Consul agent is able to multiplex connections to the Consul
      server and reduce the number of open HTTP connections. Additionally,
      it provides a "well-known" IP address for which clients can connect.

    -consul-key-root=<key>
      The Consul Key/Value Store location that Replicator will use
      for persistent configuration. By default, this is replicator/config.

    -consul-token=<token>
      The Consul ACL token to use when communicating with an ACL
      protected Consul cluster.

    -dev
      Start the Replicator agent in development mode. This runs the
      Replicator agent with a configuration which is ideal for development
      or local testing.

    -http-port=<port>
      The port used to run the HTTP server.

    -job-scaling-disable
      Passing this flag will disable job scaling completly.

    -job-scaling-interval=<seconds>
      The time period in seconds between Replicator job scaling evaluation
      runs.

    -log-level=<level>
      Specify the verbosity level of Replicator's logs. The default is
      INFO.

    -nomad=<address:port>
      The address and port Replicator will use when making connections
      to the Nomad API. By default, this http://localhost:4646, which
      is the default bind and port for a local Nomad server.

    -rpc-port=<port>
       The port used for RPC listening.

  Telemetry Options:

    -statsd-address=<address:port>
      Specifies the address of a statsd server to forward metrics
      to and should include the port.

  Notifications Options:

    -cluster-identifier=<name>
      A human readable cluster name to allow operators to quickly identify
      which cluster is alerting.

    -pagerduty-service-key=<key>
      The PagerDuty integration key which has been setup to allow
      replicator to send events.

`
	return strings.TrimSpace(helpText)
}

// Synopsis is provides a brief summary of the agent command.
func (c *Command) Synopsis() string {
	return "Runs a Replicator agent"
}
