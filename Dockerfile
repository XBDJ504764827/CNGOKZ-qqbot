# ===== 构建阶段 =====
FROM golang:1.24-alpine AS builder

WORKDIR /build

# 先拷贝依赖清单，充分利用 Docker 层缓存
COPY go.mod go.sum ./
RUN go mod download

# 拷贝源码并编译
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/lumibot ./cmd/bot

# ===== 运行阶段 =====
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Shanghai

WORKDIR /app
COPY --from=builder /out/lumibot /app/lumibot
COPY configs/.env.example /app/configs/.env.example

EXPOSE 8080

ENTRYPOINT ["/app/lumibot"]
