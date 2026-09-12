package wiki

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	ErrPageNotFound = errors.New("page not found")
	ErrInvalidPage  = errors.New("invalid page name")
)

var pageRe = regexp.MustCompile(`^[A-Za-z0-9_-]{1,50}$`)

// ValidatePageName 校验页面名：^[A-Za-z0-9_-]{1,50}$。
func ValidatePageName(page string) error {
	if !pageRe.MatchString(page) {
		return ErrInvalidPage
	}
	return nil
}

// WikiPath 返回 wiki 裸仓库路径：<reposDir>/<owner>/<name>.wiki.git。
func WikiPath(reposDir, owner, repoName string) string {
	return filepath.Join(reposDir, owner, repoName+".wiki.git")
}

// Ensure wiki 裸仓库不存在则初始化（HEAD 指向 main）。
func Ensure(ctx context.Context, wikiPath string) error {
	if _, err := os.Stat(filepath.Join(wikiPath, "HEAD")); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(wikiPath), 0o755); err != nil {
		return fmt.Errorf("create wiki dir: %w", err)
	}
	if err := run(ctx, "", "init", "--bare", wikiPath); err != nil {
		return fmt.Errorf("git init wiki: %w", err)
	}
	if err := run(ctx, wikiPath, "symbolic-ref", "HEAD", "refs/heads/main"); err != nil {
		return fmt.Errorf("set wiki HEAD: %w", err)
	}
	return nil
}

// SavePage 保存（创建或更新）页面。用纯 plumbing 序列：
// hash-object → read-tree(容忍空仓库) → update-index → write-tree → commit-tree → update-ref。
func SavePage(ctx context.Context, wikiPath, page string, content []byte, msg string) error {
	if err := ValidatePageName(page); err != nil {
		return err
	}
	if msg == "" {
		msg = "update " + page
	}
	if err := Ensure(ctx, wikiPath); err != nil {
		return err
	}
	target := page + ".md"

	// a. blob
	blobOut, err := runStdin(ctx, wikiPath, content, "hash-object", "-w", "--stdin")
	if err != nil {
		return fmt.Errorf("hash-object: %w", err)
	}
	blobSHA := strings.TrimSpace(string(blobOut))

	// b. read-tree HEAD（空仓库容忍失败）
	hasOldHead := true
	idxFile := filepath.Join(os.TempDir(), fmt.Sprintf("gitli-wiki-idx-%d", os.Getpid()))
	defer os.Remove(idxFile)
	env := append(os.Environ(), "GIT_INDEX_FILE="+idxFile)
	if _, err := runEnv(ctx, wikiPath, env, "read-tree", "HEAD"); err != nil {
		hasOldHead = false
	}

	// c. update-index
	if _, err := runEnv(ctx, wikiPath, env, "update-index", "--add",
		"--cacheinfo", "100644,"+blobSHA+","+target); err != nil {
		return fmt.Errorf("update-index: %w", err)
	}

	// d. write-tree
	treeOut, err := runEnv(ctx, wikiPath, env, "write-tree")
	if err != nil {
		return fmt.Errorf("write-tree: %w", err)
	}
	treeSHA := strings.TrimSpace(string(treeOut))

	// e. commit-tree
	args := []string{"commit-tree", treeSHA}
	if hasOldHead {
		args = append(args, "-p", "HEAD")
	}
	args = append(args, "-m", msg)
	commitOut, err := runEnv(ctx, wikiPath, env, args...)
	if err != nil {
		return fmt.Errorf("commit-tree: %w", err)
	}
	newCommit := strings.TrimSpace(string(commitOut))

	// f. update-ref
	if err := run(ctx, wikiPath, "update-ref", "refs/heads/main", newCommit); err != nil {
		return fmt.Errorf("update-ref main: %w", err)
	}
	return nil
}

// GetPage 读取页面 markdown 内容。
func GetPage(ctx context.Context, wikiPath, page string) ([]byte, error) {
	if err := ValidatePageName(page); err != nil {
		return nil, err
	}
	out, err := runOut(ctx, wikiPath, "show", "main:"+page+".md")
	if err != nil {
		return nil, ErrPageNotFound
	}
	return out, nil
}

// ListPages 列出 wiki 所有页面（去 .md 后缀）。
func ListPages(ctx context.Context, wikiPath string) ([]string, error) {
	if _, err := os.Stat(filepath.Join(wikiPath, "HEAD")); err != nil {
		return []string{}, nil
	}
	out, err := runOut(ctx, wikiPath, "ls-tree", "--name-only", "main")
	if err != nil {
		// 空仓库（main 不存在）
		return []string{}, nil
	}
	pages := []string{}
	for _, name := range strings.Split(string(out), "\n") {
		name = strings.TrimSpace(name)
		if strings.HasSuffix(name, ".md") {
			pages = append(pages, strings.TrimSuffix(name, ".md"))
		}
	}
	return pages, nil
}

// HasPage 判断页面是否存在。
func HasPage(ctx context.Context, wikiPath, page string) bool {
	_, err := GetPage(ctx, wikiPath, page)
	return err == nil
}

func gitCmd(ctx context.Context, dir string, args []string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	return cmd
}

func run(ctx context.Context, dir string, args ...string) error {
	cmd := gitCmd(ctx, dir, args)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func runOut(ctx context.Context, dir string, args ...string) ([]byte, error) {
	cmd := gitCmd(ctx, dir, args)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return bytes.TrimSuffix(stdout.Bytes(), []byte("\n")), nil
}

func runEnv(ctx context.Context, dir string, env []string, args ...string) ([]byte, error) {
	cmd := gitCmd(ctx, dir, args)
	cmd.Env = env
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return bytes.TrimSuffix(stdout.Bytes(), []byte("\n")), nil
}

func runStdin(ctx context.Context, dir string, stdin []byte, args ...string) ([]byte, error) {
	cmd := gitCmd(ctx, dir, args)
	cmd.Stdin = bytes.NewReader(stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return bytes.TrimSuffix(stdout.Bytes(), []byte("\n")), nil
}
