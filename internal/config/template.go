// SPDX-License-Identifier: AGPL-3.0
package config

// defaultTemplate is written to disk on first run when config.toml is absent.
const defaultTemplate = `# gitli 配置文件（首次运行自动生成）
# 环境变量优先级高于此文件，前缀 GITLI_。

[server]
http_listen = ["0.0.0.0:8080", "[::]:8080"]
http_external_url = ""
ssh_enabled = true
ssh_listen = ["0.0.0.0:2222", "[::]:2222"]
ssh_external = ""
allow_root = false

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
min_length = 12
hibp = false
pepper = ""

[security.crypto]
key_hex = ""

[security.login]
max_attempts = 5
lockout_minutes = 15
window_minutes = 15

[security.session]
bind_ip = false
bind_agent = false
ttl_hours = 720

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

[git]
bin = ""
timeout_seconds = 300
check_version = true
min_version = "2.30"

[ssh]
mode = "builtin"
host_key_path = "./data/ssh_host_ed25519_key"
handshake_timeout_seconds = 10
idle_timeout_seconds = 300
max_connections = 32
`
