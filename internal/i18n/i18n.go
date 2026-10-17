// SPDX-License-Identifier: AGPL-3.0
// Package i18n localizes a small set of server-side messages based on the
// Accept-Language header.
package i18n

import (
	"net/http"
	"strings"
)

// Lang is a supported language code.
type Lang string

const (
	ZH Lang = "zh-CN"
	EN Lang = "en-US"
)

var messages = map[Lang]map[string]string{
	ZH: {
		"auth.required":       "需要身份认证",
		"forbidden":           "权限不足",
		"not_found":           "资源不存在",
		"archived":            "仓库已归档（只读）",
		"bad_request":         "请求无效",
		"rate_limited":        "请求过于频繁，请稍后再试",
		"invalid_credentials": "用户名或密码错误",
	},
	EN: {
		"auth.required":       "authentication required",
		"forbidden":           "insufficient permissions",
		"not_found":           "not found",
		"archived":            "repository is archived (read-only)",
		"bad_request":         "invalid request",
		"rate_limited":        "too many requests, please try again later",
		"invalid_credentials": "invalid username or password",
	},
}

// FromRequest determines the preferred language from Accept-Language.
func FromRequest(r *http.Request) Lang {
	al := r.Header.Get("Accept-Language")
	for _, part := range strings.Split(al, ",") {
		tag := strings.ToLower(strings.TrimSpace(strings.SplitN(part, ";", 2)[0]))
		if strings.HasPrefix(tag, "zh") {
			return ZH
		}
		if strings.HasPrefix(tag, "en") {
			return EN
		}
	}
	return ZH
}

// T returns the localized message for key in lang, falling back to Chinese.
func T(lang Lang, key string) string {
	if m, ok := messages[lang]; ok {
		if v, ok := m[key]; ok {
			return v
		}
	}
	if v, ok := messages[ZH][key]; ok {
		return v
	}
	return key
}

// Localize localizes a key using the request's Accept-Language.
func Localize(r *http.Request, key string) string {
	return T(FromRequest(r), key)
}
