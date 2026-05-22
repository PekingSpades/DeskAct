## DeskAct Non-GitHub CI x86 Builds

- DeskAct 的非 GitHub CI 构建入口在 `scripts/non-github-ci/`，详细操作说明在 `NON_GITHUB_CI.md`。
- 该构建路径面向 x86_64/amd64 构建机；如果要在本机启动 Windows/macOS VM worker，需要机器支持嵌套虚拟化，并可用 `/dev/kvm`、`/dev/net/tun`、Docker Compose。
- 构建前优先运行：
  - `make non-github-preflight`
- Linux 产物可在本机直接构建：
  - `make non-github-build-linux`
- 全平台编排使用：
  - `bash scripts/non-github-ci/build-all.sh --version=<version> --platforms=all --enable-dockur-macos`
- Windows 产物必须在 Windows worker 内完成；先 staging，再在 Windows 共享目录中运行生成的 `artifacts/non-github-ci/<version>/vm/run-windows-build.ps1`。
- fresh Windows Compose worker 会通过 `scripts/non-github-ci/oem/windows/` 在首次登录后尝试自动运行最新 staged Windows build；已存在的 Windows worker disk 可能需要手动运行 staged 脚本。
- macOS 产物必须在已授权的 macOS x86_64 builder 内完成；dockur macOS 路径必须显式传入 `--enable-dockur-macos`，VM 内挂载 shared 后运行生成的 `artifacts/non-github-ci/<version>/vm/run-macos-build.sh`。
- 产物默认输出到 `artifacts/non-github-ci/<version>/`，该目录不应提交。构建脚本和文档中不要硬编码构建机厂商、资产名称、公网或内网地址；需要跨 VM 访问宿主时使用共享目录约定。
