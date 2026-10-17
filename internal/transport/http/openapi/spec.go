// SPDX-License-Identifier: AGPL-3.0
// Package openapi builds a minimal OpenAPI 3.0 description of the public API.
package openapi

// Spec returns the OpenAPI 3.0 document as a generic map ready for JSON encoding.
func Spec(baseURL string) map[string]any {
	paths := map[string]any{
		"/api/v1/version": op("get", "服务版本", nil),
		"/api/v1/auth/register": op("post", "注册用户", body(map[string]any{
			"username": str(), "email": str(), "password": str(),
		}, []string{"username", "email", "password"})),
		"/api/v1/auth/login": op("post", "登录", body(map[string]any{
			"username": str(), "email": str(), "password": str(), "totp_code": str(),
		}, []string{"username", "password"})),
		"/api/v1/auth/logout":                        op("post", "登出", nil),
		"/api/v1/user/me":                            op("get", "当前用户", nil),
		"/api/v1/user/repos":                         op("get", "我的仓库", nil),
		"/api/v1/user/me/tokens":                     op("get", "列出访问令牌", nil),
		"/api/v1/user/me/keys":                       op("get", "列出 SSH 密钥", nil),
		"/api/v1/user/me/gpg-keys":                   op("get", "列出 GPG 密钥", nil),
		"/api/v1/user/me/sessions":                   op("get", "列出登录会话", nil),
		"/api/v1/user/blockn":                        op("get", "占位", nil),
		"/api/v1/repos":                              op("post", "创建仓库", body(map[string]any{"name": str(), "description": str(), "is_private": boolT()}, []string{"name"})),
		"/api/v1/repos/{owner}/{repo}":               op("get", "仓库详情（含语言与许可证）", nil),
		"/api/v1/repos/{owner}/{repo}/branches":      op("get", "分支列表", nil),
		"/api/v1/repos/{owner}/{repo}/commits/{ref}": op("get", "提交历史", nil),
		"/api/v1/repos/{owner}/{repo}/contents":      op("get", "文件树", nil),
		"/api/v1/repos/{owner}/{repo}/issues":        op("get", "问题列表", nil),
		"/api/v1/repos/{owner}/{repo}/pulls":         op("get", "合并请求列表", nil),
		"/api/v1/repos/{owner}/{repo}/releases":      op("get", "发布列表", nil),
		"/api/v1/repos/{owner}/{repo}/keys":          op("get", "部署密钥", nil),
		"/api/v1/repos/{owner}/{repo}/webhooks":      op("get", "Webhook 列表", nil),
		"/api/v1/search":                             op("get", "全局搜索", nil),
		"/api/v1/explore/repos":                      op("get", "公开仓库", nil),
		"/api/v1/explore/trending":                   op("get", "趋势仓库", nil),
		"/api/v1/explore/topics":                     op("get", "热门话题", nil),
		"/api/v1/users/{username}":                   op("get", "用户主页", nil),
		"/api/v1/admin/audit":                        op("get", "审计日志", nil),
		"/api/v1/admin/audit/export":                 op("get", "导出审计日志", nil),
		"/api/v1/admin/runners":                      op("get", "Runner 列表", nil),
		"/healthz":                                   op("get", "健康检查", nil),
		"/metrics":                                   op("get", "Prometheus 指标", nil),
	}
	// Drop the placeholder key.
	delete(paths, "/api/v1/user/blockn")

	return map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       "gitli API",
			"description": "轻量级自研 Git 服务平台 API。认证：Session Cookie 或 Bearer Token。",
			"version":     "0.1.0",
			"license":     map[string]any{"name": "AGPL-3.0"},
		},
		"servers": []map[string]any{{"url": baseURL}},
		"components": map[string]any{
			"securitySchemes": map[string]any{
				"bearer":  map[string]any{"type": "http", "scheme": "bearer"},
				"session": map[string]any{"type": "apiKey", "in": "cookie", "name": "gitli_session"},
			},
			"schemas": map[string]any{
				"Error": map[string]any{
					"type":       "object",
					"properties": map[string]any{"error": str()},
				},
			},
		},
		"security": []map[string]any{{"bearer": []any{}}},
		"paths":    paths,
	}
}

func op(method, summary string, requestBody map[string]any) map[string]any {
	o := map[string]any{
		"summary": summary,
		"responses": map[string]any{
			"200": map[string]any{"description": "成功"},
			"401": map[string]any{"description": "未认证"},
			"403": map[string]any{"description": "无权限"},
		},
	}
	if requestBody != nil {
		o["requestBody"] = requestBody
	}
	return map[string]any{method: o}
}

func body(props map[string]any, required []string) map[string]any {
	return map[string]any{
		"required": true,
		"content": map[string]any{
			"application/json": map[string]any{
				"schema": map[string]any{
					"type":       "object",
					"properties": props,
					"required":   required,
				},
			},
		},
	}
}

func str() map[string]any   { return map[string]any{"type": "string"} }
func boolT() map[string]any { return map[string]any{"type": "boolean"} }
