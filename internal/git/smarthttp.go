package git

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

// Service 表示 smart HTTP 的 git 服务名。
type Service string

const (
	UploadPack  Service = "git-upload-pack"
	ReceivePack Service = "git-receive-pack"
)

// SmartHTTPHandler 桥接 Git smart HTTP 协议到 `git http-backend`（CGI）。
// auth 结果由调用方决定：handler 只负责协议转发。
type SmartHTTPHandler struct {
	// RelRoot 是 GIT_PROJECT_ROOT：所有裸仓库的公共父目录（如 <data>/repos）。
	RelRoot string
	// RepoPath 返回 (仓库磁盘路径, 相对 RelRoot 的路径)；ok=false 时返回 404。
	RepoPath func(r *http.Request) (path string, relPath string, ok bool)
	// Authorize 对请求做读写鉴权；返回 false 时响应 401。
	Authorize func(r *http.Request, svc Service) bool
}

func (h *SmartHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	svc, err := serviceFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !h.Authorize(r, svc) {
		w.Header().Set("WWW-Authenticate", `Basic realm="gitli"`)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	repoPath, relPath, ok := h.RepoPath(r)
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	// PATH_INFO = /<owner>/<repo>.git/...，相对于 GIT_PROJECT_ROOT
	pi := "/" + strings.TrimPrefix(strings.TrimSuffix(relPath, "/")+gitPathSuffix(r.URL.Path), "/")

	cgi := exec.Command("git", "http-backend")
	cgi.Dir = repoPath
	// 继承 PATH 等基础环境，再叠加 CGI 变量（http-backend 需要在 PATH 里找到 git 子命令）
	cgi.Env = append(os.Environ(),
		"GIT_PROJECT_ROOT="+h.RelRoot,
		"GIT_HTTP_EXPORT_ALL=1",
		"PATH_INFO="+pi,
		"QUERY_STRING="+r.URL.RawQuery,
		"REQUEST_METHOD="+r.Method,
		"CONTENT_TYPE="+r.Header.Get("Content-Type"),
		"REMOTE_ADDR="+r.RemoteAddr,
	)
	if u, p, ok := r.BasicAuth(); ok {
		// 供 hooks 日志参考；鉴权已在上游完成
		cgi.Env = append(cgi.Env, "REMOTE_USER="+u, "REMOTE_USER_PASS="+p)
	}

	var stderr bytes.Buffer
	cgi.Stderr = &stderr

	stdin, err := cgi.StdinPipe()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	stdout, err := cgi.StdoutPipe()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := cgi.Start(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// 请求体（push 可能很大）流式写入
	if r.Body != nil && (r.Method == http.MethodPost) {
		_, _ = io.Copy(stdin, r.Body)
	}
	stdin.Close()

	// 手动解析 CGI 输出：CGI 头（直到空行）+ body 流式转发
	reader := bufio.NewReader(stdout)
	headerOK := false
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		trimmed := strings.TrimRight(line, "\r\n")
		if trimmed == "" {
			headerOK = true
			break
		}
		if idx := strings.IndexByte(trimmed, ':'); idx > 0 {
			key := strings.TrimSpace(trimmed[:idx])
			val := strings.TrimSpace(trimmed[idx+1:])
			switch strings.ToLower(key) {
			case "status":
				if code := parseStatus(val); code > 0 {
					w.WriteHeader(code)
				}
			default:
				w.Header().Add(key, val)
			}
		}
	}
	if !headerOK {
		_ = cgi.Wait()
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, reader)
	_ = cgi.Wait()
	if s := stderr.String(); s != "" {
		println("git http-backend stderr: " + s)
	}
}

// gitPathSuffix 返回 URL 中 .git 后面的部分（如 /info/refs 或 /git-upload-pack）。
func gitPathSuffix(urlPath string) string {
	i := strings.Index(urlPath, ".git")
	if i < 0 {
		return urlPath
	}
	return urlPath[i+len(".git"):]
}

// parseStatus 从 CGI "Status: 404 Not Found" 头解析状态码。
func parseStatus(val string) int {
	fields := strings.Fields(val)
	if len(fields) == 0 {
		return 0
	}
	code := 0
	for _, c := range fields[0] {
		if c < '0' || c > '9' {
			return 0
		}
		code = code*10 + int(c-'0')
	}
	if code < 100 || code > 599 {
		return 0
	}
	return code
}

// serviceFromRequest 从 URL 推断 git 服务：
// /info/refs?service=git-upload-pack 或 POST /git-upload-pack。
func serviceFromRequest(r *http.Request) (Service, error) {
	if strings.HasSuffix(r.URL.Path, "/info/refs") {
		s := r.URL.Query().Get("service")
		if s != string(UploadPack) && s != string(ReceivePack) {
			return "", fmt.Errorf("unsupported service %q", s)
		}
		return Service(s), nil
	}
	for _, s := range []Service{UploadPack, ReceivePack} {
		if strings.HasSuffix(r.URL.Path, "/"+string(s)) {
			if r.Method != http.MethodPost {
				return "", fmt.Errorf("method not allowed")
			}
			return s, nil
		}
	}
	return "", fmt.Errorf("not a git smart http endpoint")
}
