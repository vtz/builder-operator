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

	"github.com/spf13/cobra"

	bobclient "github.com/centos-automotive-suite/bob/internal/buildapi/client"
)

var (
	bobServer    string
	bobToken     string
	bobNamespace string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "bob",
		Short: "bob — the builder. Build embedded software on OpenShift.",
		Long: `bob is a CLI for building embedded software artifacts on OpenShift.
It talks to the bob Build API to manage BuildJob resources.

Configure via environment variables:
  BOB_SERVER     Build API URL (e.g. https://bob-api.apps.cluster.example.com)
  BOB_TOKEN      Auth token (e.g. oc whoami -t)
  BOB_NAMESPACE  Target namespace (default: bob-builds)`,
	}

	rootCmd.PersistentFlags().StringVar(&bobServer, "server", "", "Build API server URL (env: BOB_SERVER)")
	rootCmd.PersistentFlags().StringVar(&bobToken, "token", "", "Auth token (env: BOB_TOKEN)")
	rootCmd.PersistentFlags().StringVarP(&bobNamespace, "namespace", "n", "", "Target namespace (env: BOB_NAMESPACE)")

	rootCmd.AddCommand(
		newListCmd(),
		newBuildCmd(),
		newShowCmd(),
		newLogsCmd(),
		newDeleteCmd(),
		newArtifactsCmd(),
		newSyncCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func newClient() *bobclient.Client {
	server := firstNonEmpty(bobServer, os.Getenv("BOB_SERVER"))
	token := firstNonEmpty(bobToken, os.Getenv("BOB_TOKEN"))
	namespace := firstNonEmpty(bobNamespace, os.Getenv("BOB_NAMESPACE"), "bob-builds")

	if server == "" {
		fmt.Fprintln(os.Stderr, "error: BOB_SERVER is required (set via --server or BOB_SERVER env var)")
		os.Exit(1)
	}
	return bobclient.New(server, token, namespace)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
