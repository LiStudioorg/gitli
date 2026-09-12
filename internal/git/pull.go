package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
)

var ErrMergeConflict = errors.New("merge conflict")

// MergeAvailable 判断 base 与 head 是否有共同祖先（可比较/合并）。
func MergeAvailable(ctx context.Context, repoPath, base, head string) bool {
	_, err := runOut(ctx, repoPath, "merge-base", base, head)
	return err == nil
}

// MergeBase 返回 base 与 head 的 merge-base sha。
func MergeBase(ctx context.Context, repoPath, base, head string) (string, error) {
	out, err := runOut(ctx, repoPath, "merge-base", base, head)
	if err != nil {
		return "", fmt.Errorf("merge-base: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// LogCommitsRange 列出 from..to 范围内的提交。
func LogCommitsRange(ctx context.Context, repoPath, from, to string) ([]Commit, error) {
	if err := ValidateRef(to); err != nil {
		return nil, err
	}
	out, err := runOut(ctx, repoPath, "log", "--format=%H%x00%h%x00%an%x00%at%x00%s", from+".."+to)
	if err != nil {
		return nil, fmt.Errorf("git log %s..%s: %w", from, to, err)
	}
	commits := []Commit{}
	if len(out) == 0 {
		return commits, nil
	}
	for _, line := range strings.Split(string(out), "\n") {
		parts := strings.SplitN(line, "\x00", 5)
		if len(parts) != 5 {
			continue
		}
		var ts int64
		fmt.Sscanf(parts[3], "%d", &ts)
		commits = append(commits, Commit{SHA: parts[0], Short: parts[1], Author: parts[2], Time: ts, Subject: parts[4]})
	}
	return commits, nil
}

// MergeBranch 在裸仓库中把 head 合入 base（不触碰工作区，纯 plumbing）。
// squash=false：merge-tree --write-tree 产生带双亲的合并提交；
// squash=true：只取 tree，提交仅带 base 单亲。
// 返回新 commit sha。
func MergeBranch(ctx context.Context, repoPath, base, head, msg string, squash bool) (string, error) {
	baseSHA, err := ResolveRef(ctx, repoPath, base)
	if err != nil {
		return "", fmt.Errorf("resolve base %q: %w", base, err)
	}
	headSHA, err := ResolveRef(ctx, repoPath, head)
	if err != nil {
		return "", fmt.Errorf("resolve head %q: %w", head, err)
	}

	var treeSHA string
	if !squash {
		// git merge-tree --write-tree <base> <head>：无冲突输出 tree oid
		out, err := runOut(ctx, repoPath, "merge-tree", "--write-tree", baseSHA, headSHA)
		if err != nil {
			return "", fmt.Errorf("%w: %v", ErrMergeConflict, err)
		}
		treeSHA = strings.TrimSpace(string(out))
		// 输出首行是 tree oid，冲突时后面还会带冲突文件列表，这里只取首行
		if i := strings.IndexByte(treeSHA, '\n'); i >= 0 {
			treeSHA = treeSHA[:i]
		}
		if len(treeSHA) != 40 {
			return "", fmt.Errorf("merge-tree: unexpected output %q", treeSHA)
		}
	} else {
		// squash：用 read-tree 方式拿合并 tree。merge-tree 同样适用（冲突也报错）
		out, err := runOut(ctx, repoPath, "merge-tree", "--write-tree", baseSHA, headSHA)
		if err != nil {
			return "", fmt.Errorf("%w: %v", ErrMergeConflict, err)
		}
		treeSHA = strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
		if len(treeSHA) != 40 {
			return "", fmt.Errorf("merge-tree: unexpected output %q", treeSHA)
		}
	}

	args := []string{"commit-tree", treeSHA, "-p", baseSHA}
	if !squash {
		args = append(args, "-p", headSHA)
	}
	args = append(args, "-m", msg)
	out, err := runOut(ctx, repoPath, args...)
	if err != nil {
		return "", fmt.Errorf("commit-tree: %w", err)
	}
	newCommit := strings.TrimSpace(string(out))
	if err := run(ctx, repoPath, "update-ref", "refs/heads/"+base, newCommit); err != nil {
		return "", fmt.Errorf("update-ref %s: %w", base, err)
	}
	return newCommit, nil
}

// RebaseBranch 把 head 变基到 base 之上：临时 worktree 中执行
// git rebase --onto <base>，成功后 update-ref head 分支。
// 返回 rebase 后 head 分支新 sha。
func RebaseBranch(ctx context.Context, repoPath, base, head string) (string, error) {
	baseSHA, err := ResolveRef(ctx, repoPath, base)
	if err != nil {
		return "", fmt.Errorf("resolve base %q: %w", base, err)
	}
	headSHA, err := ResolveRef(ctx, repoPath, head)
	if err != nil {
		return "", fmt.Errorf("resolve head %q: %w", head, err)
	}
	// 检查 base 是否已是 head 祖先（否则会分叉冲突）
	out, err := runOut(ctx, repoPath, "merge-base", "--is-ancestor", baseSHA, headSHA)
	_ = out
	if err != nil {
		return "", fmt.Errorf("%w: base is not ancestor of head", ErrMergeConflict)
	}

	tmp, err := worktreeAdd(ctx, repoPath, head)
	if err != nil {
		return "", err
	}
	defer worktreeRemove(ctx, repoPath, tmp)

	if err := run(ctx, tmp, "rebase", baseSHA); err != nil {
		// rebase 冲突时中止，避免留下进行中的状态
		_ = run(ctx, tmp, "rebase", "--abort")
		return "", fmt.Errorf("%w: rebase failed", ErrMergeConflict)
	}
	newHead, err := runOut(ctx, tmp, "rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("rev-parse HEAD: %w", err)
	}
	sha := strings.TrimSpace(string(newHead))
	if err := run(ctx, repoPath, "update-ref", "refs/heads/"+head, sha); err != nil {
		return "", fmt.Errorf("update-ref %s: %w", head, err)
	}
	return sha, nil
}

func worktreeAdd(ctx context.Context, repoPath, branch string) (string, error) {
	dir := repoPath + ".rebase-tmp"
	if err := run(ctx, repoPath, "worktree", "add", "--detach", dir, branch); err != nil {
		return "", fmt.Errorf("worktree add: %w", err)
	}
	return dir, nil
}

func worktreeRemove(ctx context.Context, repoPath, dir string) {
	_ = run(ctx, repoPath, "worktree", "remove", "--force", dir)
}

// DiffRange 输出 base...head 的 diff 文本。
func DiffRange(ctx context.Context, repoPath, base, head string) ([]byte, error) {
	if err := ValidateRef(base); err != nil {
		return nil, err
	}
	if err := ValidateRef(head); err != nil {
		return nil, err
	}
	out, err := runOut(ctx, repoPath, "diff", "--no-color", base+"..."+head)
	if err != nil {
		return nil, fmt.Errorf("git diff %s...%s: %w", base, head, err)
	}
	return bytes.TrimSuffix(out, []byte("\n")), nil
}
