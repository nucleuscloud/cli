package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/nucleuscloud/cli/internal/version"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the client version information",
	Long:  "Print the client version information for the current context",
	RunE: func(cmd *cobra.Command, args []string) error {
		output, err := cmd.Flags().GetString("output")
		if err != nil {
			return err
		}
		if output != "" && output != "json" && output != "yaml" {
			return fmt.Errorf("must provide valid output")
		}
		versionInfo := version.Get()

		if output == "json" {
			marshalled, err := json.MarshalIndent(&versionInfo, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(marshalled))
		} else if output == "yaml" {
			marshalled, err := yaml.Marshal(&versionInfo)
			if err != nil {
				return err
			}
			fmt.Println(string(marshalled))
		} else {
			fmt.Println("Git Version:", versionInfo.GitVersion)
			fmt.Println("Git Commit:", versionInfo.GitCommit)
			fmt.Println("Build Date:", versionInfo.BuildDate)
			fmt.Println("Go Version:", versionInfo.GoVersion)
			fmt.Println("Compiler:", versionInfo.Compiler)
			fmt.Println("Platform:", versionInfo.Platform)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)

	versionCmd.Flags().StringP("output", "o", "", "json|yaml")
}
