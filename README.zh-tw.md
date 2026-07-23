# Gitea MCP 伺服器

[English](README.md) | [简体中文](README.zh-cn.md)

**Gitea MCP 伺服器** 將 [Gitea](https://about.gitea.com) 實例接入 [Model Context Protocol](https://modelcontextprotocol.io) 客戶端，讓倉庫、問題、拉取請求等都能在相容 MCP 的聊天介面中瀏覽與管理。

[![在 VS Code 中使用 Docker 安裝](https://img.shields.io/badge/VS_Code-Install_Server-0098FF?style=flat-square&logo=visualstudiocode&logoColor=white)](https://insiders.vscode.dev/redirect/mcp/install?name=gitea&inputs=[{%22id%22:%22gitea_token%22,%22type%22:%22promptString%22,%22description%22:%22Gitea%20Personal%20Access%20Token%22,%22password%22:true}]&config={%22command%22:%22docker%22,%22args%22:[%22run%22,%22-i%22,%22--rm%22,%22-e%22,%22GITEA_ACCESS_TOKEN%22,%22docker.gitea.com/gitea-mcp-server%22],%22env%22:{%22GITEA_ACCESS_TOKEN%22:%22${input:gitea_token}%22}}) [![在 VS Code Insiders 中使用 Docker 安裝](https://img.shields.io/badge/VS_Code_Insiders-Install_Server-24bfa5?style=flat-square&logo=visualstudiocode&logoColor=white)](https://insiders.vscode.dev/redirect/mcp/install?name=gitea&inputs=[{%22id%22:%22gitea_token%22,%22type%22:%22promptString%22,%22description%22:%22Gitea%20Personal%20Access%20Token%22,%22password%22:true}]&config={%22command%22:%22docker%22,%22args%22:[%22run%22,%22-i%22,%22--rm%22,%22-e%22,%22GITEA_ACCESS_TOKEN%22,%22docker.gitea.com/gitea-mcp-server%22],%22env%22:{%22GITEA_ACCESS_TOKEN%22:%22${input:gitea_token}%22}}&quality=insiders)

## 安裝

可從 [發布頁面](https://gitea.com/gitea/gitea-mcp/releases) 下載二進位檔並放入 `PATH`，或使用 `docker.gitea.com/gitea-mcp-server` 映像檔，也可用 `make` 與 Go 1.26 以上從原始碼建置到 `$GOPATH/bin`：

```bash
git clone https://gitea.com/gitea/gitea-mcp.git
cd gitea-mcp
make install
```

## 設定

Gitea 主機與存取令牌可透過命令列參數或環境變數提供，命令列參數優先。執行 `gitea-mcp --help` 可查看完整的參數與環境變數列表。日誌寫入 `$HOME/.gitea-mcp/gitea-mcp.log`，加上 `-d` 可啟用除錯日誌。

### Claude Code

透過 `go run` 執行伺服器，需要安裝 [Go](https://go.dev)：

```bash
claude mcp add --transport stdio --scope user gitea \
  --env GITEA_ACCESS_TOKEN=token \
  --env GITEA_HOST=https://gitea.com \
  -- go run gitea.com/gitea/gitea-mcp@latest -t stdio
```

### VS Code

可使用本 README 頂部的安裝按鈕，或將下面的內容加入使用者設定 (JSON)，按 `Ctrl + Shift + P` 並輸入 `Preferences: Open User Settings (JSON)` 即可開啟。也可放在工作區的 `.vscode/mcp.json` 中，此時不需要 `mcp` 鍵。

```json
{
  "mcp": {
    "inputs": [
      {
        "type": "promptString",
        "id": "gitea_token",
        "description": "Gitea 個人存取令牌",
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

將下面的內容加入 [OpenCode](https://opencode.ai) 設定的頂層 `mcp` 物件：

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

將下面的內容加入 `~/.vibe/config.toml`：

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

### 其他客戶端

Cursor 等客戶端可使用 stdio 命令：

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

或使用 http 端點，對應以 `gitea-mcp -t http --port 8080` 啟動的伺服器：

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

設定完成後，可在聊天框輸入 `列出我所有的倉庫` 試試。

## 可用工具

| 工具                         | 範圍     | 存取 | 描述                                                                 |
| :--------------------------- | :------- | :--- | :------------------------------------------------------------------- |
| get_gitea_mcp_server_version | 版本     | 讀取 | 取得 Gitea MCP 伺服器版本                                            |
| get_me                       | 用戶     | 讀取 | 取得目前已認證用戶                                                   |
| get_user_orgs                | 用戶     | 讀取 | 列出目前用戶的組織                                                   |
| search_users                 | 搜尋     | 讀取 | 搜尋用戶                                                             |
| search_org_teams             | 搜尋     | 讀取 | 搜尋組織中的團隊                                                     |
| search_repos                 | 搜尋     | 讀取 | 搜尋倉庫                                                             |
| search_issues                | 搜尋     | 讀取 | 跨倉庫搜尋問題和拉取請求                                             |
| notification_read            | 通知     | 讀取 | 讀取通知：列出（可限定倉庫）或依 ID 取得會話                         |
| notification_write           | 通知     | 寫入 | 將某條或全部通知標記為已讀                                           |
| label_read                   | 標籤     | 讀取 | 讀取倉庫或組織標籤                                                   |
| label_write                  | 標籤     | 寫入 | 寫入標籤（倉庫或組織）：創建、編輯、刪除                             |
| milestone_read               | 里程碑   | 讀取 | 讀取里程碑：取得單個或列出                                           |
| milestone_write              | 里程碑   | 寫入 | 寫入里程碑：創建、更新、刪除                                         |
| wiki_read                    | Wiki     | 讀取 | 讀取 Wiki：列出頁面、取得內容、修訂歷史                              |
| wiki_write                   | Wiki     | 寫入 | 寫入 Wiki 頁面：創建、更新、刪除                                     |
| timetracking_read            | 時間追蹤 | 讀取 | 讀取時間追蹤：問題/倉庫耗時、活動計時器、我的追蹤記錄                |
| timetracking_write           | 時間追蹤 | 寫入 | 寫入時間追蹤：計時器和記錄項目                                       |
| package_read                 | 軟體套件 | 讀取 | 讀取軟體套件註冊表：列出套件、列出版本或取得某個版本                 |
| package_write                | 軟體套件 | 寫入 | 刪除軟體套件版本（不可復原）                                         |
| list_issues                  | 問題     | 讀取 | 列出倉庫問題                                                         |
| issue_read                   | 問題     | 讀取 | 讀取問題：詳情、評論或標籤                                           |
| issue_write                  | 問題     | 寫入 | 寫入問題：創建、更新、管理評論和標籤                                 |
| list_pull_requests           | 拉取請求 | 讀取 | 列出倉庫拉取請求                                                     |
| pull_request_read            | 拉取請求 | 讀取 | 讀取拉取請求：詳情、差異、變更檔案、頭部提交狀態、審查               |
| pull_request_write           | 拉取請求 | 寫入 | 寫入拉取請求：創建、更新、關閉、重新開啟、合併、更新分支、管理審查者 |
| pull_request_review_write    | 拉取請求 | 寫入 | 寫入 PR 審查：創建、提交、刪除、駁回                                 |
| actions_config_read          | Actions  | 讀取 | 讀取 Actions 密鑰和變數                                              |
| actions_config_write         | Actions  | 寫入 | 寫入 Actions 密鑰和變數：更新插入、創建、更新、刪除                  |
| actions_run_read             | Actions  | 讀取 | 讀取 Actions 工作流程、執行、作業、日誌和產物                        |
| actions_run_write            | Actions  | 寫入 | 寫入 Actions 執行：觸發、取消、重新執行                              |
| create_repo                  | 倉庫     | 寫入 | 創建新倉庫                                                           |
| fork_repo                    | 倉庫     | 寫入 | 復刻倉庫                                                             |
| list_my_repos                | 倉庫     | 讀取 | 列出目前用戶擁有的倉庫                                               |
| list_org_repos               | 倉庫     | 讀取 | 列出組織中的倉庫                                                     |
| get_repository_tree          | 倉庫     | 讀取 | 取得倉庫檔案樹                                                       |
| get_file_contents            | 文件     | 讀取 | 取得檔案內容與中繼資料                                               |
| get_dir_contents             | 文件     | 讀取 | 取得目錄中的項目                                                     |
| create_or_update_file        | 文件     | 寫入 | 創建或更新檔案（提供 sha 以更新現有檔案）                            |
| delete_file                  | 文件     | 寫入 | 刪除檔案                                                             |
| create_branch                | 分支     | 寫入 | 創建新分支                                                           |
| delete_branch                | 分支     | 寫入 | 刪除分支                                                             |
| list_branches                | 分支     | 讀取 | 列出倉庫分支                                                         |
| create_tag                   | Git 標籤 | 寫入 | 創建標籤                                                             |
| delete_tag                   | Git 標籤 | 寫入 | 刪除標籤                                                             |
| get_tag                      | Git 標籤 | 讀取 | 取得標籤詳情                                                         |
| list_tags                    | Git 標籤 | 讀取 | 列出倉庫標籤                                                         |
| list_commits                 | 提交     | 讀取 | 列出倉庫提交                                                         |
| get_commit                   | 提交     | 讀取 | 取得提交詳情                                                         |
| create_release               | 版本發布 | 寫入 | 創建版本發布                                                         |
| delete_release               | 版本發布 | 寫入 | 刪除版本發布                                                         |
| get_release                  | 版本發布 | 讀取 | 依 ID 取得版本發布                                                   |
| get_latest_release           | 版本發布 | 讀取 | 取得最新版本發布                                                     |
| list_releases                | 版本發布 | 讀取 | 列出倉庫版本發布                                                     |

> **說明：** 部分工具是聚合的、基於操作的工具，單個工具透過 `method` 參數暴露多個操作。當伺服器以唯讀模式執行時（`-r` / `GITEA_READONLY`），存取為「寫入」的工具會被隱藏；可透過 `-O` / `--tools`（`GITEA_TOOLS`）過濾對外暴露的工具集合。

許多工具支援 `page` 和 `per_page` 分頁參數。最大有效頁面大小由 Gitea 伺服器的 `[api].MAX_RESPONSE_ITEMS` 設定決定（預設 **50**），超出的值會被靜默截斷。
