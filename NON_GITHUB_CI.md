# DeskAct Non-GitHub x86 Builds

This build path creates DeskAct x86_64/amd64 artifacts outside GitHub Actions.
It keeps artifacts local and does not upload them to any external service.

## Targets

- Go tester:
  - `go/linux-amd64/deskact-tester`
  - `go/windows-amd64/deskact-tester.exe`
  - `go/darwin-amd64/deskact-tester`
- Go examples:
  - `capture`, `display`, `keyboard`, `mouse`, `window`, `apps`
  - emitted under `go/<platform>/examples/`
- Electron display:
  - `electron/linux-amd64/electron-display-linux-amd64.zip`
  - `electron/windows-amd64/electron-display-windows-amd64.zip`
  - `electron/darwin-amd64/electron-display-darwin-amd64.zip`

The default output directory is:

```bash
artifacts/non-github-ci/<version>/
```

Each run also writes `SHA256SUMS` and `manifest.json`.

## Host Setup

Linux host requirements:

- Go
- Node.js and npm
- `zip` or `7z`
- Linux CGO dependencies: `libx11-dev`, `libxtst-dev`, `libxinerama-dev`, `libxrandr-dev`, `libpng-dev`, `libxcomposite-dev`, `libxrender-dev`, `libxfixes-dev`
- Optional Windows/macOS VM builds: an x86_64 machine with nested virtualization enabled, Docker with Compose plugin, `/dev/kvm`, and `/dev/net/tun`

Run:

```bash
make non-github-preflight
```

The scripts default to the public npm registry. Override `NPM_CONFIG_REGISTRY`
before running the scripts if the builder needs a package mirror.

## Build Commands

Build Linux artifacts on the current host:

```bash
make non-github-build-linux VERSION=v0.1.0
```

Stage all platforms, build the native platform, and write checksums:

```bash
bash scripts/non-github-ci/build-all.sh \
  --version=v0.1.0 \
  --platforms=all \
  --enable-dockur-macos
```

Build Go-only artifacts:

```bash
bash scripts/non-github-ci/build-all.sh \
  --version=v0.1.0 \
  --platforms=linux-amd64 \
  --skip-electron
```

## Windows Worker

Windows CGO artifacts must be built on Windows. Stage the worker command and
optionally start the VM:

```bash
bash scripts/non-github-ci/build-all.sh \
  --version=v0.1.0 \
  --platforms=windows-amd64 \
  --start-windows-vm
```

Inside the Windows shared repo checkout, run:

```powershell
.\scripts\non-github-ci\bootstrap-windows.ps1
.\artifacts\non-github-ci\v0.1.0\vm\run-windows-build.ps1
```

The bootstrap script does not require `winget`; it can download and install Go,
Node.js, and MSYS2 directly. `git` is optional and only used to fill
`manifest.json`'s commit field when present.

For a fresh Compose Windows worker, the `scripts/non-github-ci/oem/windows`
folder is mounted as the VM OEM folder. After first login, the worker attempts
to bootstrap tools and run the newest staged `run-windows-build.ps1`
automatically. Existing Windows worker disks may require the manual commands
above because OEM scripts run during initial setup.

Windows artifacts are written back under:

```text
artifacts/non-github-ci/v0.1.0/
```

## macOS Worker

macOS artifacts must be built on an authorized macOS x86_64 builder. Stage the
worker command and optionally start the VM:

```bash
bash scripts/non-github-ci/build-all.sh \
  --version=v0.1.0 \
  --platforms=darwin-amd64 \
  --enable-dockur-macos \
  --start-macos-vm
```

If using the Compose macOS worker, mount the shared folder after installation:

```bash
sudo -S mount_9p shared
```

Then run from the shared repo checkout:

```bash
bash artifacts/non-github-ci/v0.1.0/vm/run-macos-build.sh
```

The staged macOS script loads `bootstrap-macos.sh` in the same shell before it
builds so user-local Go/Node installs are available on `PATH`.

macOS artifacts are written back under:

```text
artifacts/non-github-ci/v0.1.0/
```

## Notes

- The GitHub workflows remain available, but this path does not depend on
  GitHub Actions.
- The build manifest records version, commit, time, relative artifact paths,
  and checksums only.
- No signing, notarization, installer packaging, or artifact upload is handled
  by this path.
