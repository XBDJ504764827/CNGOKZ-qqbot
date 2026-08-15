// Package logger 基于 zap 提供结构化日志，并适配 botgo SDK 的日志接口，
// 使 SDK 内部日志与业务日志统一输出，方便后续排查机器人事件。
package logger

import (
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/XBDJ504764827/LumiBot/internal/config"
)

// New 根据配置创建结构化 logger。
//
//   - json 格式：输出到 stdout，适合容器 / 日志平台采集
//   - console 格式：彩色控制台输出，适合本地开发
func New(cfg config.LogConfig) (*zap.Logger, error) {
	level, err := parseLevel(cfg.Level)
	if err != nil {
		return nil, err
	}

	encoderCfg := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	var encoder zapcore.Encoder
	switch cfg.Format {
	case "json":
		encoder = zapcore.NewJSONEncoder(encoderCfg)
	default:
		encoder = zapcore.NewConsoleEncoder(encoderCfg)
	}

	core := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level)
	return zap.New(core, zap.AddCaller()), nil
}

// parseLevel 解析日志级别配置。
func parseLevel(s string) (zapcore.Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return zapcore.DebugLevel, nil
	case "info":
		return zapcore.InfoLevel, nil
	case "warn", "warning":
		return zapcore.WarnLevel, nil
	case "error":
		return zapcore.ErrorLevel, nil
	default:
		return zapcore.InfoLevel, fmt.Errorf("LOG_LEVEL 取值非法 %q，支持 debug / info / warn / error", s)
	}
}

// BotgoAdapter 将 zap.Logger 适配为 botgo SDK 的 log.Logger 接口，
// 通过 botgo.SetLogger(NewBotgoAdapter(logger)) 注入。
type BotgoAdapter struct {
	zap *zap.Logger
}

// NewBotgoAdapter 创建 botgo 日志适配器。
func NewBotgoAdapter(logger *zap.Logger) *BotgoAdapter {
	return &BotgoAdapter{zap: logger}
}

func (a *BotgoAdapter) Debug(v ...interface{}) {
	a.zap.Debug(fmt.Sprint(v...))
}

func (a *BotgoAdapter) Info(v ...interface{}) {
	a.zap.Info(fmt.Sprint(v...))
}

func (a *BotgoAdapter) Warn(v ...interface{}) {
	a.zap.Warn(fmt.Sprint(v...))
}

func (a *BotgoAdapter) Error(v ...interface{}) {
	a.zap.Error(fmt.Sprint(v...))
}

func (a *BotgoAdapter) Debugf(format string, v ...interface{}) {
	a.zap.Debug(fmt.Sprintf(format, v...))
}

func (a *BotgoAdapter) Infof(format string, v ...interface{}) {
	a.zap.Info(fmt.Sprintf(format, v...))
}

func (a *BotgoAdapter) Warnf(format string, v ...interface{}) {
	a.zap.Warn(fmt.Sprintf(format, v...))
}

func (a *BotgoAdapter) Errorf(format string, v ...interface{}) {
	a.zap.Error(fmt.Sprintf(format, v...))
}

func (a *BotgoAdapter) Sync() error {
	return a.zap.Sync()
}
