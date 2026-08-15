# LumiBot CI 说明

本文档说明项目的持续集成（CI）流程、分支规范与合并规则。

## 1. CI 作用

每次代码变更（push / Pull Request）自动在 GitHub Actions 上执行代码检查，
在合并前发现格式、静态分析、测试问题，保证 `develop` / `main` 分支质量。

**触发时机：**

| 事件 | 分支 |
| --- | --- |
| push | `develop`、`main` |
| pull_request | 目标为 `develop`、`main` |

**执行环境：** `ubuntu-latest`，Go 1.24（依赖缓存开启，`go.sum` 未变化时跳过依赖下载）。

## 2. 自动检查内容（按顺序执行）

| 步骤 | 命令 / Action | 失败条件 |
| --- | --- | --- |
| 检出代码 | `actions/checkout@v4` | - |
| 安装 Go | `actions/setup-go@v5`（go-version: 1.24, cache: true） | - |
| 下载依赖 | `go mod download` | `go.mod` / `go.sum` 不一致或依赖缺失 |
| 格式检查 | `gofmt -l .` | 存在未格式化文件 |
| 静态检查 | `go vet ./...` | 潜在错误 / 不安全代码 |
| lint | `golangci-lint-action@v6` + `.golangci.yml` | unused / govet / staticcheck / ineffassign 发现违规 |
| 单元测试 | `go test ./...` | 任一测试失败（禁止合并） |

> 本地可先执行等价检查：`gofmt -l .`、`go vet ./...`、`go test ./...`。

## 3. Pull Request 流程

```
feature/xxx
    │  push
    ▼
创建 PR → 目标分支 develop
    │
    ├─ CI 自动运行（未通过则无法合并）
    │
    ├─ Code Review（至少 1 人通过）
    │
    └─ 合并（Squash）→ 删除 feature 分支
```

## 4. 开发分支规范

| 分支 | 说明 |
| --- | --- |
| `main` | 生产分支，只接受来自 `develop` 的合并，禁止直接 push |
| `develop` | 集成分支，feature 合并目标 |
| `feature/*` | 功能分支，从 `develop` 检出，命名如 `feature/command-system` |

## 5. 合并规则

- **feature → develop**：需 Pull Request + CI 全部通过 + Code Review
- **develop → main**：需 Pull Request + CI 全部通过 + 合并自 `develop`（禁止跳过 develop 直接合入 main）

## 6. GitHub Branch Protection 配置（需在仓库 Settings → Branches 开启）

| 分支 | 规则 |
| --- | --- |
| `develop` | ☑ Require a pull request before merging<br>☑ Require status checks to pass before merging（选择 `Go CI` 工作流）<br>☑ Require branches to be up to date<br>☑ Require conversation resolution（可选） |
| `main` | ☑ Require a pull request before merging<br>☑ Require status checks to pass before merging（选择 `Go CI` 工作流）<br>☑ Require branches to be up to date<br>☑ 禁止直接 push（Include administrators 可选，建议开启） |

> 状态检查名称：工作流 job 名为 `Go CI`（在 PR 页面显示为 `ci / Go CI`）。

## 7. 生产部署方式

本项目采用**传统二进制部署**（不使用 Docker / Kubernetes）：

```bash
# 构建（CI 通过后本地或构建机执行）
CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o lumibot ./cmd/bot

# 部署到服务器
scp lumibot user@server:/opt/lumibot/
```

服务器上使用 **systemd** 管理服务生命周期：

```ini
# /etc/systemd/system/lumibot.service
[Unit]
Description=LumiBot QQ Bot Service
After=network-online.target

[Service]
WorkingDirectory=/opt/lumibot
ExecStart=/opt/lumibot/lumibot
Restart=always
RestartSec=5
EnvironmentFile=/opt/lumibot/.env

[Install]
WantedBy=multi-user.target
```

```bash
systemctl daemon-reload && systemctl enable --now lumibot
```

自动构建、产物上传、服务器部署将在后续 CD 阶段实现（本项目暂不包含）。
