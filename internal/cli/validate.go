package cli

import (
	"fmt"
	"os"

	"github.com/athyr-tech/athyr-agent/internal/config"

	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate <file>",
	Short: "Validate an agent YAML file",
	Long: `Validate an agent YAML file without running it.

Checks that the YAML is well-formed and contains all required fields.

Example:
  athyr-agent validate agent.yaml`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filepath := args[0]

		cfg, err := config.LoadFile(filepath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Validation failed: %v\n", err)
			return err
		}

		if err := cfg.Validate(); err != nil {
			fmt.Fprintf(os.Stderr, "Validation failed: %v\n", err)
			return err
		}

		fmt.Printf("Agent '%s' is valid.\n", cfg.Agent.Name)
		if verbose {
			fmt.Printf("  description: %s\n", cfg.Agent.Description)
			fmt.Printf("  model:       %s\n", cfg.Agent.Model)
			fmt.Printf("  subscribe:   %v\n", cfg.Agent.Topics.Subscribe)
			fmt.Printf("  publish:     %v\n", cfg.Agent.Topics.Publish)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}
