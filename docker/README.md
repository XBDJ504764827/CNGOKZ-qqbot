# Docker 部署说明

## 文件说明

| 文件 | 用途 |
| --- | --- |
| `../Dockerfile` | 多阶段构建：golang:1.24-alpine 编译 → alpine 运行（静态二进制） |
| `../docker-compose.yml` | 一键编排：构建、环境变量注入、端口映射、健康检查、自动重启 |

## 快速开始

```bash
# 1. 准备 .env（compose 自动读取进行变量插值）
cp ../configs/.env.example ../.env
#    编辑 ../.env 填写 QQ_APP_ID / QQ_SECRET

# 2. 构建并启动
docker compose up -d --build

# 3. 验证
curl http://127.0.0.1:8080/health

# 4. 日志与停止
docker compose logs -f lumibot
docker compose down
```

## 常用运维命令

```bash
docker compose ps          # 查看状态
docker compose restart     # 重启
docker compose logs --tail=100 lumibot   # 最近 100 行日志
docker compose exec lumibot wget -q -O - http://127.0.0.1:8080/health   # 容器内自检
```

## 注意事项

1. **环境变量注入**：compose 通过 `${VAR:-default}` 插值注入环境变量，
   `.env` 文件缺失时服务仍可启动，但缺少 `QQ_APP_ID / QQ_SECRET` 会因配置校验失败而退出
2. **健康检查**：基于容器内 busybox `wget`，探测 `/health`，失败自动重启（`restart: unless-stopped`）
3. **时区**：镜像默认 `TZ=Asia/Shanghai`
4. **构建缓存**：Dockerfile 先拷贝 `go.mod / go.sum` 再 `go mod download`，
   依赖未变化时后续构建可命中缓存层
