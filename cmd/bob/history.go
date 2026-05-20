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

func newHistoryCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "history <name>",
		Short: "Show build history for a BuildJob",
		Long: `List previous build runs for a BuildJob, showing the status,
timestamps, duration, and commit SHA for each PipelineRun.

Similar to 'oc get builds' in OpenShift, which shows all historical
Build objects. In bob, each BuildJob is re-triggered and the underlying
PipelineRun objects represent individual build executions.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			resp, err := c.History(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("getting build history: %w", err)
			}

			if len(resp.Entries) == 0 {
				fmt.Println("No build history found.")
				fmt.Println("The BuildJob may not have been triggered yet.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "RUN\tNAME\tPHASE\tSTARTED\tDURATION\tCOMMIT")
			for _, e := range resp.Entries {
				commit := e.CommitSHA
				if len(commit) > 8 {
					commit = commit[:8]
				}
				duration := e.Duration
				if duration == "" {
					duration = "-"
				}
				started := e.StartedAt
				if started == "" {
					started = "-"
				}
				fmt.Fprintf(w, "#%d\t%s\t%s\t%s\t%s\t%s\n",
					e.Run, e.Name, e.Phase, started, duration, commit)
			}
			return w.Flush()
		},
	}
}
