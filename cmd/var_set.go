/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

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
	"io/ioutil"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

// setCmd represents the set command
var varSetCmd = &cobra.Command{
	Use:   "set KEY=VALUE",
	Short: "Set environment variables for your services. Set multiple by separating them with a space. ",
	Long:  "Set environment variables for your services. Set multiple by separating them with a space. For ex. nucleus var set KEY1=VALUE1 KEY2=VALUE2 KEY3=VALUE3.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("must provide at least one environment variable key-value pair")
		}

		err := storeVars("./nucleus.yaml", args)
		if err != nil {
			return err
		}

		return nil
	},
}

func storeVars(fileName string, args []string) error {
	file, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}
	root := SpecStruct{}
	err = yaml.Unmarshal(file, &root)

	if err != nil {
		return err
	}

	if root.Vars == nil {
		root.Vars = make(map[string]string)
	}

	for i := 0; i < len(args); i++ {
		s := strings.Split(args[i], "=")
		root.Vars[s[0]] = s[1]
	}

	newBlob, err := yaml.Marshal(root)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(fileName, newBlob, 0644)
	if err != nil {
		return errors.New("unable to write data into the file")
	}

	return nil
}

func init() {
	varCmd.AddCommand(varSetCmd)
}
