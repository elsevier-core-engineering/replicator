package main

import (
	"os"

	"github.com/d3sw/replicator/command"
	"github.com/d3sw/replicator/command/agent"
	"github.com/d3sw/replicator/version"
	"github.com/mitchellh/cli"
)

// Commands returns the mapping of CLI commands for Replicator. The meta
// parameter lets you set meta options for all commands.
func Commands(metaPtr *command.Meta) map[string]cli.CommandFactory {
	if metaPtr == nil {
		metaPtr = new(command.Meta)
	}

	meta := *metaPtr
	if meta.UI == nil {
		meta.UI = &cli.BasicUi{
			Reader:      os.Stdin,
			Writer:      os.Stdout,
			ErrorWriter: os.Stderr,
		}
	}

	return map[string]cli.CommandFactory{
		"agent": func() (cli.Command, error) {
			return &agent.Command{
				Meta: meta,
			}, nil
		},
		"init": func() (cli.Command, error) {
			return &command.InitCommand{
				Meta: meta,
			}, nil
		},
		"failsafe": func() (cli.Command, error) {
			return &command.FailsafeCommand{
				Meta: meta,
			}, nil
		},
		"version": func() (cli.Command, error) {
			ver := version.Version
			rel := version.VersionPrerelease

			if rel == "" && version.VersionPrerelease != "" {
				rel = "dev"
			}

			return &command.VersionCommand{
				Version:           ver,
				VersionPrerelease: rel,
				UI:                meta.UI,
			}, nil
		},
	}
}
