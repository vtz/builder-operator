package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/centos-automotive-suite/bob/internal/buildapi"
	bobclient "github.com/centos-automotive-suite/bob/internal/buildapi/client"
)

func newArtifactsCmd() *cobra.Command {
	var downloadDir string

	cmd := &cobra.Command{
		Use:     "artifacts [name]",
		Short:   "List or download build artifacts",
		Aliases: []string{"art"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			name := args[0]
			ctx := context.Background()

			build, err := c.Get(ctx, name)
			if err != nil {
				return fmt.Errorf("getting build: %w", err)
			}

			if build.OCIArtifactRef != "" {
				fmt.Printf("Artifacts pushed as OCI artifact:\n\n")
				fmt.Printf("  Reference:  %s\n", build.OCIArtifactRef)
				if build.OCIArtifactDigest != "" {
					fmt.Printf("  Digest:     %s\n", build.OCIArtifactDigest)
				}
				fmt.Println()
				fmt.Printf("Pull with:\n")
				if build.OCIArtifactDigest != "" {
					fmt.Printf("  oras pull %s@%s --output ./\n\n", build.OCIArtifactRef, build.OCIArtifactDigest)
				} else {
					fmt.Printf("  oras pull %s --output ./\n\n", build.OCIArtifactRef)
				}
				fmt.Printf("Inspect manifest:\n")
				fmt.Printf("  oras manifest fetch %s | jq\n", build.OCIArtifactRef)
				return nil
			}

			resp, err := c.ListArtifacts(ctx, name)
			if err != nil {
				return fmt.Errorf("listing artifacts: %w", err)
			}
			if len(resp.Files) == 0 {
				fmt.Println("No artifacts available.")
				fmt.Println("The build may still be running, or artifacts have not been uploaded yet.")
				return nil
			}

			if downloadDir != "" {
				return downloadAllArtifacts(ctx, c, name, resp.Files, downloadDir)
			}

			tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(tw, "NAME\tSIZE")
			for _, f := range resp.Files {
				fmt.Fprintf(tw, "%s\t%s\n", f.Name, humanSize(f.Size))
			}
			tw.Flush()

			fmt.Printf("\nDownload with: bob artifacts %s --download ./out\n", name)
			return nil
		},
	}

	cmd.Flags().StringVarP(&downloadDir, "download", "d", "", "Download all artifacts to this directory")
	return cmd
}

func downloadAllArtifacts(ctx context.Context, c *bobclient.Client, name string, files []buildapi.ArtifactFileInfo, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	for _, f := range files {
		fmt.Printf("  downloading %s (%s)...\n", f.Name, humanSize(f.Size))
		body, err := c.DownloadArtifact(ctx, name, f.Name)
		if err != nil {
			return fmt.Errorf("downloading %s: %w", f.Name, err)
		}
		outPath := filepath.Join(dir, f.Name)
		out, err := os.Create(outPath)
		if err != nil {
			body.Close()
			return fmt.Errorf("creating %s: %w", outPath, err)
		}
		_, err = io.Copy(out, body)
		body.Close()
		out.Close()
		if err != nil {
			return fmt.Errorf("writing %s: %w", outPath, err)
		}
	}
	fmt.Printf("Downloaded %d artifacts to %s/\n", len(files), dir)
	return nil
}

func humanSize(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
