package i18n

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

// I18n 是由 Core 和模块语言资源合并出的只读国际化目录。
type I18n struct {
	bundle      *i18n.Bundle
	sourceKeys  map[string]string
	messageKeys map[string]struct{}
}

// NewI18n 从宿主语言文件系统收集国际化消息和消息键。
func NewI18n(resourceName string, files fs.FS) (*I18n, error) {
	if resourceName == "" {
		resourceName = "resource"
	}
	if files == nil {
		return nil, fmt.Errorf("国际化资源 %s 未提供文件系统", resourceName)
	}
	entries, err := fs.ReadDir(files, ".")
	if err != nil {
		return nil, fmt.Errorf("读取国际化资源 %s: %w", resourceName, err)
	}
	locales := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			continue
		}
		locales = append(locales, strings.TrimSuffix(entry.Name(), path.Ext(entry.Name())))
	}
	sort.Strings(locales)
	if len(locales) == 0 {
		return &I18n{}, nil
	}

	bundle := i18n.NewBundle(language.Und)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)
	sourceKeys := make(map[string]string)
	messageKeys := make(map[string]struct{})
	for _, localeValue := range locales {
		if _, err = language.Parse(localeValue); err != nil {
			return nil, fmt.Errorf("解析国际化语言 %s: %w", localeValue, err)
		}
		var data []byte
		data, err = fs.ReadFile(files, localeValue+".json")
		if err != nil {
			return nil, fmt.Errorf("读取国际化资源 %s/%s.json: %w", resourceName, localeValue, err)
		}
		if len(strings.TrimSpace(string(data))) == 0 {
			continue
		}
		var messages map[string]json.RawMessage
		if err = json.Unmarshal(data, &messages); err != nil {
			return nil, fmt.Errorf("解析国际化资源 %s/%s.json: %w", resourceName, localeValue, err)
		}
		if _, err = bundle.ParseMessageFileBytes(data, localeValue+".json"); err != nil {
			return nil, fmt.Errorf("加载国际化目录 %s: %w", localeValue, err)
		}
		for messageKey, messageData := range messages {
			messageKeys[messageKey] = struct{}{}
			var message struct {
				Other string `json:"other"`
			}
			if err = json.Unmarshal(messageData, &message); err != nil {
				return nil, fmt.Errorf("解析国际化源文 %s: %w", messageKey, err)
			}
			if message.Other == "" {
				continue
			}
			if currentKey, exists := sourceKeys[message.Other]; !exists || messageKey < currentKey {
				sourceKeys[message.Other] = messageKey
			}
		}
	}
	return &I18n{bundle: bundle, sourceKeys: sourceKeys, messageKeys: messageKeys}, nil
}

// Empty 判断消息目录是否没有可用消息。
func (c *I18n) Empty() bool {
	return c == nil || c.bundle == nil || len(c.messageKeys) == 0
}

// KeyForSource 返回语言包中与源文完全匹配的消息键。
func (c *I18n) KeyForSource(source string) (string, bool) {
	if c == nil || source == "" {
		return "", false
	}
	key, ok := c.sourceKeys[source]
	return key, ok
}

// HasMessage 判断目录中是否存在指定消息键。
func (c *I18n) HasMessage(messageKey string) bool {
	if c == nil || messageKey == "" {
		return false
	}
	_, ok := c.messageKeys[messageKey]
	return ok
}

// Localize 按当前语言和主语言渲染消息，缺少译文时回退原始消息。
func (c *I18n) Localize(localeValue, fallbackLocale, messageKey string, messageArgs map[string]any, fallback string) string {
	if c == nil || c.bundle == nil || messageKey == "" {
		return fallbackMessage(fallback)
	}
	localizer := i18n.NewLocalizer(c.bundle, localeValue)
	message, err := localizer.Localize(&i18n.LocalizeConfig{MessageID: messageKey, TemplateData: messageArgs})
	if err == nil && message != "" {
		return message
	}
	if fallbackLocale != "" && fallbackLocale != localeValue {
		localizer = i18n.NewLocalizer(c.bundle, fallbackLocale)
		message, err = localizer.Localize(&i18n.LocalizeConfig{MessageID: messageKey, TemplateData: messageArgs})
		if err == nil && message != "" {
			return message
		}
	}
	return fallbackMessage(fallback)
}

func fallbackMessage(fallback string) string {
	if fallback != "" {
		return fallback
	}
	return "系统内部错误"
}
