// Package images implements the CLI command for working with container images from OLM bundles.
package images

import (
	"github.com/spf13/cobra"

	"github.com/lburgazzoli/olm-extractor/cmd/images/ls"
)

const (
	shortDescription = "Work with container images from OLM bundles"

	longDescription = `Work with container images from OLM bundles.

This command provides subcommands for inspecting and extracting container
image references from OLM bundles, useful for disconnected/airgapped
installation scenarios.`
)

// NewCommand creates the images command with subcommands.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "images",
		Short: shortDescription,
		Long:  longDescription,
	}

	// Add subcommands
	cmd.AddCommand(ls.NewCommand())

	return cmd
}
