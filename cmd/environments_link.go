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
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/nucleuscloud/cli/internal/pkg/utils"
	"github.com/spf13/cobra"

	svcmgmtv1alpha1 "github.com/nucleuscloud/mgmt-api/gen/proto/go/servicemgmt/v1alpha1"
)

var envsLinkCmd = &cobra.Command{
	Use:   "link",
	Short: "Links an admission api to one or all environments",
	Long:  "Call this command to link an admission api to one or all environments. Will default to all if no env type is provided.",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		environmentTypes, err := cmd.Flags().GetStringArray("env")
		if err != nil {
			return err
		}

		if len(args) == 0 {
			return fmt.Errorf("must provide admission api url as argument")
		}

		admissionApiUrl := args[0]
		_, err = url.ParseRequestURI(admissionApiUrl)
		if err != nil {
			return err
		}

		if len(environmentTypes) == 0 {
			environmentTypes = utils.ValidEnvTypes
		}

		for _, envType := range environmentTypes {
			if !utils.IsValidEnvironmentType(envType) {
				return fmt.Errorf("invalid value for environment")
			}
		}

		for _, envType := range environmentTypes {
			if envType == "prod" {
				err := utils.PromptToProceed(cmd, "prod", "yes")
				if err != nil {
					return err
				}
			}
		}

		fmt.Printf("Setting admission api to the following envs: %s\n", strings.Join(environmentTypes, ","))
		return linkAdmissionApi(ctx, environmentTypes, admissionApiUrl)
	},
}

func init() {
	environmentsCmd.AddCommand(envsLinkCmd)

	envsLinkCmd.Flags().StringArrayP("env", "e", []string{}, "set the nucleus environment. may provide multiple times to apply to one or more environments")
	envsLinkCmd.Flags().BoolP("yes", "y", false, "automatically answer yes to the prod prompt")
}

func linkAdmissionApi(ctx context.Context, environmentTypes []string, admissionApiUrl string) error {
	conn, err := utils.NewApiConnectionByEnv(ctx, utils.GetEnv())
	if err != nil {
		return err
	}
	defer conn.Close()

	client := svcmgmtv1alpha1.NewServiceMgmtServiceClient(conn)

	var wg sync.WaitGroup
	wgDone := make(chan bool)
	errChan := make(chan error)

	for _, envType := range environmentTypes {
		wg.Add(1)

		go func(environmentType string) {
			defer wg.Done()
			_, err = client.SetAdmissionApiLink(ctx, &svcmgmtv1alpha1.SetAdmissionApiLinkRequest{
				EnvironmentType: environmentType,
				AdmissionApiUrl: admissionApiUrl,
			})
			if err != nil {
				errChan <- err
				return
			}
		}(envType)
	}

	go func() {
		wg.Wait()
		close(wgDone)
	}()

	select {
	case <-wgDone:
		break
	case err := <-errChan:
		close(errChan)
		return err
	}
	return nil
}
