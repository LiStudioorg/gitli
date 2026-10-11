// SPDX-License-Identifier: AGPL-3.0
package config

// defaultTemplate is written to disk on first run when config.toml is absent.
const defaultTemplate = `# gitli 配置文件（首次运行自动生成）
# 环境变量优先级高于此文件，前缀 GITLI_。

[server]
http = ":8080"
ssh = ":2222"
external_url = "http://localhost:8080"

[database]
driver = "sqlite"
dsn = "./data/gitli.db"
max_open_conns = 25
max_idle_conns = 5

[storage]
repos = "./data/repos"
lfs = "./data/lfs"
packages = "./data/packages"
avatars = "./data/avatars"
attachments = "./data/attachments"

[security.password]
algorithm = "argon2id"
memory_mb = 64
iterations = 3
parallelism = 2
salt_length = 16
key_length = 32
pepper = ""

[security.crypto]
key_hex = ""

[security.login]
max_attempts = 5
lockout_minutes = 15
window_minutes = 15

[modules]
core = true
collaboration = true
git_ext = true
registry = true
ci = false
ide = false
ops = true
search = "grep"

[modules.packages]
enabled = ["generic", "npm", "docker", "go", "maven"]

[limits]
max_git_concurrency = 4
max_webhook_concurrency = 8
max_ci_concurrency = 1
mem_limit_mb = 256

[limits.packages]
max_concurrent = 4
generic_max_file_mb = 200
npm_max_tarball_mb = 50
npm_max_metadata_kb = 1024
docker_max_blob_mb = 500
docker_max_manifest_kb = 256
docker_upload_ttl_minutes = 60
go_max_zip_mb = 100
maven_max_artifact_mb = 100

[log]
level = "info"
format = "text"

[mail]
enabled = false
host = "smtp.example.com"
port = 587
username = ""
password = ""
from = "noreply@example.com"
tls = true
`
