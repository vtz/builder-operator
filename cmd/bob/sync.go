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
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	originalSourceAnnotation = "builder.sdv.cloud.redhat.com/original-source"
	sourceTypeLabel          = "builder.sdv.cloud.redhat.com/source-type"
	runAtAnnotation          = "builder.sdv.cloud.redhat.com/run-at"
)

func newSyncCmd() *cobra.Command {
	var pvcName string
	var namespace string
	var pvcPath string

	cmd := &cobra.Command{
		Use:   "sync <local-dir>",
		Short: "Sync local source directory to a PVC on the cluster",
		Long: `Upload your local working directory to a PersistentVolumeClaim on the cluster.
Typically you don't need this directly — use "bob build --local" instead.

Examples:
  bob sync .                          # sync current dir to default PVC
  bob sync ~/my-project --pvc mypvc   # sync to a named PVC`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ns := firstNonEmpty(namespace, bobNamespace, os.Getenv("BOB_NAMESPACE"), "bob-builds")
			kubecli := detectKubeClient()
			if kubecli == "" {
				return fmt.Errorf("no Kubernetes CLI found (install oc or kubectl)")
			}
			return runSync(args[0], pvcName, ns, pvcPath, kubecli)
		},
	}

	cmd.Flags().StringVar(&pvcName, "pvc", "source-code", "Name of the target PVC")
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Target namespace (defaults to BOB_NAMESPACE or bob-builds)")
	cmd.Flags().StringVar(&pvcPath, "path", "/", "Target path within the PVC")
	return cmd
}

func detectKubeClient() string {
	if p, err := exec.LookPath("oc"); err == nil {
		return p
	}
	if p, err := exec.LookPath("kubectl"); err == nil {
		return p
	}
	return ""
}

func ensurePVC(kubecli, namespace, pvcName string) error {
	checkCmd := exec.Command(kubecli, "get", "pvc", pvcName, "-n", namespace, "-o", "name")
	if err := checkCmd.Run(); err == nil {
		return nil
	}

	fmt.Print("Creating PVC... ")
	pvcManifest := fmt.Sprintf(`{
  "apiVersion": "v1",
  "kind": "PersistentVolumeClaim",
  "metadata": {"name": %q, "namespace": %q},
  "spec": {
    "accessModes": ["ReadWriteOnce"],
    "resources": {"requests": {"storage": "5Gi"}}
  }
}`, pvcName, namespace)
	createCmd := exec.Command(kubecli, "apply", "-n", namespace, "-f", "-")
	createCmd.Stdin = strings.NewReader(pvcManifest)
	createCmd.Stderr = os.Stderr
	if err := createCmd.Run(); err != nil {
		return fmt.Errorf("creating PVC: %w", err)
	}
	fmt.Println("done")
	return nil
}

func switchToPVCAndTrigger(kubecli, namespace, bjName, pvcName, pvcPath string) error {
	fmt.Printf("\nSwitching BuildJob %s to local source...\n", bjName)

	// Save the original source spec as JSON so we can restore later
	out, err := exec.Command(kubecli, "get", "buildjob", bjName, "-n", namespace,
		"-o", "json").CombinedOutput()
	if err != nil {
		return fmt.Errorf("reading BuildJob: %w", err)
	}
	var bjObj map[string]interface{}
	if err := json.Unmarshal(out, &bjObj); err != nil {
		return fmt.Errorf("parsing BuildJob JSON: %w", err)
	}
	spec, _ := bjObj["spec"].(map[string]interface{})
	sourceSpec, _ := spec["source"].(map[string]interface{})
	sourceJSON, err := json.Marshal(sourceSpec)
	if err != nil {
		return fmt.Errorf("serializing source spec: %w", err)
	}
	originalSource := string(sourceJSON)

	// Check if already has an original-source annotation (don't overwrite)
	existing, _ := exec.Command(kubecli, "get", "buildjob", bjName, "-n", namespace,
		"-o", `jsonpath={.metadata.annotations.builder\.sdv\.cloud\.redhat\.com/original-source}`).CombinedOutput()
	if len(existing) == 0 || string(existing) == "" {
		if err := exec.Command(kubecli, "annotate", "buildjob", bjName, "-n", namespace,
			originalSourceAnnotation+"="+originalSource, "--overwrite").Run(); err != nil {
			return fmt.Errorf("saving original source: %w", err)
		}
	}

	// Build the PVC source patch
	pvcSource := map[string]interface{}{
		"type": "pvc",
		"pvc": map[string]interface{}{
			"claimName": pvcName,
			"path":      pvcPath,
		},
	}
	// Clear git field explicitly
	patch := map[string]interface{}{
		"spec": map[string]interface{}{
			"source": pvcSource,
		},
	}
	patchJSON, _ := json.Marshal(patch)

	fmt.Print("Patching source to PVC... ")
	if err := exec.Command(kubecli, "patch", "buildjob", bjName, "-n", namespace,
		"--type=merge", "-p", string(patchJSON)).Run(); err != nil {
		return fmt.Errorf("patching BuildJob: %w", err)
	}
	fmt.Println("done")

	// Label as local build
	fmt.Print("Labeling as local build... ")
	if err := exec.Command(kubecli, "label", "buildjob", bjName, "-n", namespace,
		sourceTypeLabel+"=local", "--overwrite").Run(); err != nil {
		return fmt.Errorf("labeling BuildJob: %w", err)
	}
	fmt.Println("done")

	// Trigger the build
	fmt.Print("Triggering build... ")
	ts := time.Now().UTC().Format(time.RFC3339)
	if err := exec.Command(kubecli, "annotate", "buildjob", bjName, "-n", namespace,
		runAtAnnotation+"="+ts, "--overwrite").Run(); err != nil {
		return fmt.Errorf("triggering build: %w", err)
	}
	fmt.Println("done")

	fmt.Printf("\nLocal build started for %s\n", bjName)
	fmt.Printf("  Watch progress:  bob list\n")
	fmt.Printf("  Stream logs:     bob logs %s\n", bjName)
	return nil
}

func restoreGitSource(kubecli, namespace, bjName string) error {
	fmt.Printf("Restoring BuildJob %s to git source...\n", bjName)

	out, err := exec.Command(kubecli, "get", "buildjob", bjName, "-n", namespace,
		"-o", `jsonpath={.metadata.annotations.builder\.sdv\.cloud\.redhat\.com/original-source}`).CombinedOutput()
	if err != nil || len(out) == 0 || string(out) == "" {
		return fmt.Errorf("no original source found; BuildJob may not have been switched to local")
	}

	patch := fmt.Sprintf(`{"spec":{"source":%s}}`, string(out))

	fmt.Print("Restoring source spec... ")
	if err := exec.Command(kubecli, "patch", "buildjob", bjName, "-n", namespace,
		"--type=merge", "-p", patch).Run(); err != nil {
		return fmt.Errorf("restoring source: %w", err)
	}
	fmt.Println("done")

	// Remove the local label
	_ = exec.Command(kubecli, "label", "buildjob", bjName, "-n", namespace,
		sourceTypeLabel+"-").Run()

	// Remove the saved annotation
	_ = exec.Command(kubecli, "annotate", "buildjob", bjName, "-n", namespace,
		originalSourceAnnotation+"-").Run()

	fmt.Printf("BuildJob %s restored to git source.\n", bjName)
	return nil
}

var syncExcludes = []string{".git", "node_modules", "__pycache__", ".tox", ".venv", "build", ".bob-output"}

func runSync(localDir, pvcName, namespace, pvcPath, kubecli string) error {
	absDir, err := filepath.Abs(localDir)
	if err != nil {
		return fmt.Errorf("resolving source directory: %w", err)
	}
	if info, err := os.Stat(absDir); err != nil || !info.IsDir() {
		return fmt.Errorf("source path is not a directory: %s", absDir)
	}

	podName := fmt.Sprintf("bob-sync-%s", pvcName)
	if pvcPath == "" {
		pvcPath = "/"
	}

	cli := filepath.Base(kubecli)
	useRsync := cli == "oc"

	fmt.Printf("Syncing local source to cluster PVC\n")
	fmt.Printf("  Source:     %s\n", absDir)
	fmt.Printf("  PVC:        %s/%s\n", namespace, pvcName)
	fmt.Printf("  Path:       %s\n", pvcPath)
	fmt.Printf("  CLI:        %s\n", cli)
	if useRsync {
		fmt.Printf("  Strategy:   rsync (incremental)\n\n")
	} else {
		fmt.Printf("  Strategy:   tar (full upload)\n\n")
	}

	if err := ensurePVC(kubecli, namespace, pvcName); err != nil {
		return err
	}

	syncImage := "busybox:latest"
	syncCmd := "echo ready && sleep 3600"
	if useRsync {
		syncImage = "alpine:latest"
		syncCmd = "apk add --no-cache rsync >/dev/null 2>&1 && echo ready && sleep 3600"
	}

	if err := ensureSyncPod(kubecli, namespace, podName, syncImage, syncCmd, pvcName); err != nil {
		return err
	}

	destPath := filepath.Join("/mnt/pvc", pvcPath)

	if useRsync {
		return rsyncUpload(kubecli, podName, namespace, absDir, destPath)
	}
	return tarUpload(kubecli, podName, namespace, absDir, destPath)
}

func ensureSyncPod(kubecli, namespace, podName, image, cmd, pvcName string) error {
	// Check if pod already exists and is Ready
	phase, _ := exec.Command(kubecli, "get", "pod", podName, "-n", namespace,
		"-o", "jsonpath={.status.phase}").CombinedOutput()

	switch string(phase) {
	case "Running":
		fmt.Println("Reusing existing sync pod")
		return nil
	case "":
		// Pod doesn't exist, create it
	default:
		// Pod exists but not running (Pending, Terminating, Failed, etc.) -- delete and recreate
		fmt.Print("Removing stale sync pod... ")
		delCmd := exec.Command(kubecli, "delete", "pod", podName, "-n", namespace,
			"--ignore-not-found", "--grace-period=0", "--force")
		delCmd.Stderr = os.Stderr
		_ = delCmd.Run()
		fmt.Println("done")
	}

	podManifest := fmt.Sprintf(`{
  "apiVersion": "v1",
  "kind": "Pod",
  "metadata": {"name": %q, "namespace": %q, "labels": {"app.kubernetes.io/managed-by": "bob"}},
  "spec": {
    "restartPolicy": "Never",
    "containers": [{
      "name": "sync",
      "image": %q,
      "command": ["sh", "-c", %q],
      "volumeMounts": [{"name": "source", "mountPath": "/mnt/pvc"}]
    }],
    "volumes": [{"name": "source", "persistentVolumeClaim": {"claimName": %q}}]
  }
}`, podName, namespace, image, cmd, pvcName)

	fmt.Print("Creating sync pod... ")
	applyCmd := exec.Command(kubecli, "create", "-n", namespace, "-f", "-")
	applyCmd.Stdin = strings.NewReader(podManifest)
	applyCmd.Stderr = os.Stderr
	if err := applyCmd.Run(); err != nil {
		return fmt.Errorf("creating sync pod: %w", err)
	}
	fmt.Println("done")

	fmt.Print("Waiting for pod to be ready... ")
	waitCmd := exec.Command(kubecli, "wait", "--for=condition=Ready", "pod/"+podName, "-n", namespace, "--timeout=120s")
	waitCmd.Stderr = os.Stderr
	if err := waitCmd.Run(); err != nil {
		return fmt.Errorf("waiting for sync pod: %w", err)
	}
	fmt.Println("ready")
	return nil
}

func rsyncUpload(kubecli, podName, namespace, srcDir, destPath string) error {
	fmt.Print("Ensuring target directory... ")
	mkdirCmd := exec.Command(kubecli, "exec", podName, "-n", namespace, "--",
		"mkdir", "-p", destPath)
	mkdirCmd.Stderr = os.Stderr
	if err := mkdirCmd.Run(); err != nil {
		return fmt.Errorf("creating target directory: %w", err)
	}
	fmt.Println("done")

	// Trailing slash on source tells oc rsync to sync contents, not the directory itself
	src := srcDir
	if !strings.HasSuffix(src, "/") {
		src += "/"
	}
	dest := fmt.Sprintf("%s:%s", podName, destPath)

	args := []string{"rsync", "--delete", "--no-perms", "-n", namespace}
	for _, ex := range syncExcludes {
		args = append(args, fmt.Sprintf("--exclude=%s", ex))
	}
	args = append(args, src, dest)

	fmt.Print("Syncing files (rsync)... ")
	rsyncCmd := exec.Command(kubecli, args...)
	rsyncCmd.Stderr = os.Stderr
	if err := rsyncCmd.Run(); err != nil {
		return fmt.Errorf("rsync upload: %w", err)
	}
	fmt.Println("done")
	return nil
}

func tarUpload(kubecli, podName, namespace, srcDir, destPath string) error {
	fmt.Print("Clearing target path... ")
	clearCmd := exec.Command(kubecli, "exec", podName, "-n", namespace, "--", "sh", "-c",
		fmt.Sprintf("rm -rf %s/* %s/.[!.]* 2>/dev/null; mkdir -p %s", destPath, destPath, destPath))
	clearCmd.Stderr = os.Stderr
	if err := clearCmd.Run(); err != nil {
		return fmt.Errorf("clearing target path: %w", err)
	}
	fmt.Println("done")

	fmt.Print("Uploading files (tar)... ")
	cpCmd := exec.Command(kubecli, "exec", "-i", podName, "-n", namespace, "--", "tar", "xzf", "-", "-C", destPath)
	cpCmd.Stderr = os.Stderr

	stdin, err := cpCmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("creating stdin pipe: %w", err)
	}

	if err := cpCmd.Start(); err != nil {
		return fmt.Errorf("starting tar extract: %w", err)
	}

	count, err := tarDirectory(srcDir, stdin)
	stdin.Close()
	if err != nil {
		return fmt.Errorf("creating tar archive: %w", err)
	}

	if err := cpCmd.Wait(); err != nil {
		return fmt.Errorf("uploading files: %w", err)
	}
	fmt.Printf("%d files uploaded\n", count)
	return nil
}

func tarDirectory(dir string, w io.Writer) (int, error) {
	gw := gzip.NewWriter(w)
	tw := tar.NewWriter(gw)
	count := 0

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		if shouldSkipPath(rel) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = rel

		if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			header.Linkname = link
		}

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if !info.Mode().IsRegular() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(tw, f)
		f.Close()
		if copyErr != nil {
			return copyErr
		}
		count++
		return nil
	})
	if err != nil {
		return count, err
	}

	if err := tw.Close(); err != nil {
		return count, err
	}
	return count, gw.Close()
}

func shouldSkipPath(rel string) bool {
	base := filepath.Base(rel)
	for _, s := range syncExcludes {
		if base == s {
			return true
		}
	}
	return false
}
