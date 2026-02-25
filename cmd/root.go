package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "bbranch",
	Short: "Create git branches across multiple Bitbucket repos",
	Long:  "A CLI tool to create branches across multiple Bitbucket Cloud repositories simultaneously.",
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: .bbranch.yaml)")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName(".bbranch")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")

		home, err := os.UserHomeDir()
		if err == nil {
			viper.AddConfigPath(home)
		}
	}

	// Silently ignore missing config — login/config init don't need it
	viper.ReadInConfig()
}
