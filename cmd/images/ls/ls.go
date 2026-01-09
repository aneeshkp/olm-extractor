// Package ls implements the CLI subcommand for extracting related images from OLM bundles.
package ls

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/lburgazzoli/olm-extractor/pkg/bundle"
	"github.com/lburgazzoli/olm-extractor/pkg/catalog"
	"github.com/lburgazzoli/olm-extractor/pkg/images"
)

const (
	// OutputText is the text output format (one image per line).
	OutputText = "text"
	// OutputJSON is the JSON output format.
	OutputJSON = "json"

	// EnvPrefix is the environment variable prefix for configuration.
	EnvPrefix = "BUNDLE_EXTRACT"
)

// Config holds all configuration for the images subcommand.
type Config struct {
	images.Config `mapstructure:",squash"`

	Output   string                `mapstructure:"output"`
	TempDir  string                `mapstructure:"temp-dir"`
	Catalog  string                `mapstructure:"catalog"`
	Channel  string                `mapstructure:"channel"`
	Registry bundle.RegistryConfig `mapstructure:",squash"`
}

const (
	shortDescription = "List container images from an OLM bundle"

	longDescription = `Extract related container images from an OLM bundle.

This command extracts container image references from an OLM bundle, useful for
disconnected/airgapped installations where all images must be known in advance.

By default, it extracts images from environment variables matching the RELATED_IMAGE
prefix (the OLM convention for related images). Use --env-pattern to match different
prefixes.

Output formats:
  - text (default): One image per line, suitable for piping to other tools
  - json: JSON array of image references

All flags can be configured using environment variables with the BUNDLE_EXTRACT_ prefix.`

	exampleUsage = `  # List related images from a bundle (one per line)
  bundle-extract images ls ./path/to/bundle

  # List images from a bundle container image
  bundle-extract images ls quay.io/example/operator-bundle:v1.0.0

  # List images from a catalog
  bundle-extract images ls --catalog quay.io/operatorhubio/catalog:latest prometheus

  # Exclude operator container images (only related images)
  bundle-extract images ls --include-operator-images=false ./bundle

  # Output as JSON
  bundle-extract images ls --output=json ./bundle

  # Use custom environment variable pattern
  bundle-extract images ls --env-pattern=MY_IMAGE ./bundle

  # Pipe to skopeo for mirroring
  bundle-extract images ls ./bundle | xargs -I{} skopeo copy docker://{} docker://mirror.example.com/{}`

	tempDirPerms = 0750
)

// NewCommand creates the ls subcommand for listing images.
func NewCommand() *cobra.Command {
	// Create a local viper instance to avoid conflicts with other subcommands
	v := viper.New()
	v.SetEnvPrefix(EnvPrefix)
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	cmd := &cobra.Command{
		Use:          "ls <bundle-path-or-image>",
		Short:        shortDescription,
		Long:         longDescription,
		Example:      exampleUsage,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Bind flags after command is parsed
			if err := v.BindPFlags(cmd.Flags()); err != nil {
				return fmt.Errorf("failed to bind flags: %w", err)
			}

			return execute(cmd.Context(), cmd.OutOrStdout(), v, args[0])
		},
	}

	// Define flags
	cmd.Flags().StringP("output", "o", OutputText, "Output format: text or json")
	cmd.Flags().StringP("env-pattern", "e", images.DefaultEnvPattern, "Environment variable prefix to match")
	cmd.Flags().Bool("include-operator-images", true, "Include operator container images in output")
	cmd.Flags().String("temp-dir", "", "Directory for temporary files and cache")
	cmd.Flags().String("catalog", "", "Catalog image to resolve bundle from")
	cmd.Flags().String("channel", "", "Channel to use when resolving from catalog")
	cmd.Flags().Bool("registry-insecure", false, "Allow insecure connections to registries")
	cmd.Flags().String("registry-username", "", "Username for registry authentication")
	cmd.Flags().String("registry-password", "", "Password for registry authentication")

	return cmd
}

// execute runs the image extraction.
func execute(
	ctx context.Context,
	out io.Writer,
	v *viper.Viper,
	input string,
) error {
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("failed to parse configuration: %w", err)
	}

	// Create temp directory if specified and doesn't exist
	if cfg.TempDir != "" {
		if err := os.MkdirAll(cfg.TempDir, tempDirPerms); err != nil {
			return fmt.Errorf("failed to create temp-dir: %w", err)
		}
	}

	// Resolve bundle source (handles catalog mode)
	bundleImageOrDir, err := catalog.ResolveBundleSource(
		ctx,
		input,
		cfg.Catalog,
		cfg.Channel,
		cfg.Registry,
		cfg.TempDir,
	)
	if err != nil {
		return fmt.Errorf("failed to resolve bundle source: %w", err)
	}

	// Load bundle
	b, err := bundle.Load(ctx, bundleImageOrDir, cfg.Registry, cfg.TempDir)
	if err != nil {
		return fmt.Errorf("failed to load bundle: %w", err)
	}

	// Extract images
	result, err := images.Extract(b, cfg.Config)
	if err != nil {
		return fmt.Errorf("failed to extract images: %w", err)
	}

	// Output based on format
	switch cfg.Output {
	case OutputJSON:
		return outputJSON(out, result, cfg.Config)
	case OutputText:
		return outputText(out, result, cfg.Config)
	default:
		return fmt.Errorf("unsupported output format: %s (use '%s' or '%s')", cfg.Output, OutputText, OutputJSON)
	}
}

// outputJSON writes all images as a JSON array.
func outputJSON(w io.Writer, result *images.Result, cfg images.Config) error {
	var imagesToPrint []string
	if cfg.IncludeOperatorImages {
		imagesToPrint = result.AllImages()
	} else {
		imagesToPrint = result.RelatedImages()
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(imagesToPrint); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

// outputText writes images as one per line.
func outputText(w io.Writer, result *images.Result, cfg images.Config) error {
	var imagesToPrint []string
	if cfg.IncludeOperatorImages {
		imagesToPrint = result.AllImages()
	} else {
		imagesToPrint = result.RelatedImages()
	}

	for _, img := range imagesToPrint {
		if _, err := fmt.Fprintln(w, img); err != nil {
			return fmt.Errorf("failed to write image: %w", err)
		}
	}

	return nil
}
