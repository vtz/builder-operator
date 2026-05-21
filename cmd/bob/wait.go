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
	"context"
	"fmt"
	"os"
	"time"

	"github.com/centos-automotive-suite/bob/internal/buildapi"
	bobclient "github.com/centos-automotive-suite/bob/internal/buildapi/client"
)

const (
	buildPollInterval   = 5 * time.Second
	buildStartupTimeout = 2 * time.Minute
)

func waitAndDownload(ctx context.Context, name, downloadDir string, skipVerify bool) error {
	c := newClient()

	initial, err := c.Get(ctx, name)
	if err != nil {
		return fmt.Errorf("getting build status: %w", err)
	}
	previousRun := initial.PipelineRun

	fmt.Printf("\nWaiting for build %q to complete...\n", name)
	start := time.Now()
	lastPhase := ""
	buildStarted := false

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		build, err := c.Get(ctx, name)
		if err != nil {
			return fmt.Errorf("polling build status: %w", err)
		}

		if !buildStarted {
			switch {
			case build.PipelineRun != previousRun:
				buildStarted = true
				fmt.Printf("  New PipelineRun: %s\n", build.PipelineRun)
			case build.Phase == "Running" || build.Phase == "Pending":
				buildStarted = true
				fmt.Printf("  PipelineRun: %s\n", build.PipelineRun)
			case build.Phase == "Succeeded" || build.Phase == "Failed":
				if time.Since(start) > buildStartupTimeout {
					fmt.Printf("  Build already completed (run %s), downloading existing artifacts.\n", build.PipelineRun)
					buildStarted = true
				} else {
					time.Sleep(buildPollInterval)
					continue
				}
			}
		}

		if !buildStarted {
			time.Sleep(buildPollInterval)
			continue
		}

		if build.Phase != lastPhase {
			elapsed := time.Since(start).Truncate(time.Second)
			fmt.Printf("  [%s] Phase: %s\n", elapsed, build.Phase)
			lastPhase = build.Phase
		}

		switch build.Phase {
		case "Succeeded":
			fmt.Printf("\nBuild succeeded in %s\n", time.Since(start).Truncate(time.Second))
			return downloadBuildArtifacts(ctx, c, build, downloadDir, skipVerify)
		case "Failed":
			fmt.Fprintf(os.Stderr, "\nBuild failed after %s\n", time.Since(start).Truncate(time.Second))
			fmt.Fprintf(os.Stderr, "View logs with: bob logs %s\n", name)
			return fmt.Errorf("build %q failed", name)
		}

		time.Sleep(buildPollInterval)
	}
}

func downloadBuildArtifacts(ctx context.Context, c *bobclient.Client, build *buildapi.BuildJobSummary, dir string, skipVerify bool) error {
	if build.OCIArtifactRef != "" {
		if build.OCISigned && !skipVerify {
			if err := verifyCosignSignature(build.OCIArtifactRef, build.OCIArtifactDigest); err != nil {
				fmt.Fprintf(os.Stderr, "WARNING: signature verification failed: %v\n", err)
				fmt.Fprintf(os.Stderr, "Use --skip-verify to bypass signature checks\n\n")
				return fmt.Errorf("signature verification failed: %w", err)
			}
			fmt.Printf("Signature verified OK\n\n")
		}
		return downloadOCIArtifact(build.OCIArtifactRef, build.OCIArtifactDigest, dir)
	}

	resp, err := c.ListArtifacts(ctx, build.Name)
	if err != nil {
		return fmt.Errorf("listing artifacts: %w", err)
	}
	if len(resp.Files) == 0 {
		fmt.Println("Build succeeded but no artifacts found.")
		return nil
	}

	return downloadAllArtifacts(ctx, c, build.Name, resp.Files, dir)
}
