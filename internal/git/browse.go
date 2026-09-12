package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// TreeEntry 目录树条目。
type TreeEntry struct {
	Name string
	Mode string
	Type string // "blob" | "tree"
	SHA  string
	Size int64 // tree 为 -1
	Path string
}

// Commit 提交元数据。
type Commit struct {
	SHA     string
	Short   string
	Author  string
	Time    int64 // unix 秒
	Subject string
	Body    string
}

// BranchRef 分支/标签。
type BranchRef struct {
	Name string
	SHA  string
	Date int64
}

// BlameLine blame 单行结果。
type BlameLine struct {
	Line    int
	SHA     string // 短 sha（8 位）
	Author  string
	Content string
}

// DiffLine diff 单行（Kind 决定 CSS 类）。
type DiffLine struct {
	Kind string // "meta" | "hunk" | "add" | "del" | "ctx"
	Text string
}

// refRe 安全 ref 字符集：字母数字与 _ . -（不含 / 的单段 ref，如分支名、tag、sha）。
var refRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)

// ValidateRef 校验 URL 中的 ref（分支/tag/短 sha），防止参数注入与路径穿越。
func ValidateRef(ref string) error {
	if ref == "" || len(ref) > 200 {
		return fmt.Errorf("invalid ref length")
	}
	if !refRe.MatchString(ref) {
		return fmt.Errorf("invalid characters in ref")
	}
	if strings.Contains(ref, "..") {
		return fmt.Errorf("ref must not contain '..'")
	}
	if strings.HasSuffix(ref, ".lock") {
		return fmt.Errorf("reserved ref name")
	}
	return nil
}

// ValidatePath 校验仓库内文件路径：不以 / 开头、不含 ".."、每段合法。
func ValidatePath(path string) error {
	if path == "" {
		return nil
	}
	if strings.HasPrefix(path, "/") {
		return fmt.Errorf("path must not start with '/'")
	}
	clean := filepath.Clean(path)
	if clean == "." || strings.HasPrefix(clean, "..") {
		return fmt.Errorf("path must not contain '..'")
	}
	for _, seg := range strings.Split(clean, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return fmt.Errorf("invalid path segment")
		}
	}
	return nil
}

// runOut 执行 git 命令并返回 stdout（参数数组，cmd.Dir=repoPath）。
func runOut(ctx context.Context, dir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return []byte(strings.TrimSuffix(stdout.String(), "\n")), nil
}

// HasCommit 判断仓库是否有任何提交（HEAD 可解析）。
func HasCommit(ctx context.Context, repoPath string) bool {
	_, err := runOut(ctx, repoPath, "rev-parse", "--verify", "HEAD")
	return err == nil
}

// DefaultBranch 返回 HEAD 指向的分支名（如 main），失败返回 "main"。
func DefaultBranch(ctx context.Context, repoPath string) string {
	out, err := runOut(ctx, repoPath, "symbolic-ref", "--short", "HEAD")
	if err != nil || len(out) == 0 {
		return "main"
	}
	return string(out)
}

// ResolveRef 校验 ref 可解析为 commit，返回完整 sha。
func ResolveRef(ctx context.Context, repoPath, ref string) (string, error) {
	if err := ValidateRef(ref); err != nil {
		return "", err
	}
	out, err := runOut(ctx, repoPath, "rev-parse", "--verify", ref+"^{commit}")
	if err != nil {
		return "", fmt.Errorf("resolve ref %q: %w", ref, err)
	}
	return string(out), nil
}

// ParseLsTree 列出 ref 下 path 的目录树（path 为空列根；path 为目录时列其内容）。
// 返回错误时区分：ErrPathIsBlob 由调用方处理跳转。
func ParseLsTree(ctx context.Context, repoPath, ref, path string) ([]TreeEntry, error) {
	if err := ValidateRef(ref); err != nil {
		return nil, err
	}
	if err := ValidatePath(path); err != nil {
		return nil, err
	}
	if path != "" {
		// ref:path 形式列出目录内容；若 path 是 blob 会报错，回退到单条查询
		entries, err := lsTreeRaw(ctx, repoPath, ref+":"+path)
		if err == nil {
			return entries, nil
		}
		single, err2 := lsTreeRaw(ctx, repoPath, ref, "--", path)
		if err2 != nil {
			return nil, fmt.Errorf("ls-tree %s %s: %w", ref, path, err)
		}
		if len(single) == 1 && single[0].Type == "blob" {
			return single, nil
		}
		return nil, fmt.Errorf("ls-tree %s %s: not found", ref, path)
	}
	return lsTreeRaw(ctx, repoPath, ref)
}

func lsTreeRaw(ctx context.Context, repoPath string, args ...string) ([]TreeEntry, error) {
	full := append([]string{"ls-tree", "-l", "-z"}, args...)
	out, err := runOut(ctx, repoPath, full...)
	if err != nil {
		return nil, err
	}
	return parseLsTree(out), nil
}

// parseLsTree 解析 ls-tree -z 输出：<mode> SP <type> SP <sha> SP <size> TAB <path> NUL
func parseLsTree(out []byte) []TreeEntry {
	entries := []TreeEntry{}
	for _, rec := range bytes.Split(out, []byte{0}) {
		if len(rec) == 0 {
			continue
		}
		tab := bytes.IndexByte(rec, '\t')
		if tab < 0 {
			continue
		}
		meta := strings.Fields(string(rec[:tab]))
		if len(meta) < 4 {
			continue
		}
		size := int64(-1)
		if meta[3] != "-" {
			if n, err := strconv.ParseInt(meta[3], 10, 64); err == nil {
				size = n
			}
		}
		entries = append(entries, TreeEntry{
			Mode: meta[0],
			Type: meta[1],
			SHA:  meta[2],
			Size: size,
			Path: string(rec[tab+1:]),
			Name: filepath.Base(string(rec[tab+1:])),
		})
	}
	return entries
}

// ListBranches 列出本地分支。
func ListBranches(ctx context.Context, repoPath string) ([]BranchRef, error) {
	return listRefs(ctx, repoPath, "refs/heads")
}

// ListTags 列出标签。
func ListTags(ctx context.Context, repoPath string) ([]BranchRef, error) {
	return listRefs(ctx, repoPath, "refs/tags")
}

func listRefs(ctx context.Context, repoPath, prefix string) ([]BranchRef, error) {
	out, err := runOut(ctx, repoPath, "for-each-ref", prefix,
		"--format=%(refname:short)%00%(objectname)%00%(committerdate:unix)")
	if err != nil {
		return nil, fmt.Errorf("for-each-ref %s: %w", prefix, err)
	}
	refs := []BranchRef{}
	for _, line := range strings.Split(string(out), "\n") {
		parts := strings.SplitN(line, "\x00", 3)
		if len(parts) != 3 {
			continue
		}
		date, _ := strconv.ParseInt(parts[2], 10, 64)
		refs = append(refs, BranchRef{Name: parts[0], SHA: parts[1], Date: date})
	}
	return refs, nil
}

// LogCommits 列出 ref 的提交记录（limit 条，skip 跳过前 N 条用于分页）。
func LogCommits(ctx context.Context, repoPath, ref string, limit, skip int) ([]Commit, error) {
	if err := ValidateRef(ref); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 50
	}
	args := []string{"log", "-n", strconv.Itoa(limit), "--skip=" + strconv.Itoa(skip),
		"--format=%H%x00%h%x00%an%x00%at%x00%s"}
	args = append(args, ref)
	out, err := runOut(ctx, repoPath, args...)
	if err != nil {
		return nil, fmt.Errorf("git log %s: %w", ref, err)
	}
	commits := []Commit{}
	for _, line := range strings.Split(string(out), "\n") {
		parts := strings.SplitN(line, "\x00", 5)
		if len(parts) != 5 {
			continue
		}
		ts, _ := strconv.ParseInt(parts[3], 10, 64)
		commits = append(commits, Commit{
			SHA: parts[0], Short: parts[1], Author: parts[2], Time: ts, Subject: parts[4],
		})
	}
	return commits, nil
}

// GetCommit 取单个提交的元数据与 diff 补丁文本。
func GetCommit(ctx context.Context, repoPath, sha string) (Commit, []DiffLine, error) {
	meta, err := runOut(ctx, repoPath, "log", "-n", "1",
		"--format=%H%x00%h%x00%an%x00%at%x00%s%x00%b", sha)
	if err != nil {
		return Commit{}, nil, fmt.Errorf("git log %s: %w", sha, err)
	}
	var c Commit
	for _, line := range strings.Split(string(meta), "\n") {
		parts := strings.SplitN(line, "\x00", 6)
		if len(parts) != 6 {
			continue
		}
		ts, _ := strconv.ParseInt(parts[3], 10, 64)
		c = Commit{SHA: parts[0], Short: parts[1], Author: parts[2], Time: ts, Subject: parts[4], Body: parts[5]}
		break
	}
	if c.SHA == "" {
		return Commit{}, nil, fmt.Errorf("commit %s not found", sha)
	}

	patch, err := runOut(ctx, repoPath, "show", "--format=", "--no-color", sha)
	if err != nil {
		return c, nil, fmt.Errorf("git show %s: %w", sha, err)
	}
	return c, parseDiff(string(patch)), nil
}

// ParseDiffText 把 unified diff 文本分类为逐行结构（供 PR diff 视图复用）。
func ParseDiffText(text string) []DiffLine {
	return parseDiff(text)
}

// parseDiff 把 unified diff 文本分类为逐行结构（Text 为内容，转义交给模板）。
func parseDiff(text string) []DiffLine {
	const maxLines = 5000
	lines := []DiffLine{}
	for i, line := range strings.Split(text, "\n") {
		if i >= maxLines {
			lines = append(lines, DiffLine{Kind: "meta", Text: "-- diff 过长，已截断 --"})
			break
		}
		kind := "ctx"
		body := line
		switch {
		case strings.HasPrefix(line, "diff --git"), strings.HasPrefix(line, "index "),
			strings.HasPrefix(line, "--- "), strings.HasPrefix(line, "+++ "),
			strings.HasPrefix(line, "new file"), strings.HasPrefix(line, "deleted file"),
			strings.HasPrefix(line, "rename "), strings.HasPrefix(line, "old mode"),
			strings.HasPrefix(line, "new mode"), strings.HasPrefix(line, "similarity"),
			strings.HasPrefix(line, "Binary files"):
			kind, body = "meta", line
		case strings.HasPrefix(line, "@@"):
			kind, body = "hunk", line
		case strings.HasPrefix(line, "+"):
			kind, body = "add", strings.TrimPrefix(line, "+")
		case strings.HasPrefix(line, "-"):
			kind, body = "del", strings.TrimPrefix(line, "-")
		case strings.HasPrefix(line, " "):
			kind, body = "ctx", strings.TrimPrefix(line, " ")
		}
		lines = append(lines, DiffLine{Kind: kind, Text: body})
	}
	return lines
}

// GetBlob 读取 ref 下 path 的文件内容。
func GetBlob(ctx context.Context, repoPath, ref, path string) ([]byte, error) {
	if err := ValidateRef(ref); err != nil {
		return nil, err
	}
	if err := ValidatePath(path); err != nil {
		return nil, err
	}
	out, err := runOut(ctx, repoPath, "show", ref+":"+path)
	if err != nil {
		return nil, fmt.Errorf("git show %s:%s: %w", ref, path, err)
	}
	return out, nil
}

// IsBinary 前 8KB 含 NUL 字节即视为二进制。
func IsBinary(content []byte) bool {
	n := len(content)
	if n > 8192 {
		n = 8192
	}
	return bytes.IndexByte(content[:n], 0) >= 0
}

// Blame 解析 git blame --porcelain 输出为逐行结果。
func Blame(ctx context.Context, repoPath, ref, path string) ([]BlameLine, error) {
	if err := ValidateRef(ref); err != nil {
		return nil, err
	}
	if err := ValidatePath(path); err != nil {
		return nil, err
	}
	out, err := runOut(ctx, repoPath, "blame", "--porcelain", ref, "--", path)
	if err != nil {
		return nil, fmt.Errorf("git blame %s -- %s: %w", ref, path, err)
	}

	result := []BlameLine{}
	cur := struct{ sha, author string }{}
	lineNo := 0
	for _, line := range strings.Split(string(out), "\n") {
		switch {
		case strings.HasPrefix(line, "\t"):
			lineNo++
			result = append(result, BlameLine{
				Line:    lineNo,
				SHA:     shortSHA(cur.sha),
				Author:  cur.author,
				Content: strings.TrimPrefix(line, "\t"),
			})
		case line == "":
			// 记录分隔
		default:
			fields := strings.Fields(line)
			if len(fields) >= 2 && isHex(fields[0]) {
				cur = struct{ sha, author string }{sha: fields[0]}
				continue
			}
			if strings.HasPrefix(line, "author ") {
				cur.author = strings.TrimPrefix(line, "author ")
			}
		}
	}
	return result, nil
}

func shortSHA(sha string) string {
	if len(sha) > 8 {
		return sha[:8]
	}
	return sha
}

func isHex(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return len(s) >= 7
}
