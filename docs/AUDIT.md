# 审计日志（Audit）

gitli 的审计系统记录所有敏感操作，供安全合规、异常检测与事后追溯。

## 1. 数据模型

主表 `audit_logs`：

| 列 | 说明 |
| --- | --- |
| `id` | 主键 |
| `actor_id` | 操作者用户 id（可空，如系统操作） |
| `actor_type` | `user` / `admin` / `system` / `runner` |
| `actor_ip` / `actor_user_agent` | 来源 IP 与 UA |
| `event` | 标准化事件名，`resource.action` 格式 |
| `resource` / `action` | 从 event 拆分 |
| `target_type` / `target_id` / `target_name` | 操作目标 |
| `old_value` / `new_value` | 变更前后值（JSON） |
| `result` | `success` / `failure` / `denied` |
| `error_message` | 失败原因 |
| `request_id` | 请求关联 id |
| `metadata` | 附加信息（JSON） |
| `created_at` | 时间 |

索引：`(actor_id, created_at)`、`(event, created_at)`、`(target_type, target_id, created_at)`、`(result, created_at)`。

归档表 `audit_logs_archive`：结构与主表一致，额外 `orig_id`、`archived_at`。

告警表 `audit_alert_rules`：`name`、`event`、`threshold`、`window_seconds`、`channel`、`target`、`enabled`、`last_fired_at`。

## 2. 事件清单

### 认证
`auth.login.success`、`auth.login.failure`、`auth.logout`、`auth.password.change`、`auth.2fa.enable`、`auth.2fa.disable`、`auth.session.revoke`、`auth.oidc_login`

### 用户
`user.create`、`user.update`、`user.rename`、`user.delete`、`user.register`

### 密钥与令牌
`user.sshkey_add`、`user.sshkey_update`、`user.sshkey_delete`、`user.gpgkey_add`、`user.gpgkey_delete`、`user.pat_create`、`user.pat_revoke`

### 组织与团队
`org.create`、`org.member.add`、`org.member.remove`、`team.create`、`team.repo.grant`、`team.member.add`、`team.member.remove`

### 仓库
`repo.create`、`repo.delete`、`repo.transfer`、`repo.fork`、`repo.branch_protection_add`、`repo.tag_protection_add`、`repo.collaborator_add`、`repo.archive`、`repo.unarchive`

### Git
`git.push`、`git.protected_branch.push.failure`

### 协作
`issue.create`、`pull.create`、`pull.merge`、`issue_comment.create`

### 发布与包
`release.*`、`package.publish`、`release.asset_upload`

### 管理
`webhook.create`、`runner.job_cancel`、`admin.audit.export`

> 事件命名遵循 `resource.action`；新增敏感操作时必须补充对应事件。

## 3. 服务接口

`internal/service/audit` 提供：

- `Log(Entry)`：异步写入（队列满时降级同步）。推荐业务侧使用。
- `Record(ctx, Entry)`：同步写入（向后兼容）。
- `Query(ctx, Query)`：按 event/actor/target/result/时间过滤，分页（默认 50，最大 200）。
- `Get(ctx, id)`：单条详情。
- `Stream(ctx, Query, format, w)`：流式导出 CSV / JSON，不载入内存。
- `StatsSince(ctx, since)`：按天/事件/用户聚合。
- `Archive(ctx, retentionDays)`：迁移超期日志到归档表。
- `CreateAlertRule` / `ListAlertRules` / `UpdateAlertRule` / `DeleteAlertRule` / `EvaluateAlerts` / `MarkAlertFired`。

## 4. HTTP API（需管理员）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/v1/admin/audit` | 分页查询（`event`、`actor_id`、`result`、`target`、`from`、`to`、`limit`、`offset`） |
| GET | `/api/v1/admin/audit/{id}` | 单条详情 |
| GET | `/api/v1/admin/audit/stats?days=30` | 统计 |
| GET | `/api/v1/admin/audit/export?format=csv\|json` | 流式导出 |
| POST | `/api/v1/admin/audit/archive?retention_days=365` | 手动归档 |
| GET/POST | `/api/v1/admin/audit/alert-rules` | 告警规则 |
| DELETE | `/api/v1/admin/audit/alert-rules/{id}` | 删除规则 |

用户侧：`GET /api/v1/user/activity`（自己的审计记录）；`GET/DELETE /api/v1/user/me/sessions`（会话管理）。

## 5. 保留与归档

```toml
[audit]
retention_days = 365          # 在线保留天数
archive_retention_days = 1095 # 归档额外保留
```

主服务每日执行一次归档任务：将超过 `retention_days` 的记录迁移到 `audit_logs_archive` 并从主表删除。

## 6. 告警规则

示例：5 分钟内 `auth.login.failure` 超过 10 次发送邮件。

```json
{
  "name": "brute-force",
  "event": "auth.login.failure",
  "threshold": 10,
  "window_seconds": 300,
  "channel": "email",
  "target": "security@example.com",
  "enabled": true
}
```

`channel` 支持 `email` 与 `webhook`。

## 7. 安全与性能

- 异步写入，不阻塞业务。
- 查询走索引；分页上限 200。
- 导出流式处理，禁止全量载入内存。
- 审计记录不可修改、不可删除（仅归档）。
- 仅管理员可查询/导出。
