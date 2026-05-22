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

## Non-Preemptive Window Ops + Per-Window Screenshot

- 入口 example：`examples/windowops/`（人机交互菜单）与 `examples/windowops/selftest/`（无人值守 JSON 场景跑通器）。
- 自测器 stdin 接 JSON 场景、stdout 输出 JSON 报告 + PNG 落到 `outDir`，方便在 VM 内通过 RDP/VNC/9p 共享盘跑、把结果拷回宿主对比。
- 验证脚本：`scripts/non-github-ci/verify/`
  - `run-linux-selftest.sh`（本机 X 会话）
  - `run-windows-vm-selftest.sh`（dockur Windows VM；Z:\ 共享）
  - `run-macos-vm-selftest.sh`（dockur macOS VM；/Volumes/shared 9p）
  - `novnc-snapshot.sh`（用 noVNC/VNC 抓帧做交叉验证）
  - `scenarios/{xterm-basic,notepad,textedit}.json` 提供各平台默认场景。
- 已知限制（写进 godoc 与 PR description）：
  - Wayland 与未知 session：所有 per-window 入口返回 `capture.ErrUnsupported`。
  - macOS 窗口操作首次会请求 Accessibility 授权；未授权返回 `capture.ErrPermissionDenied`。
  - Windows WGC C++ wrapper 当前为运行时 stub；构建期已通过 worker env 修通，运行时自动回退到 `PrintWindow(PW_RENDERFULLCONTENT)`。原因记录在 `screenshot/windows_wgc.cpp` 顶部注释（mingw-w64 13.x 的 WinRT ABI 头与 `__mingw_uuidof<>` 模板冲突，需要走 C ABI/COBJMACROS 路径重写）。
  - X11 `XSendEvent` 的 `send_event` 标记会被部分 GTK/Qt 应用过滤。
