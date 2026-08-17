// Package config 负责加载 LumiBot 的全部运行配置。
//
// 配置来源优先级（高 → 低）：
//  1. 进程环境变量
//  2. .env 文件（默认读取项目根目录 .env）
//  3. 内置默认值（集中定义于本包，禁止在业务代码中散落硬编码）
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Env 运行环境。
type Env string

const (
	// EnvDev 开发环境，连接 QQ 沙箱网关。
	EnvDev Env = "dev"
	// EnvProd 生产环境，连接 QQ 正式网关。
	EnvProd Env = "prod"
)

// Config 汇总全部配置项。
type Config struct {
	// Env 运行环境，决定连接沙箱还是正式 QQ 网关。
	Env Env
	// Bot QQ 官方机器人接入配置。
	Bot BotConfig
	// HTTP 内置 HTTP 服务配置（供 LumiAdmin 回调）。
	HTTP HTTPConfig
	// Event 统一事件系统配置。
	Event EventConfig
	// Notification 通知系统配置。
	Notification NotificationConfig
	// LumiAdmin LumiAdmin 后端集成配置（QQ 审批回写）。
	LumiAdmin LumiAdminConfig
	// Log 日志配置。
	Log LogConfig
}

// BotConfig QQ 官方机器人接入配置。
type BotConfig struct {
	// AppID QQ 开放平台机器人 AppID。
	AppID string
	// Token 机器人 Token（预留：旧式 BotToken 鉴权，新式鉴权使用 Secret）。
	Token string
	// Secret 机器人 AppSecret，用于新式 oauth2 鉴权。
	Secret string
	// Debug 是否开启 SDK 调试模式（BOT_DEBUG），输出更多过程日志。
	Debug bool
	// Timeout openapi 请求超时时间。
	Timeout time.Duration
}

// HTTPConfig 内置 HTTP 服务配置。
type HTTPConfig struct {
	// Addr HTTP 监听地址，如 ":8080"。
	Addr string
	// ReadTimeout 读请求超时。
	ReadTimeout time.Duration
	// WriteTimeout 写响应超时。
	WriteTimeout time.Duration
	// IdleTimeout 空闲连接超时。
	IdleTimeout time.Duration
}

// EventConfig 统一事件系统配置。
type EventConfig struct {
	// APIKeys 事件上报接口（POST /api/v1/events）允许的 API Key 列表，
	// 逗号分隔配置于 EVENT_API_KEYS。未配置时拒绝所有上报请求（fail-closed）。
	APIKeys []string
	// RateLimit 单来源（API Key）每分钟允许的事件上报次数（EVENT_RATE_LIMIT），
	// 防止异常系统大量发送事件。
	RateLimit int
	// RateWindow 限流时间窗口（固定 1 分钟）。
	RateWindow time.Duration
}

// NotificationConfig 通知系统配置。
type NotificationConfig struct {
	// Enable 是否启用通知（NOTIFICATION_ENABLE）。
	Enable bool
	// Cooldown 全局默认通知冷却时间（NOTICE_COOLDOWN，单位秒），
	// 同一事件类型在窗口内的重复事件只通知一次。
	Cooldown time.Duration
	// PrivateTarget QQ_PRIVATE 渠道目标：管理员 QQ openid（NOTIFY_PRIVATE_TARGET）。
	PrivateTarget string
	// ChannelTarget QQ_CHANNEL 渠道目标：子频道 ID（NOTIFY_CHANNEL_TARGET）。
	ChannelTarget string
}

// LumiAdminConfig LumiAdmin 后端集成配置。
type LumiAdminConfig struct {
	// CallbackBaseURL LumiAdmin 地址（LUMIADMIN_CALLBACK_URL），LumiBot 回调 LumiAdmin 做 QQ 审批。
	CallbackBaseURL string
	// IntegrationToken LumiAdmin 的 QQ 集成令牌（LUMIADMIN_QQ_TOKEN），用于调用审批接口鉴权。
	IntegrationToken string
	// ApprovalAuditPath QQ 白名单审批审计文件路径（QQ_APPROVAL_AUDIT_PATH）。
	ApprovalAuditPath string
}

// LogConfig 日志配置。
type LogConfig struct {
	// Level 日志级别：debug / info / warn / error。
	Level string
	// Format 输出格式：console（开发友好）/ json（生产采集）。
	Format string
}

// Load 加载配置：.env 文件提供默认值，同名环境变量优先。
func Load() (*Config, error) {
	// .env 文件不存在时忽略错误，纯环境变量方式仍可正常工作。
	_ = godotenv.Load()

	env := Env(getEnv("ENV", string(EnvDev)))

	cfg := &Config{
		Env: env,
		Bot: BotConfig{
			AppID:   getEnv("QQ_APP_ID", ""),
			Token:   getEnv("QQ_TOKEN", ""),
			Secret:  getEnv("QQ_SECRET", ""),
			Debug:   getBool("BOT_DEBUG", false),
			Timeout: getDuration("QQ_OPENAPI_TIMEOUT", 5*time.Second),
		},
		HTTP: HTTPConfig{
			Addr:         getEnv("HTTP_ADDR", ":8080"),
			ReadTimeout:  getDuration("HTTP_READ_TIMEOUT", 5*time.Second),
			WriteTimeout: getDuration("HTTP_WRITE_TIMEOUT", 10*time.Second),
			IdleTimeout:  getDuration("HTTP_IDLE_TIMEOUT", 60*time.Second),
		},
		Event: EventConfig{
			APIKeys:    getStringSlice("EVENT_API_KEYS"),
			RateLimit:  getInt("EVENT_RATE_LIMIT", 100),
			RateWindow: time.Minute,
		},
		Notification: NotificationConfig{
			Enable:        getBool("NOTIFICATION_ENABLE", true),
			Cooldown:      getSecondsDuration("NOTICE_COOLDOWN", 300*time.Second),
			PrivateTarget: getEnv("NOTIFY_PRIVATE_TARGET", ""),
			ChannelTarget: getEnv("NOTIFY_CHANNEL_TARGET", ""),
		},
		LumiAdmin: LumiAdminConfig{
			CallbackBaseURL:   getEnv("LUMIADMIN_CALLBACK_URL", ""),
			IntegrationToken:  getEnv("LUMIADMIN_QQ_TOKEN", ""),
			ApprovalAuditPath: getEnv("QQ_APPROVAL_AUDIT_PATH", "logs/qq-approval-audit.jsonl"),
		},
		Log: LogConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: logFormat(env),
		},
	}

	if env != EnvDev && env != EnvProd {
		return nil, fmt.Errorf("ENV 取值非法 %q，仅支持 %q / %q", env, EnvDev, EnvProd)
	}
	if cfg.Bot.AppID == "" || cfg.Bot.Secret == "" {
		return nil, fmt.Errorf("缺少必填配置 QQ_APP_ID / QQ_SECRET，请通过 .env 文件或环境变量提供")
	}
	return cfg, nil
}

// logFormat 根据运行环境推导日志格式：生产环境 JSON，开发环境 console。
func logFormat(env Env) string {
	if env == EnvProd {
		return "json"
	}
	return "console"
}

// getEnv 读取环境变量，未设置或为空时返回默认值。
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// getBool 读取布尔型环境变量（true/false/1/0），未设置或解析失败时返回默认值。
func getBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

// getStringSlice 读取逗号分隔的字符串列表环境变量（自动 trim，忽略空项）。
func getStringSlice(key string) []string {
	raw := os.Getenv(key)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// getInt 读取整型环境变量，非法或未设置时使用默认值。
func getInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

// getSecondsDuration 读取整秒数时长环境变量（如 "300"），非法或未设置时使用默认值。
func getSecondsDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return fallback
	}
	return time.Duration(n) * time.Second
}

// getDuration 读取时长型环境变量（如 "5s"、"30s"），解析失败时使用默认值。
func getDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
