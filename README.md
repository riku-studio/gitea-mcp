# Gitea MCP Server

[繁體中文](README.zh-tw.md) | [简体中文](README.zh-cn.md)

**Gitea MCP Server** connects a [Gitea](https://about.gitea.com) instance to [Model Context Protocol](https://modelcontextprotocol.io) clients, so repositories, issues, pull requests and more can be browsed and managed from an MCP-compatible chat interface.

[![Install with Docker in VS Code](https://img.shields.io/badge/VS_Code-Install_Server-0098FF?style=flat-square&logo=visualstudiocode&logoColor=white)](https://insiders.vscode.dev/redirect/mcp/install?name=gitea&inputs=[{%22id%22:%22gitea_token%22,%22type%22:%22promptString%22,%22description%22:%22Gitea%20Personal%20Access%20Token%22,%22password%22:true}]&config={%22command%22:%22docker%22,%22args%22:[%22run%22,%22-i%22,%22--rm%22,%22-e%22,%22GITEA_ACCESS_TOKEN%22,%22docker.gitea.com/gitea-mcp-server%22],%22env%22:{%22GITEA_ACCESS_TOKEN%22:%22${input:gitea_token}%22}}) [![Install with Docker in VS Code Insiders](https://img.shields.io/badge/VS_Code_Insiders-Install_Server-24bfa5?style=flat-square&logo=visualstudiocode&logoColor=white)](https://insiders.vscode.dev/redirect/mcp/install?name=gitea&inputs=[{%22id%22:%22gitea_token%22,%22type%22:%22promptString%22,%22description%22:%22Gitea%20Personal%20Access%20Token%22,%22password%22:true}]&config={%22command%22:%22docker%22,%22args%22:[%22run%22,%22-i%22,%22--rm%22,%22-e%22,%22GITEA_ACCESS_TOKEN%22,%22docker.gitea.com/gitea-mcp-server%22],%22env%22:{%22GITEA_ACCESS_TOKEN%22:%22${input:gitea_token}%22}}&quality=insiders)

## Installation

Download a binary from the [releases page](https://gitea.com/gitea/gitea-mcp/releases) and put it in your `PATH`, use the `docker.gitea.com/gitea-mcp-server` image, or build from source into `$GOPATH/bin` with `make` and Go 1.26 or later:

```bash
git clone https://gitea.com/gitea/gitea-mcp.git
cd gitea-mcp
make install
```

## Configuration

Pass the Gitea host and access token as command-line flags or environment variables, flags take precedence. Run `gitea-mcp --help` for the full list of flags and environment variables. Logs are written to `$HOME/.gitea-mcp/gitea-mcp.log`, add `-d` for debug logging.

### Claude Code

Runs the server through `go run` and requires [Go](https://go.dev):

```bash
claude mcp add --transport stdio --scope user gitea \
  --env GITEA_ACCESS_TOKEN=token \
  --env GITEA_HOST=https://gitea.com \
  -- go run gitea.com/gitea/gitea-mcp@latest -t stdio
```

### VS Code

Use the install buttons at the top of this README, or add the block below to your User Settings (JSON), reachable via `Ctrl + Shift + P` and `Preferences: Open User Settings (JSON)`. It also works in a workspace `.vscode/mcp.json`, where the `mcp` key is omitted.

```json
{
  "mcp": {
    "inputs": [
      {
        "type": "promptString",
        "id": "gitea_token",
        "description": "Gitea Personal Access Token",
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

Add the following to the top-level `mcp` object of your [OpenCode](https://opencode.ai) config:

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

Add the following to `~/.vibe/config.toml`:

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

### Other clients

Clients such as Cursor take either a stdio command:

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

or an http endpoint, for a server started with `gitea-mcp -t http --port 8080`:

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

Once configured, try `list all my repositories` in the chat box.

## Available Tools

| Tool                         | Scope        | Access | Description                                                                                |
| :--------------------------- | :----------- | :----- | :----------------------------------------------------------------------------------------- |
| get_gitea_mcp_server_version | Version      | Read   | Get the Gitea MCP server version                                                           |
| get_me                       | User         | Read   | Get the current authenticated user                                                         |
| get_user_orgs                | User         | Read   | List the current user's organizations                                                      |
| search_users                 | Search       | Read   | Search for users                                                                           |
| search_org_teams             | Search       | Read   | Search teams within an organization                                                        |
| search_repos                 | Search       | Read   | Search for repositories                                                                    |
| search_issues                | Search       | Read   | Search issues and pull requests across repositories                                        |
| notification_read            | Notification | Read   | Read notifications: list (optionally scoped to a repo) or get a thread by ID               |
| notification_write           | Notification | Write  | Mark a notification or all notifications as read                                           |
| label_read                   | Label        | Read   | Read repository or organization labels                                                     |
| label_write                  | Label        | Write  | Write labels (repo or org): create, edit, delete                                           |
| milestone_read               | Milestone    | Read   | Read milestones: get one or list                                                           |
| milestone_write              | Milestone    | Write  | Write milestones: create, update, delete                                                   |
| wiki_read                    | Wiki         | Read   | Read wiki: list pages, get content, revision history                                       |
| wiki_write                   | Wiki         | Write  | Write wiki pages: create, update, delete                                                   |
| timetracking_read            | Timetracking | Read   | Read time tracking: issue/repo times, active stopwatches, your tracked times               |
| timetracking_write           | Timetracking | Write  | Write time tracking: stopwatches and entries                                               |
| package_read                 | Packages     | Read   | Read package registry: list packages, list versions, or get a version                      |
| package_write                | Packages     | Write  | Delete a package version (irreversible)                                                    |
| list_issues                  | Issue        | Read   | List repository issues                                                                     |
| issue_read                   | Issue        | Read   | Read issue: details, comments, or labels                                                   |
| issue_write                  | Issue        | Write  | Write issues: create, update, manage comments and labels                                   |
| list_pull_requests           | Pull Request | Read   | List repository pull requests                                                              |
| pull_request_read            | Pull Request | Read   | Read pull request: details, diff, changed files, head commit status, reviews               |
| pull_request_write           | Pull Request | Write  | Write pull requests: create, update, close, reopen, merge, update branch, manage reviewers |
| pull_request_review_write    | Pull Request | Write  | Write PR reviews: create, submit, delete, dismiss                                          |
| actions_config_read          | Actions      | Read   | Read Actions secrets and variables                                                         |
| actions_config_write         | Actions      | Write  | Write Actions secrets and variables: upsert, create, update, delete                        |
| actions_run_read             | Actions      | Read   | Read Actions workflows, runs, jobs, logs, and artifacts                                    |
| actions_run_write            | Actions      | Write  | Write Actions runs: dispatch, cancel, rerun                                                |
| create_repo                  | Repository   | Write  | Create a new repository                                                                    |
| fork_repo                    | Repository   | Write  | Fork a repository                                                                          |
| list_my_repos                | Repository   | Read   | List repositories owned by the current user                                                |
| list_org_repos               | Repository   | Read   | List repositories in an organization                                                       |
| get_repository_tree          | Repository   | Read   | Get the repository file tree                                                               |
| get_file_contents            | File         | Read   | Get file content and metadata                                                              |
| get_dir_contents             | File         | Read   | Get the entries in a directory                                                             |
| create_or_update_file        | File         | Write  | Create or update a file (provide sha to update an existing file)                           |
| delete_file                  | File         | Write  | Delete a file                                                                              |
| create_branch                | Branch       | Write  | Create a new branch                                                                        |
| delete_branch                | Branch       | Write  | Delete a branch                                                                            |
| list_branches                | Branch       | Read   | List repository branches                                                                   |
| create_tag                   | Tag          | Write  | Create a tag                                                                               |
| delete_tag                   | Tag          | Write  | Delete a tag                                                                               |
| get_tag                      | Tag          | Read   | Get tag details                                                                            |
| list_tags                    | Tag          | Read   | List repository tags                                                                       |
| list_commits                 | Commit       | Read   | List repository commits                                                                    |
| get_commit                   | Commit       | Read   | Get commit details                                                                         |
| create_release               | Release      | Write  | Create a release                                                                           |
| delete_release               | Release      | Write  | Delete a release                                                                           |
| get_release                  | Release      | Read   | Get a release by ID                                                                        |
| get_latest_release           | Release      | Read   | Get the latest release                                                                     |
| list_releases                | Release      | Read   | List repository releases                                                                   |

> **Note:** Several tools are consolidated, action-based tools, a single tool exposes multiple operations through a `method` parameter. Tools with `Write` access are hidden when the server runs in read-only mode (`-r` / `GITEA_READONLY`), and the exposed tool set can be filtered with `-O` / `--tools` (`GITEA_TOOLS`).

Many tools accept `page` and `per_page` for pagination. The maximum effective page size is the Gitea server's `[api].MAX_RESPONSE_ITEMS` setting (default **50**), larger values are silently capped.
