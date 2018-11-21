package command

import (
	"fmt"
	"strings"

	"github.com/d3sw/replicator/command/base"
	"github.com/d3sw/replicator/notifier"
	"github.com/d3sw/replicator/replicator"
	"github.com/d3sw/replicator/replicator/structs"
)

// FailsafeCommand is a command implementation that allows operators to
// place the daemon in or take the daemon out of failsafe mode.
type FailsafeCommand struct {
	args []string
	Meta
	statePath string
}

// Help provides the help information for the failsafe command.
func (c *FailsafeCommand) Help() string {
	helpText := `
Usage: replicator failsafe [options]

  Allows an operator to administratively control the failsafe behavior of
  Replicator. When Replicator places a job or worker pool in failsafe mode, 
  all running copies of Replicator will prohibit any scaling operations
  against the resource in question.

  Failsafe mode is intended to stabilize a respource that has experienced
  critical failures while attempting to perform scaling operations.

  To exit failsafe mode, an operator must explicitly remove the failsafe
  lock after identifying the root cause of the failures.

  General Options:

    -config=<path>
      The path to either a single config file or a directory of config
      files to use when configuring the Replicator agent. Replicator
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

    -consul-token=<token>
      The Consul ACL token to use when communicating with an ACL protected
      Consul cluster.

    -state-path=<key>
      The full Consul Key/Value path where the state object for the resource
      is stored. By default, Replicator stores all state objects at a common
      base path, replicator/config/state; within this base context, worker
      pools state is stored at nodes/<pool_name> and jobs at
      jobs/<job_name>/<group_name>.

  Failsafe Mode Options:

    -disable
      Disable the failsafe lock on the provided resource. All copies of
      Replicator will return to normal operations.

    -enable
      Enable the failsafe lock on the provided resource. All copies of
      Replicator will be prohibited from taking any scaling actions on
      the failsafe enabled jobgroup or worker pool.

    -force
      Suppress confirmation prompts when enabling or disabling the failsafe
      lock.
`
	return strings.TrimSpace(helpText)
}

// Synopsis is provides a brief summary of the failsafe command.
func (c *FailsafeCommand) Synopsis() string {
	return "Provide an administrative interface to control failsafe mode"
}

// Run triggers the failsafe command to update the distributed state tracking
// data and manipulate the distributed failsafe lock.
func (c *FailsafeCommand) Run(args []string) int {
	// Initialize a new empty state tracking object.
	state := &structs.ScalingState{}

	// The operator must specify at least one operation.
	if len(args) == 0 {
		c.UI.Error(c.Help())
		return 1
	}

	// Parse flags and generate a resulting configuration.
	c.args = args
	conf := c.parseFlags()
	if conf == nil {
		return 1
	}

	// Set the state path in the state object.
	state.StatePath = c.statePath

	// Setup the Consul client.
	if err := base.InitializeClients(conf.Config); err != nil {
		c.UI.Error(fmt.Sprintf("An error occurred while attempting to initialize "+
			"the Consul client: %v", err))
		return 1
	}

	// Grab the initialized consul client from the returned configuration object.
	consul := conf.Config.ConsulClient

	// Check that we were sent either enable or disable, but not both.
	if (conf.Enable && conf.Disable) || (!conf.Enable && !conf.Disable) {
		c.UI.Error(c.Help())
		return 1
	}

	// Attempt to load state tracking data from Consul.
	consul.ReadState(state, false)

	// If there was no state object at the specified path, throw an error.
	if state.LastUpdated.IsZero() {
		c.UI.Error(fmt.Sprintf("No state object was found at the specified "+
			"path %v, no action will be taken", c.statePath))
		return 1
	}

	// If failsafe mode is already in the desired state, report and take no
	// action.
	if state.FailsafeMode && conf.Enable || !state.FailsafeMode && conf.Disable {
		c.UI.Warn(fmt.Sprintf("Failsafe mode is already in desired state \"%vd\""+
			", no action required.", conf.Verb))
		return 0
	}

	// If the user has not disabled confirmation prompts, ask for confirmation.
	if !conf.Force {
		question := fmt.Sprintf("Are you sure you want to %s failsafe mode for "+
			"%v %v at location %q?\n", conf.Verb, state.ResourceType,
			state.ResourceName, c.statePath)

		// If we're enabling failsafe mode, give the user a clear warning about
		// the implications.
		if conf.Enable {
			question = fmt.Sprintf("%vNo scaling operations will be permitted "+
				"from any running copies of Replicator.\n", question)
		}

		// Ask for confirmation and parse the response.
		answer, err := c.UI.Ask(fmt.Sprintf("%vConfirm [y/N]: ", question))
		if err != nil {
			c.UI.Error(fmt.Sprintf("Failed to parse answer: %v", err))
			return 1
		}

		// Validate the confirmation response.
		if answer == "" || strings.ToLower(answer)[0] == 'n' {
			c.UI.Output(fmt.Sprintf("Cancelling, will not %v failsafe mode.",
				conf.Verb))
			return 0
		} else if strings.ToLower(answer)[0] == 'y' && len(answer) > 1 {
			c.UI.Output("For confirmation, an exact 'y' is required.")
			return 0
		} else if answer != "y" {
			c.UI.Output("No confirmation detected. For confirmation, an exact 'y' " +
				"is required.")
			return 1
		}
	}

	// Indicate that failsafe mode was administratively updated.
	state.FailsafeAdmin = true

	// Setup a failure message to pass to the failsafe method.
	message := &notifier.FailureMessage{
		AlertUID:     "replicator-failsafe-admin-cli",
		ResourceID:   state.ResourceName,
		ResourceType: state.ResourceType,
	}

	// Set desired failsafe mode.
	err := replicator.SetFailsafeMode(state, conf.Config, conf.Enable, message)
	if err != nil {
		c.UI.Error(fmt.Sprintf("An error occurred while attempting to %v "+
			"failsafe mode for %v %v: %v", conf.Verb, state.ResourceType,
			state.ResourceName, err))
		return 1
	}

	if err := consul.PersistState(state); err != nil {
		c.UI.Error(fmt.Sprintf("An error occurred while attempting to %v "+
			"failsafe mode for %v %v: %v", conf.Verb, state.ResourceType,
			state.ResourceName, err))
	}

	c.UI.Info(fmt.Sprintf("Successfully %vd failsafe mode for %v %v at "+
		"location %s", conf.Verb, state.ResourceType, state.ResourceName,
		c.statePath))

	return 0
}

func (c *FailsafeCommand) parseFlags() *structs.FailsafeMode {
	var configPath string

	// Initialize an empty configuration object that will be populated with
	// any passed CLI flags for later merging.
	cliConfig := &structs.FailsafeMode{
		Config: &structs.Config{},
	}

	// Initialize command flags.
	flags := c.Meta.FlagSet("failsafe", FlagSetClient)
	flags.Usage = func() { c.UI.Error(c.Help()) }

	// General configuration flags.
	flags.StringVar(&configPath, "config", "", "")
	flags.StringVar(&cliConfig.Config.Consul, "consul", "", "")
	flags.StringVar(&cliConfig.Config.ConsulToken, "consul-token", "", "")
	flags.StringVar(&c.statePath, "state-path", "", "")

	// Failsafe mode configuration flags.
	flags.BoolVar(&cliConfig.Enable, "enable", false, "")
	flags.BoolVar(&cliConfig.Disable, "disable", false, "")
	flags.BoolVar(&cliConfig.Force, "force", false, "")

	// Parse the passed CLI flags.
	if err := flags.Parse(c.args); err != nil {
		return nil
	}

	// Determine the appropriate verbage for confirmation prompts.
	cliConfig.Verb = "enable"
	if cliConfig.Disable {
		cliConfig.Verb = "disable"
	}

	// Create default configuration object on which to base the merge.
	config := base.DefaultConfig()

	// If a configuration path has been specified, load configuration from the
	// specified location.
	if configPath != "" {
		current, err := base.LoadConfig(configPath)
		if err != nil {
			c.UI.Error(fmt.Sprintf("Error loading configuration from %s: %s",
				configPath, err))
			return nil
		}

		// Merge loaded configuration with the default configuration.
		config = config.Merge(current)
	}

	// Merge passed CLI flags with the configuration derived from the defaults
	// and optionally, the loaded configuration.
	cliConfig.Config = config.Merge(cliConfig.Config)

	return cliConfig
}
