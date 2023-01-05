/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/nucleuscloud/cli/internal/utils"
	"github.com/spf13/cobra"

	"github.com/spf13/viper"
)

const (
	nucleusDirName           = ".nucleus"
	cliSettingsFileNameNoExt = ".nucleus-cli"
	cliSettingsFileExt       = "yaml"
)

var verbose bool

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "nucleus",
	Short: "Terminal UI that interfaces with the Nucleus system.",
	Long:  "Terminal UI that allows authenticated access to the Nucleus system.\nThis CLI allows you to deploy and manage all of the environments and services within your Nucleus account or accounts.",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	utils.CheckErr(rootCmd.Execute())
}

func init() {
	var cfgFile string
	cobra.OnInitialize(func() { initConfig(cfgFile) })

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", fmt.Sprintf("config file (default is $HOME/%s.%s)", cliSettingsFileNameNoExt, cliSettingsFileExt))
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Silences errors because we print them in Execute()
	rootCmd.SilenceErrors = true
}

// initConfig reads in config file and ENV variables if set.
func initConfig(cfgFilePath string) {
	if cfgFilePath != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFilePath)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		fullNucluesSettingsDir := fmt.Sprintf("%s/%s", home, nucleusDirName)

		// Search config in home directory with name ".nucleus-cli" (without extension).
		// Higher priority is first. (which seems like the opposite to me, but that is how it works ¯\_(ツ)_/¯)
		viper.AddConfigPath(".")
		viper.AddConfigPath(fullNucluesSettingsDir)
		viper.AddConfigPath(home)
		viper.SetConfigType(cliSettingsFileExt)
		viper.SetConfigName(cliSettingsFileNameNoExt)
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	err := viper.ReadInConfig()
	if err != nil {
		if !errors.As(err, &viper.ConfigFileNotFoundError{}) {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
			return
		}
	}
	cfgUsed := viper.ConfigFileUsed()
	if cfgUsed != "" {
		fmt.Println("Using config file:", cfgUsed)
	}
}
