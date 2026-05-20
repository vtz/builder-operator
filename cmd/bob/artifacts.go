package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/centos-automotive-suite/bob/internal/buildapi"
	bobclient "github.com/centos-automotive-suite/bob/internal/buildapi/client"
)

func newArtifactsCmd() *cobra.Command {
	var downloadDir string
	var skipVerify bool

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
				if downloadDir != "" {
					if build.OCISigned && !skipVerify {
						if err := verifyCosignSignature(build.OCIArtifactRef, build.OCIArtifactDigest); err != nil {
							fmt.Fprintf(os.Stderr, "WARNING: signature verification failed: %v\n", err)
							fmt.Fprintf(os.Stderr, "Use --skip-verify to bypass signature checks\n\n")
							return fmt.Errorf("signature verification failed: %w", err)
						}
						fmt.Printf("Signature verified OK\n\n")
					}
					return downloadOCIArtifact(build.OCIArtifactRef, build.OCIArtifactDigest, downloadDir)
				}
				fmt.Printf("Artifacts pushed as OCI artifact:\n\n")
				fmt.Printf("  %s\n", build.OCIArtifactRef)
				if build.OCIArtifactDigest != "" {
					fmt.Printf("  %s\n", build.OCIArtifactDigest)
				}
				if build.OCISigned {
					fmt.Printf("  Signed: yes (cosign)\n")
				}
				fmt.Println()
				fmt.Printf("Pull with:\n")
				if build.OCIArtifactDigest != "" {
					fmt.Printf("  oras pull %s@%s --output ./\n\n", build.OCIArtifactRef, build.OCIArtifactDigest)
				} else {
					fmt.Printf("  oras pull %s --output ./\n\n", build.OCIArtifactRef)
				}
				if build.OCISigned {
					verifyRef := build.OCIArtifactRef
					if build.OCIArtifactDigest != "" {
						verifyRef = fmt.Sprintf("%s@%s", build.OCIArtifactRef, build.OCIArtifactDigest)
					}
					fmt.Printf("Verify signature:\n")
					fmt.Printf("  cosign verify --key cosign.pub %s\n\n", verifyRef)
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
			fmt.Fprintln(tw, "NAME\tSIZE\tMODIFIED")
			for _, f := range resp.Files {
				fmt.Fprintf(tw, "%s\t%s\t%s\n", f.Name, humanSize(f.Size), f.ModTime)
			}
			tw.Flush()

			fmt.Printf("\nDownload with: bob artifacts %s --download ./out\n", name)
			return nil
		},
	}

	cmd.Flags().StringVarP(&downloadDir, "download", "d", "", "Download all artifacts to this directory")
	cmd.Flags().BoolVar(&skipVerify, "skip-verify", false, "Skip cosign signature verification on download")
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
			_ = body.Close()
			return fmt.Errorf("creating %s: %w", outPath, err)
		}
		_, err = io.Copy(out, body)
		_ = body.Close()
		_ = out.Close()
		if err != nil {
			return fmt.Errorf("writing %s: %w", outPath, err)
		}
	}
	fmt.Printf("Downloaded %d artifacts to %s/\n", len(files), dir)
	return nil
}

func downloadOCIArtifact(ref, digest, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	orasPath, err := exec.LookPath("oras")
	if err != nil {
		return fmt.Errorf("oras CLI not found in PATH — install from https://oras.land/docs/installation")
	}

	pullRef := ref
	if digest != "" {
		pullRef = ref + "@" + digest
	}

	fmt.Printf("Pulling OCI artifact: %s\n", pullRef)
	fmt.Printf("  -> %s/\n", dir)

	cmd := exec.Command(orasPath, "pull", pullRef, "--output", dir, "--insecure")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("oras pull failed: %w", err)
	}

	entries, _ := os.ReadDir(dir)
	fmt.Printf("\nDownloaded %d artifact(s) to %s/\n", len(entries), dir)
	for _, e := range entries {
		fmt.Printf("  %s\n", e.Name())
	}
	return nil
}

func verifyCosignSignature(ref, digest string) error {
	cosignPath, err := exec.LookPath("cosign")
	if err != nil {
		return fmt.Errorf("cosign CLI not found in PATH — install from https://docs.sigstore.dev/cosign/installation")
	}

	verifyRef := ref
	if digest != "" {
		verifyRef = ref + "@" + digest
	}

	pubKeyPath := os.Getenv("COSIGN_PUB_KEY")
	if pubKeyPath == "" {
		pubKeyPath = "cosign.pub"
	}

	if _, err := os.Stat(pubKeyPath); os.IsNotExist(err) {
		return fmt.Errorf("public key not found at %s (set COSIGN_PUB_KEY to override)", pubKeyPath)
	}

	fmt.Printf("Verifying signature for %s...\n", verifyRef)
	args := []string{"verify", "--key", pubKeyPath}
	if os.Getenv("COSIGN_IGNORE_TLOG") != "" {
		args = append(args, "--insecure-ignore-tlog")
	}
	args = append(args, verifyRef)
	cmd := exec.Command(cosignPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cosign verify failed: %w", err)
	}
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
