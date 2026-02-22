package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile  string
	verbose  bool
	server   string
	insecure bool
)

var rootCmd = &cobra.Command{
	Use:   "athyr-pipe",
	Short: "Unix-native agent piping for Athyr",
	Long: `athyr-pipe brings AI agents to Unix pipelines.

It connects to an Athyr server and lets you compose agents with
standard Unix tools using stdin/stdout.

Example:
  echo "summarize this" | athyr-pipe --model openai/gpt-4o
  cat report.md | athyr-pipe --preset review
  athyr-pipe serve --port 8081`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.athyr-pipe.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().StringVarP(&server, "server", "s", "localhost:9090", "Athyr server address")
	rootCmd.PersistentFlags().BoolVar(&insecure, "insecure", false, "disable TLS verification")

	// Bind flags to viper
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("server", rootCmd.PersistentFlags().Lookup("server"))
	viper.BindPFlag("insecure", rootCmd.PersistentFlags().Lookup("insecure"))
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
		viper.SetConfigName(".athyr-pipe")
	}

	viper.SetEnvPrefix("ATHYR_PIPE")
	viper.AutomaticEnv()

	// Read config file if it exists (silently ignore if not found)
	_ = viper.ReadInConfig()
}
