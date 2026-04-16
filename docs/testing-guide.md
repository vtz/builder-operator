# Testing bob — Guide for Colleagues

How to test `bob` (the builder) against the shared OpenShift cluster.

---

## 0. What you need

One thing: the **Build API URL**. Ask the person who deployed bob for it.
It looks like: `https://bob-api-bob-system.apps.rosa.my-cluster.abcd.p3.openshiftapps.com`

No cluster login required. No `oc` needed. Just the URL.

---

## 1. Get the `bob` binary

### Option A: Download from the cluster (recommended)

Set the URL you got in step 0, then download:

```bash
BOB_SERVER=https://bob-api-bob-system.apps.rosa.my-cluster.abcd.p3.openshiftapps.com

# Mac (Apple Silicon — M1/M2/M3/M4)
curl -Lo bob $BOB_SERVER/v1/cli/darwin/arm64
chmod +x bob && sudo mv bob /usr/local/bin/

# Mac (Intel)
curl -Lo bob $BOB_SERVER/v1/cli/darwin/amd64
chmod +x bob && sudo mv bob /usr/local/bin/

# Linux (x86_64)
curl -Lo bob $BOB_SERVER/v1/cli/linux/amd64
chmod +x bob && sudo mv bob /usr/local/bin/

# Linux (arm64)
curl -Lo bob $BOB_SERVER/v1/cli/linux/arm64
chmod +x bob && sudo mv bob /usr/local/bin/
```

### Option B: Copy from the dist/ folder

If someone shared the repo with you, pre-built binaries are in `dist/`:

```bash
# Mac Apple Silicon
cp dist/bob-darwin-arm64 /usr/local/bin/bob && chmod +x /usr/local/bin/bob

# Mac Intel
cp dist/bob-darwin-amd64 /usr/local/bin/bob && chmod +x /usr/local/bin/bob

# Linux x86_64
sudo cp dist/bob-linux-amd64 /usr/local/bin/bob && sudo chmod +x /usr/local/bin/bob

# Linux arm64
sudo cp dist/bob-linux-arm64 /usr/local/bin/bob && sudo chmod +x /usr/local/bin/bob
```

### Option C: Build from source

Requires Go 1.22+:

```bash
cd builder-operator
make build-cli
# Binary at: bin/bob
```

### Verify

```bash
bob --help
```

---

## 2. Configure the CLI

Two environment variables. Add to your `~/.zshrc` or `~/.bashrc`:

```bash
export BOB_SERVER=https://bob-api-bob-system.apps.rosa.my-cluster.abcd.p3.openshiftapps.com
export BOB_NAMESPACE=bob-builds
```

Replace the `BOB_SERVER` value with the URL from step 0.

### Verify the connection

```bash
bob list
```

You should see an empty list or existing BuildJobs. If you get a connection
error, check that the server is reachable:

```bash
curl -k $BOB_SERVER/healthz
# Should return: ok
```

---

## 3. Build the body-ecu repo — Step by step

### Step 1: Check what's already on the cluster

```bash
bob list
```

If you see `body-ecu-nucleo` in the list, skip to Step 3.

### Step 2: Create the BuildJob (one-time, needs `oc`)

If the BuildJob doesn't exist yet, someone with cluster access needs to
create it once. This is typically done by the person who deployed bob:

```bash
oc apply -f docs/examples/zephyr-nucleo-cross.yaml
```

This tells bob:
- **What to build:** `https://github.com/vtz/body-ecu` (branch `feature/openbsw-integration`)
- **Toolchain:** Zephyr CI base image
- **Target board:** `nucleo_h755zi_q` (STM32H755 Cortex-M7)
- **Stages:** fetch (west init + update) --> build (west build) --> package (copy artifacts)

### Step 3: Trigger a build

```bash
bob build body-ecu-nucleo
```

This sends a request to the Build API, which triggers a Tekton PipelineRun
on the cluster. The build runs in a container with the Zephyr SDK -- nothing
is installed on your machine.

### Step 4: Watch the build

```bash
bob show body-ecu-nucleo
```

Shows the current phase (Pending --> Running --> Succeeded/Failed) and the
status of each stage.

### Step 5: Stream logs

```bash
bob logs body-ecu-nucleo
```

Streams the build output to your terminal in real time.

### Step 6: Check artifacts

When the build succeeds:

```bash
bob show body-ecu-nucleo
```

Look for the `Artifact URI` in the output. The artifacts are:
- `zephyr.bin` -- raw binary for flashing
- `zephyr.hex` -- Intel HEX format
- `zephyr.elf` -- ELF with debug symbols

---

## 4. Try other examples

These need someone with `oc` access to create the BuildJobs first (or ask
the person who deployed bob to do it). Once created, anyone can trigger
builds with `bob`.

### Zephyr Hello World (native simulation -- no hardware needed)

```bash
bob build zephyr-hello-world
bob logs zephyr-hello-world
```

### OpenBSW POSIX FreeRTOS

```bash
bob build openbsw-posix-freertos
bob logs openbsw-posix-freertos
```

---

## 5. Common commands

| Command | What it does |
|---------|-------------|
| `bob list` | List all BuildJobs and their status |
| `bob build <name>` | Trigger a build |
| `bob show <name>` | Detailed status + stages |
| `bob logs <name>` | Stream build logs |
| `bob delete <name>` | Delete a BuildJob |

All commands accept `--server`, `-n` (namespace) as flags if you prefer not
to use environment variables.

---

## 6. Troubleshooting

### "BOB_SERVER is required"

You haven't set the `BOB_SERVER` environment variable:

```bash
export BOB_SERVER=https://...   # the URL from step 0
bob list
```

### "connection refused" or timeout

Check that the Build API is reachable:

```bash
curl -k $BOB_SERVER/healthz
```

### Build stuck in Pending

Tekton may be scheduling the PipelineRun. Ask someone with cluster access
to check:

```bash
oc get pipelineruns -n bob-builds
```
