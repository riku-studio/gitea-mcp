# riku-studio/gitea-mcp

基于官方 [gitea.com/gitea/gitea-mcp](https://gitea.com/gitea/gitea-mcp) **v1.8.0** 修改，原版历史保存在 `upstream` 分支和 `v1.8.0` 标签上。

## 与上游的差异

1. **移除 `create_or_update_file`**：这个工具要求把文件全文写进请求参数里，大批量改动时非常低效。现在改用本地 git 提交。过滤逻辑在 `operation/riku.go`，上游的 `operation/repo` 代码未做改动。
2. **新增本地 git 工具**：直接对 MCP 服务器所在机器上的工作副本执行 `git` 命令，文件内容不经过工具参数传递。

| 工具 | 作用 | 说明 |
|---|---|---|
| `git_status` | 查看分支、ahead/behind、已暂存/未暂存/未跟踪文件 | 只读 |
| `git_diff` | 查看改动 | 默认看工作区；`staged` 看暂存区；`target` 和指定提交比较；可选 `stat` / `name_only` / `paths` |
| `git_log` | 查看提交历史 | `ref`、`max_count`（不超过 500）、`paths`、`stat` |
| `git_fetch` | 下载远端更新 | 不改动工作区 |
| `git_add` | 暂存文件 | 指定 `paths`，或用 `all=true`（`git add -A`） |
| `git_commit` | 提交 | 提交说明经 stdin 传入，多行和 trailer 都能完整保留；可选 `all` 和作者覆盖 |
| `git_push` | 推送 | 不支持 force push；可选 `set_upstream`、`remote_branch`、`tags` |
| `git_pull` | 拉取并合并 | 默认 `--ff-only`；`rebase=true` 改为 rebase；不会产生 merge commit |
| `git_branch` | 列出、新建、删除分支 | 删除用 `-d`，拒绝删除未合并的分支 |
| `git_checkout` | 切换分支（`git switch`） | `create=true` 时新建分支；不丢弃任何文件改动 |

对上游的全部改动：`operation/git/`（新增）、`operation/riku.go`（新增）、`operation/operation.go`（注册）、三个 README 的工具表、`.github/workflows/ci.yml`、`RIKU.md`、`RIKU_VERSION`。

## 安全限制

- **路径白名单**：必须设置环境变量 `GITEA_GIT_ALLOWED_ROOTS`，git 工具只能操作其中目录下的仓库，仓库根目录也必须在白名单内。未设置时，所有 git 工具都会拒绝执行。多个目录在 Windows 上用 `;` 分隔，其他系统用 `:` 分隔。
- **防参数注入**：用户提供的分支、远端、ref 不允许以 `-` 开头；路径统一放在 `--` 之后传入；推送分支名禁止包含 `: + *`，避免绕过限制实现 force push。
- **不会卡在交互上**：执行时设置 `GIT_TERMINAL_PROMPT=0`、`GCM_INTERACTIVE=never`、`GIT_EDITOR=true`。
- **超时**：本地操作 2 分钟，网络操作 10 分钟；输出最多 256KB，超出部分截断。
- **认证**：远端是 `GITEA_HOST` 上的 https 地址时，会自动注入 MCP 的 Gitea token。token 通过 `GIT_CONFIG_*` 环境变量传给 git，不会出现在命令行，也不会出现在返回结果里，不会发给其他主机。其他远端（GitHub、ssh 等）使用本机已有的 git 凭证。

## 配置示例（Claude 桌面 App / Windows）

```json
{
  "mcpServers": {
    "gitea": {
      "command": "C:\\Tools\\gitea-mcp\\gitea-mcp.exe",
      "args": ["-t", "stdio"],
      "env": {
        "GITEA_HOST": "https://git.lan",
        "GITEA_ACCESS_TOKEN": "<token>",
        "GITEA_GIT_ALLOWED_ROOTS": "C:\\Users\\<you>\\src"
      }
    }
  }
}
```

可选环境变量 `GITEA_GIT_BINARY`：指定 git 可执行文件路径，默认从 PATH 中查找 `git`。

仓库远端建议使用 `https://git.lan/<owner>/<repo>.git`。如果远端是 `ssh://git@localhost:2222/...`，只有在 Gitea 服务器本机上才能访问。

```bash
git remote set-url origin https://git.lan/riku-studio/<repo>.git
```

## 构建

`.github/workflows/ci.yml` 会在每次推送到 `main` 时运行：

1. 如果仓库里还没有上游源码，就导入官方 v1.8.0（只执行一次）。
2. 在 Ubuntu 和 Windows 上运行 `go vet` 和 `go test`。
3. 交叉编译 windows/linux/darwin 版本。如果 `RIKU_VERSION` 对应的 Release 还不存在，就自动发布 `v<RIKU_VERSION>`；如果已存在，就更新其中的文件。

要发布新版本，修改 `RIKU_VERSION` 即可。
