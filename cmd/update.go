//  Copyright 2016 Red Hat, Inc.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package cmd

import (
	"github.com/funktionio/funktion/pkg/update"
	"os"

	"github.com/spf13/cobra"
)

type updateCmd struct {
}

func init() {
	RootCmd.AddCommand(newUpdateCmd())
}

func newUpdateCmd() *cobra.Command {
	p := &updateCmd{}
	cmd := &cobra.Command{
		Use:   "update",
		Short: "updates this binary to the latest version from github",
		Long:  `This command checks if there is a newer release of the funktion binary and if so downloads and replaces the file`,
		Run: func(cmd *cobra.Command, args []string) {
			handleError(p.run())
		},
	}
	return cmd
}

func (p *updateCmd) run() error {
	update.MaybeUpdateFromGithub(os.Stdout)
	return nil
}
