// Copyright 2026 Red Hat Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all BuildJobs",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			items, err := c.List(cmd.Context())
			if err != nil {
				return fmt.Errorf("listing build jobs: %w", err)
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tPHASE\tBOARD\tPLATFORM\tIMAGE\tSOURCE")
			for _, item := range items {
				source := ""
				if item.Source != nil {
					source = item.Source.URL
					if item.Source.Revision != "" && item.Source.Revision != "main" {
						source += "@" + item.Source.Revision
					}
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
					item.Name, item.Phase, item.Board, item.Platform, item.Image, source)
			}
			return w.Flush()
		},
	}
}
