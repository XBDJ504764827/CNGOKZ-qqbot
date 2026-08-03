package notification

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/XBDJ504764827/LumiBot/internal/event"
)

// 内置事件模板（每个事件类型独立模板，禁止在 Service / Handler 中硬编码消息格式）。
//
// 模板变量说明：
//   - {{.Title}} / {{.Message}}      事件标题 / 描述
//   - {{.TimeText}}                  事件时间（2006-01-02 15:04）
//   - {{.LevelText}}                 级别中文（普通/警告/错误/严重）
//   - {{get .Data "key"}}            事件 data 字段取值，缺失时输出 "-"
var builtinTemplates = map[string]eventTemplate{
	event.EventServerOffline: {
		title: "服务器异常",
		body: `[服务器异常]

服务器: {{get .Data "server"}}
状态: {{get .Data "status"}}
时间: {{.TimeText}}

{{.Message}}
请管理员处理。`,
	},
	event.EventServerOnline: {
		title: "服务器恢复",
		body: `[服务器恢复]

服务器: {{get .Data "server"}}
时间: {{.TimeText}}

{{.Message}}`,
	},
	event.EventSystemWarning: {
		title: "系统警告",
		body: `[系统警告]

级别: {{.LevelText}}
时间: {{.TimeText}}

{{.Title}}: {{.Message}}
请管理员关注。`,
	},
	event.EventForumReportCreated: {
		title: "论坛举报",
		body: `[论坛举报]

举报人: {{get .Data "reporter"}}
帖子: {{get .Data "post_title"}}
时间: {{.TimeText}}

{{.Message}}
请管理员处理。`,
	},
	event.EventAdminAction: {
		title: "管理操作",
		body: `[管理操作]

操作人: {{get .Data "operator"}}
时间: {{.TimeText}}

{{.Message}}`,
	},
	event.EventWhitelistRequestCreated: {
		title: "新白名单申请",
		body: `[新白名单申请]

玩家: {{get .Data "nickname"}}
SteamID: {{get .Data "steamid64"}}
联系方式: {{get .Data "contact"}}
时间: {{.TimeText}}

{{.Message}}
请管理员审核。`,
	},
}

// fallbackTemplate 通用兜底模板（事件类型无专属模板或专属模板渲染失败时使用）。
const fallbackTemplate = `[{{.Title}}]

级别: {{.LevelText}}
时间: {{.TimeText}}

{{.Message}}`

// TemplateData 模板渲染数据。
type TemplateData struct {
	event.Event
}

// TimeText 事件时间（本地时区，分钟精度）。
func (d TemplateData) TimeText() string {
	return d.Timestamp.Format("2006-01-02 15:04")
}

// LevelText 事件级别中文。
func (d TemplateData) LevelText() string {
	return LevelText(d.Level)
}

// eventTemplate 单个事件类型模板。
type eventTemplate struct {
	title string
	body  string
}

// Templates 模板集合。
type Templates struct {
	byType   map[string]*template.Template
	titles   map[string]string
	fallback *template.Template
}

// NewTemplates 构建模板集合（内置全部事件类型模板）。
func NewTemplates() *Templates {
	funcs := template.FuncMap{
		// get 从事件 data 中安全取值，缺失时输出 "-"
		"get": func(data map[string]interface{}, key string) string {
			if data == nil {
				return "-"
			}
			if v, ok := data[key]; ok && v != nil {
				return fmt.Sprint(v)
			}
			return "-"
		},
	}

	t := &Templates{
		byType: make(map[string]*template.Template, len(builtinTemplates)),
		titles: make(map[string]string, len(builtinTemplates)),
	}
	for eventType, et := range builtinTemplates {
		t.byType[eventType] = template.Must(template.New(eventType).Funcs(funcs).Parse(et.body))
		t.titles[eventType] = et.title
	}
	t.fallback = template.Must(template.New("fallback").Funcs(funcs).Parse(fallbackTemplate))
	return t
}

// Render 渲染事件为通知标题与内容。
// 专属模板渲染失败时回退通用模板；通用模板也失败时返回纯文本兜底。
func (t *Templates) Render(ev event.Event) (title, content string) {
	data := TemplateData{Event: ev}

	// 1. 专属模板
	if tmpl, ok := t.byType[ev.EventType]; ok {
		if body, err := t.execute(tmpl, data); err == nil {
			return t.titles[ev.EventType], body
		}
	}

	// 2. 通用兜底模板
	if body, err := t.execute(t.fallback, data); err == nil {
		return ev.Title, body
	}

	// 3. 纯文本兜底（极端情况，保证通知不丢失）
	return ev.Title, fmt.Sprintf("[%s] %s (%s)", ev.Title, ev.Message, data.TimeText())
}

// execute 执行模板。
func (t *Templates) execute(tmpl *template.Template, data TemplateData) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("模板渲染失败: %w", err)
	}
	return buf.String(), nil
}
