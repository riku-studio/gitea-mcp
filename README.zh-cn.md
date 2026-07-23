# Gitea MCP 服务器

[English](README.md) | [繁體中文](README.zh-tw.md)

**Gitea MCP 服务器** 将 [Gitea](https://about.gitea.com) 实例接入 [Model Context Protocol](https://modelcontextprotocol.io) 客户端，让仓库、问题、拉取请求等都能在兼容 MCP 的聊天界面中浏览和管理。

[![在 VS Code 中使用 Docker 安装](https://img.shields.io/badge/VS_Code-Install_Server-0098FF?style=flat-square&logo=visualstudiocode&logoColor=white)](https://insiders.vscode.dev/redirect/mcp/install?name=gitea&inputs=[{%22id%22:%22gitea_token%22,%22type%22:%22promptString%22,%22description%22:%22Gitea%20Personal%20Access%20Token%22,%22password%22:true}]&config={%22command%22:%22docker%22,%22args%22:[%22run%22,%22-i%22,%22--rm%22,%22-e%22,%22GITEA_ACCESS_TOKEN%22,%22docker.gitea.com/gitea-mcp-server%22],%22env%22:{%22GITEA_ACCESS_TOKEN%22:%22${input:gitea_token}%22}}) [![在 VS Code Insiders 中使用 Docker 安装](https://img.shields.io/badge/VS_Code_Insiders-Install_Server-24bfa5?style=flat-square&logo=visualstudiocode&logoColor=white)](https://insiders.vscode.dev/redirect/mcp/install?name=gitea&inputs=[{%22id%22:%22gitea_token%22,%22type%22:%22promptString%22,%22description%22:%22Gitea%20Personal%20Access%20Token%22,%22password%22:true}]&config={%22command%22:%22docker%22,%22args%22:[%22run%22,%22-i%22,%22--rm%22,%22-e%22,%22GITEA_ACCESS_TOKEN%22,%22docker.gitea.com/gitea-mcp-server%22],%22env%22:{%22GITEA_ACCESS_TOKEN%22:%22${input:gitea_token}%22}}&quality=insiders)

## 安装

可从 [发布页面](https://gitea.com/gitea/gitea-mcp/releases) 下载二进制文件并放入 `PATH`，或使用 `docker.gitea.com/gitea-mcp-server` 镜像，也可用 `make` 和 Go 1.26 及以上从源码构建到 `$GOPATH/bin`：

```bash
git clone https://gitea.com/gitea/gitea-mcp.git
cd gitea-mcp
make install
```

## 配置

Gitea 主机和访问令牌可通过命令行参数或环境变量提供，命令行参数优先。运行 `gitea-mcp --help` 可查看完整的参数与环境变量列表。日志写入 `$HOME/.gitea-mcp/gitea-mcp.log`，加上 `-d` 可启用调试日志。

### Claude Code

通过 `go run` 运行服务器，需要安装 [Go](https://go.dev)：

```bash
claude mcp add --transport stdio --scope user gitea \
  --env GITEA_ACCESS_TOKEN=token \
  --env GITEA_HOST=https://gitea.com \
  -- go run gitea.com/gitea/gitea-mcp@latest -t stdio
```

### VS Code

可使用本 README 顶部的安装按钮，或将下面的内容加入用户设置 (JSON)，按 `Ctrl + Shift + P` 并输入 `Preferences: Open User Settings (JSON)` 即可打开。也可放在工作区的 `.vscode/mcp.json` 中，此时无需 `mcp` 键。

```json
{
  "mcp": {
    "inputs": [
      {
        "type": "promptString",
        "id": "gitea_token",
        "description": "Gitea 个人访问令牌",
        "password": true
      }
    ],
    "servers": {
      "gitea-mcp": {
        "command": "docker",
        "args": ["run", "-i", "--rm", "-e", "GITEA_ACCESS_TOKEN", "docker.gitea.com/gitea-mcp-server"],
        "env": {
          "GITEA_ACCESS_TOKEN": "${input:gitea_token}"
        }
      }
    }
  }
}
```

### OpenCode

将下面的内容加入 [OpenCode](https://opencode.ai) 配置的顶层 `mcp` 对象：

```json
    "gitea-mcp": {
      "enabled": true,
      "type": "local",
      "command": [
        "gitea-mcp",
        "-t", "stdio",
        "-H", "https://gitea.com",
        "-T", "<your personal access token>"
      ]
    }
```

### Mistral Vibe

将下面的内容加入 `~/.vibe/config.toml`：

```toml
[[mcp_servers]]
name = "gitea"
transport = "stdio"
command = "docker"
args = ["run", "--rm", "-i", "-e", "GITEA_ACCESS_TOKEN", "-e", "GITEA_HOST", "docker.gitea.com/gitea-mcp-server"]

[mcp_servers.env]
GITEA_ACCESS_TOKEN = "TOKEN"
GITEA_HOST = "https://gitea.com"
```

### 其他客户端

Cursor 等客户端可使用 stdio 命令：

```json
{
  "mcpServers": {
    "gitea": {
      "command": "gitea-mcp",
      "args": ["-t", "stdio", "--host", "https://gitea.com"],
      "env": {
        "GITEA_ACCESS_TOKEN": "<your personal access token>"
      }
    }
  }
}
```

或使用 http 端点，对应以 `gitea-mcp -t http --port 8080` 启动的服务器：

```json
{
  "mcpServers": {
    "gitea": {
      "url": "http://localhost:8080/mcp",
      "headers": {
        "Authorization": "Bearer <your personal access token>"
      }
    }
  }
}
```

配置完成后，可在聊天框输入 `列出我所有的仓库` 试试。

## 可用工具

| 工具                         | 范围     | 访问 | 描述                                                                 |
| :--------------------------- | :------- | :--- | :------------------------------------------------------------------- |
| get_gitea_mcp_server_version | 版本     | 读取 | 获取 Gitea MCP 服务器版本                                            |
| get_me                       | 用户     | 读取 | 获取当前已认证用户                                                   |
| get_user_orgs                | 用户     | 读取 | 列出当前用户的组织                                                   |
| search_users                 | 搜索     | 读取 | 搜索用户                                                             |
| search_org_teams             | 搜索     | 读取 | 搜索组织中的团队                                                     |
| search_repos                 | 搜索     | 读取 | 搜索仓库                                                             |
| search_issues                | 搜索     | 读取 | 跨仓库搜索问题和拉取请求                                             |
| notification_read            | 通知     | 读取 | 读取通知：列出（可限定仓库）或按 ID 获取会话                         |
| notification_write           | 通知     | 写入 | 将某条或全部通知标记为已读                                           |
| label_read                   | 标签     | 读取 | 读取仓库或组织标签                                                   |
| label_write                  | 标签     | 写入 | 写入标签（仓库或组织）：创建、编辑、删除                             |
| milestone_read               | 里程碑   | 读取 | 读取里程碑：获取单个或列出                                           |
| milestone_write              | 里程碑   | 写入 | 写入里程碑：创建、更新、删除                                         |
| wiki_read                    | Wiki     | 读取 | 读取 Wiki：列出页面、获取内容、修订历史                              |
| wiki_write                   | Wiki     | 写入 | 写入 Wiki 页面：创建、更新、删除                                     |
| timetracking_read            | 时间跟踪 | 读取 | 读取时间跟踪：问题/仓库耗时、活动计时器、我的跟踪记录                |
| timetracking_write           | 时间跟踪 | 写入 | 写入时间跟踪：计时器和记录条目                                       |
| package_read                 | 软件包   | 读取 | 读取软件包注册表：列出软件包、列出版本或获取某个版本                 |
| package_write                | 软件包   | 写入 | 删除软件包版本（不可恢复）                                           |
| list_issues                  | 问题     | 读取 | 列出仓库问题                                                         |
| issue_read                   | 问题     | 读取 | 读取问题：详情、评论或标签                                           |
| issue_write                  | 问题     | 写入 | 写入问题：创建、更新、管理评论和标签                                 |
| list_pull_requests           | 拉取请求 | 读取 | 列出仓库拉取请求                                                     |
| pull_request_read            | 拉取请求 | 读取 | 读取拉取请求：详情、差异、变更文件、头部提交状态、审查               |
| pull_request_write           | 拉取请求 | 写入 | 写入拉取请求：创建、更新、关闭、重新打开、合并、更新分支、管理审查者 |
| pull_request_review_write    | 拉取请求 | 写入 | 写入 PR 审查：创建、提交、删除、驳回                                 |
| actions_config_read          | Actions  | 读取 | 读取 Actions 密钥和变量                                              |
| actions_config_write         | Actions  | 写入 | 写入 Actions 密钥和变量：更新插入、创建、更新、删除                  |
| actions_run_read             | Actions  | 读取 | 读取 Actions 工作流、运行、作业、日志和构件                          |
| actions_run_write            | Actions  | 写入 | 写入 Actions 运行：触发、取消、重新运行                              |
| create_repo                  | 仓库     | 写入 | 创建新仓库                                                           |
| fork_repo                    | 仓库     | 写入 | 复刻仓库                                                             |
| list_my_repos                | 仓库     | 读取 | 列出当前用户拥有的仓库                                               |
| list_org_repos               | 仓库     | 读取 | 列出组织中的仓库                                                     |
| get_repository_tree          | 仓库     | 读取 | 获取仓库文件树                                                       |
| get_file_contents            | 文件     | 读取 | 获取文件内容和元数据                                                 |
| get_dir_contents             | 文件     | 读取 | 获取目录中的条目                                                     |
| create_or_update_file        | 文件     | 写入 | 创建或更新文件（提供 sha 以更新现有文件）                            |
| delete_file                  | 文件     | 写入 | 删除文件                                                             |
| create_branch                | 分支     | 写入 | 创建新分支                                                           |
| delete_branch                | 分支     | 写入 | 删除分支                                                             |
| list_branches                | 分支     | 读取 | 列出仓库分支                                                         |
| create_tag                   | Git 标签 | 写入 | 创建标签                                                             |
| delete_tag                   | Git 标签 | 写入 | 删除标签                                                             |
| get_tag                      | Git 标签 | 读取 | 获取标签详情                                                         |
| list_tags                    | Git 标签 | 读取 | 列出仓库标签                                                         |
| list_commits                 | 提交     | 读取 | 列出仓库提交                                                         |
| get_commit                   | 提交     | 读取 | 获取提交详情                                                         |
| create_release               | 版本发布 | 写入 | 创建版本发布                                                         |
| delete_release               | 版本发布 | 写入 | 删除版本发布                                                         |
| get_release                  | 版本发布 | 读取 | 按 ID 获取版本发布                                                   |
| get_latest_release           | 版本发布 | 读取 | 获取最新版本发布                                                     |
| list_releases                | 版本发布 | 读取 | 列出仓库版本发布                                                     |

> **说明：** 部分工具是聚合的、基于操作的工具，单个工具通过 `method` 参数暴露多个操作。当服务器以只读模式运行时（`-r` / `GITEA_READONLY`），访问为「写入」的工具会被隐藏；可通过 `-O` / `--tools`（`GITEA_TOOLS`）过滤对外暴露的工具集合。

许多工具支持 `page` 和 `per_page` 分页参数。最大有效页面大小由 Gitea 服务器的 `[api].MAX_RESPONSE_ITEMS` 设置决定（默认 **50**），超出的值会被静默截断。
