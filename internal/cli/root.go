package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	verbose bool
	server  string
)

var rootCmd = &cobra.Command{
	Use:   "athyr-agent",
	Short: "YAML-driven agent runner for Athyr",
	Long: `athyr-agent runs AI agents defined in YAML files.

It connects to an Athyr server and manages the agent lifecycle,
allowing you to define simple agents without writing Go code.

Example:
  athyr-agent run agent.yaml --server localhost:9090
  athyr-agent validate agent.yaml`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.athyr-agent.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().StringVar(&server, "server", "localhost:9090", "Athyr server address")

	// Bind flags to viper
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("server", rootCmd.PersistentFlags().Lookup("server"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Warning: could not find home directory:", err)
			return
		}

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".athyr-agent")
	}

	viper.SetEnvPrefix("ATHYR_AGENT")
	viper.AutomaticEnv()

	// Read config file if it exists (silently ignore if not found)
	_ = viper.ReadInConfig()
}
