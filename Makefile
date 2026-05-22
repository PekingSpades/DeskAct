VERSION ?=
PLATFORMS ?= all
NON_GITHUB_CI_FLAGS ?=

.PHONY: test non-github-preflight non-github-build non-github-build-all non-github-build-linux non-github-build-vms

test:
	go test ./...

non-github-preflight:
	bash scripts/non-github-ci/preflight.sh --include-vms

non-github-build:
	bash scripts/non-github-ci/build-all.sh --version="$(VERSION)" --platforms="$(PLATFORMS)" $(NON_GITHUB_CI_FLAGS)

non-github-build-all: non-github-build

non-github-build-linux:
	bash scripts/non-github-ci/build-all.sh --version="$(VERSION)" --platforms=linux-amd64 $(NON_GITHUB_CI_FLAGS)

non-github-build-vms:
	bash scripts/non-github-ci/build-all.sh --version="$(VERSION)" --platforms=windows-amd64,darwin-amd64 --enable-dockur-macos --start-windows-vm --start-macos-vm $(NON_GITHUB_CI_FLAGS)
