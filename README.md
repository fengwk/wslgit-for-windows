# wslgit-for-windows

`wslgit-for-windows` 是一个运行在 Windows 上的 `git.exe` 代理程序。它本身不直接处理 Git 仓库，而是把所有 Git 命令转发到 WSL 中的 `git`，从而让 Windows CLI、PowerShell 和 IDE 与 WSL 使用同一套 Git 实现，避免 Git for Windows 与 WSL Git 交替刷新文件权限和索引元数据。

## What

这个项目要解决的问题是：

- 日常开发主要在 WSL 内进行
- 偶尔需要在 Windows 侧执行 `git`，例如给 IDE、终端或其他工具调用
- Windows Git 与 WSL Git 对同一工作区的文件系统语义不同，容易引发大量索引刷新、权限位变更和 `git status` 异常噪音

`wslgit-for-windows` 的做法是把 Windows 侧的 `git.exe` 变成一个中介层：

```text
Windows CLI / IDE
        |
        v
   wslgit-for-windows (git.exe)
        |
        v
      wsl.exe
        |
        v
   WSL distro 中的 git
```

这样 Windows 侧不再直接操作仓库索引，真正执行命令的始终是 WSL 内的 `git`。

## How it works

程序启动后会执行以下流程：

1. 读取 Windows 当前工作目录
2. 把当前目录从 `C:\...` / `D:\...` 转成 `/mnt/c/...` / `/mnt/d/...`
3. 透传原始 Git 参数
4. 对少量明确是路径值的参数做路径转换
   - `-C <path>`
   - `--git-dir <path>` / `--git-dir=<path>`
   - `--work-tree <path>` / `--work-tree=<path>`
5. 对明确存在的相对 Windows 路径和通配路径，把反斜杠规范化为 `/`
6. 调用 `wsl.exe --cd ... --exec git ...`
7. 若当前 WSL 不支持 `--cd/--exec`，自动回退到 `sh -lc` 模式
8. 原样透传 `stdin`、`stdout`、`stderr` 和退出码

这意味着它不是“重写了一套 Git”，而是一个尽量薄的跨平台桥接层。

## Features

- 使用 WSL 中的 Git 处理真实仓库状态
- 兼容 Windows `cmd.exe`、PowerShell 和大多数 IDE 的 Git 调用方式
- 支持本机盘符路径，例如 `C:\repo`、`D:\work\app`
- 支持旧版 WSL fallback 执行模式
- 支持调试日志，便于排查参数转换和调用链问题

## Build

在 WSL 中构建 Windows 可执行文件：

```bash
cd /home/fengwk/proj/wslgit-for-windows
GOOS=windows GOARCH=amd64 go build -o bin/git.exe ./cmd/git
```

如果目标机器是 Windows on ARM：

```bash
GOOS=windows GOARCH=arm64 go build -o bin/git.exe ./cmd/git
```

## Installation

推荐按下面步骤安装：

1. 在 WSL 中构建 `git.exe`
2. 将生成的 `bin/git.exe` 复制到 Windows 目录，例如：
   - `C:\tools\wslgit\git.exe`
3. 将该目录加入 Windows `PATH`
4. 确保该目录优先级高于 Git for Windows
5. 先验证 CLI 和 IDE 行为正常，再决定是否卸载 Git for Windows

示例：

```powershell
$env:Path = "C:\tools\wslgit;" + $env:Path
git --version
```

## Usage

安装完成后，Windows 侧像平时一样直接使用 `git`：

```powershell
git --version
git status
git -C C:\repo status
git add src\main.go
git commit
git fetch
```

只要当前仓库位于本机盘符路径下，实际执行命令的将是 WSL 内的 Git。

## Environment Variables

- `WSLGIT_DEBUG=1`
  - 开启调试日志
- `WSLGIT_DISTRO=Ubuntu-22.04`
  - 指定目标 WSL 发行版
- `WSLGIT_WSL_PATH=C:\Windows\System32\wsl.exe`
  - 指定 `wsl.exe` 路径
- `WSLGIT_GIT_BINARY=/usr/bin/git`
  - 指定 WSL 内 Git 可执行文件
- `WSLGIT_FORCE_SHELL=1`
  - 强制使用 shell fallback 模式

调试日志默认输出到：

- `%LOCALAPPDATA%\wslgit\logs\YYYYMMDD.log`

## Validation Checklist

建议至少验证以下命令：

```powershell
git --version
git status
git -C C:\repo status
git diff --name-only
git add src\main.go
git commit
git fetch
```

## Known Risks

当前版本的已知边界和风险如下：

1. 只保证本机盘符路径
   - 暂不支持 UNC / 网络共享路径
   - 暂不支持非常规自定义挂载路径

2. 不完整替代 Git for Windows 附带生态
   - 不提供 `ssh.exe`
   - 不提供 `git-lfs.exe`
   - 不提供 Git for Windows 附带的其他辅助程序

3. 依赖 WSL 运行环境
   - 目标 Windows 必须已安装并启用 WSL
   - WSL 发行版中必须有可用的 `git`
   - `wsl.exe` 必须可执行

4. 某些高级场景可能仍需补兼容
   - 部分 GUI 工具可能依赖 Git for Windows 的额外组件
   - 某些非常规 path-like 参数当前未覆盖
   - editor、credential helper、hook 行为最终以 WSL 环境为准

5. 回退模式存在 shell 语义差异
   - 老版本 WSL 下会使用 `sh -lc` fallback
   - 极少数边界参数在 fallback 模式下可能与 `--exec` 模式存在差异

## Design Principles

这个项目遵循以下原则：

- 不重写 Git 业务逻辑
- 不按子命令重建一套命令解释器
- 只处理跨 Windows/WSL 执行必须处理的最小桥接逻辑
- 优先保证主路径稳定，再按日志补边角兼容
