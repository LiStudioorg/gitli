package git

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gitli/internal/db"

	"golang.org/x/crypto/ssh"
)

// AuthorizeFunc 判断 user（可为 nil，即匿名/未识别公钥）能否对 repo 做读（write=false）或写操作。
type AuthorizeFunc func(repo db.Repo, user *db.User, write bool) bool

// SSHServer 内置 Git SSH server：公钥认证 → spawn git upload-pack/receive-pack。
type SSHServer struct {
	addr      string
	reposDir  string
	q         db.Querier
	authorize AuthorizeFunc
}

func NewSSHServer(addr, reposDir string, q db.Querier, authorize AuthorizeFunc) *SSHServer {
	return &SSHServer{addr: addr, reposDir: reposDir, q: q, authorize: authorize}
}

func (s *SSHServer) ListenAndServe() error {
	hostKey, err := loadOrCreateHostKey(s.reposDir)
	if err != nil {
		return fmt.Errorf("ssh host key: %w", err)
	}
	cfg := &ssh.ServerConfig{
		PublicKeyCallback: func(meta ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			fp := ssh.FingerprintSHA256(key)
			ctx := context.Background()
			row, err := s.q.GetSSHKeyByFingerprint(ctx, fp)
			if err != nil {
				return nil, fmt.Errorf("unknown public key")
			}
			user, err := s.q.GetUserByID(ctx, row.UserID)
			if err != nil {
				return nil, fmt.Errorf("unknown public key")
			}
			return &ssh.Permissions{Extensions: map[string]string{"user_id": fmt.Sprint(user.ID)}}, nil
		},
	}
	cfg.AddHostKey(hostKey)

	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("ssh listen %s: %w", s.addr, err)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			return fmt.Errorf("ssh accept: %w", err)
		}
		go s.handleConn(cfg, conn)
	}
}

// loadOrCreateHostKey 加载（或首次生成并持久化）SSH host key，路径 <reposDir>/../ssh_host_ed25519_key。
func loadOrCreateHostKey(reposDir string) (ssh.Signer, error) {
	dataDir := filepath.Dir(reposDir)
	keyPath := filepath.Join(dataDir, "ssh_host_ed25519_key")
	if _, err := os.Stat(keyPath); err != nil {
		_, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("generate host key: %w", err)
		}
		block, err := x509.MarshalPKCS8PrivateKey(priv)
		if err != nil {
			return nil, fmt.Errorf("marshal host key: %w", err)
		}
		pemData := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: block})
		if err := os.WriteFile(keyPath, pemData, 0o600); err != nil {
			return nil, fmt.Errorf("save host key: %w", err)
		}
	}
	pemData, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("read host key: %w", err)
	}
	key, err := ssh.ParsePrivateKey(pemData)
	if err != nil {
		return nil, fmt.Errorf("parse host key: %w", err)
	}
	return key, nil
}

func (s *SSHServer) handleConn(cfg *ssh.ServerConfig, conn net.Conn) {
	defer conn.Close()
	sconn, chans, reqs, err := ssh.NewServerConn(conn, cfg)
	if err != nil {
		slog.Debug("ssh handshake failed", "err", err)
		return
	}
	defer sconn.Close()

	var user *db.User
	if idStr, ok := sconn.Permissions.Extensions["user_id"]; ok {
		var id int64
		if _, err := fmt.Sscanf(idStr, "%d", &id); err == nil {
			if u, err := s.q.GetUserByID(context.Background(), id); err == nil {
				user = &u
			}
		}
	}

	go ssh.DiscardRequests(reqs)
	for newCh := range chans {
		if newCh.ChannelType() != "session" {
			newCh.Reject(ssh.UnknownChannelType, "unsupported channel type")
			continue
		}
		ch, chReqs, err := newCh.Accept()
		if err != nil {
			continue
		}
		go s.handleSession(ch, chReqs, user)
	}
}

func (s *SSHServer) handleSession(ch ssh.Channel, chReqs <-chan *ssh.Request, user *db.User) {
	defer ch.Close()
	for req := range chReqs {
		switch req.Type {
		case "exec":
			if len(req.Payload) >= 4 {
				cmdLine := string(req.Payload[4:])
				if service, repoPath, err := parseGitCommand(cmdLine); err == nil {
					req.Reply(true, nil)
					s.runGitCommand(ch, service, repoPath, user)
					return
				}
			}
			req.Reply(false, nil)
		case "pty-req", "shell", "env":
			req.Reply(false, nil)
		default:
			if req.WantReply {
				req.Reply(false, nil)
			}
		}
	}
}

// parseGitCommand 严格解析 "git-upload-pack '/owner/name.git'" 形式的命令行。
func parseGitCommand(cmdLine string) (service, repoPath string, err error) {
	parts := strings.SplitN(cmdLine, " ", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("bad command")
	}
	service = parts[0]
	if service != "git-upload-pack" && service != "git-receive-pack" {
		return "", "", fmt.Errorf("unsupported service")
	}
	repoPath = strings.TrimSpace(parts[1])
	repoPath = strings.Trim(repoPath, "'\"")
	repoPath = strings.TrimPrefix(repoPath, "/")
	if repoPath == "" {
		return "", "", fmt.Errorf("empty repo path")
	}
	return service, repoPath, nil
}

func (s *SSHServer) runGitCommand(ch ssh.Channel, service, repoPath string, user *db.User) bool {
	owner, name, err := splitRepoPath(repoPath)
	if err != nil {
		fmt.Fprintf(ch.Stderr(), "gitli: %v\r\n", err)
		return false
	}
	ctx := context.Background()
	repo, err := s.q.GetRepoByOwnerAndName(ctx, db.GetRepoByOwnerAndNameParams{Username: owner, Name: name})
	if err != nil {
		fmt.Fprintf(ch.Stderr(), "gitli: repository not found\r\n")
		return false
	}
	write := service == "git-receive-pack"
	if s.authorize == nil || !s.authorize(repo, user, write) {
		fmt.Fprintf(ch.Stderr(), "gitli: access denied\r\n")
		return false
	}

	path := RepoPath(s.reposDir, owner, name)
	cmd := exec.Command("git", service[len("git-"):], path)
	cmd.Stdout = ch
	cmd.Stderr = ch.Stderr()
	stdin, err := cmd.StdinPipe()
	if err != nil {
		sendExitStatus(ch, fmt.Errorf("stdin pipe"))
		return false
	}
	if err := cmd.Start(); err != nil {
		sendExitStatus(ch, err)
		return false
	}
	go func() {
		io.Copy(stdin, ch)
		stdin.Close()
	}()
	err = cmd.Wait()
	sendExitStatus(ch, err)
	return true
}

// splitRepoPath 校验并拆分 "owner/name.git"。
func splitRepoPath(repoPath string) (owner, name string, err error) {
	idx := strings.Index(repoPath, "/")
	if idx < 0 {
		return "", "", errors.New("invalid repo path")
	}
	owner = repoPath[:idx]
	name = strings.TrimSuffix(repoPath[idx+1:], ".git")
	if err := ValidateUsername(owner); err != nil {
		return "", "", fmt.Errorf("invalid repo path")
	}
	if err := ValidateRepoName(name); err != nil {
		return "", "", fmt.Errorf("invalid repo path")
	}
	return owner, name, nil
}

func sendExitStatus(ch ssh.Channel, err error) {
	status := uint32(0)
	if err != nil {
		status = 1
	}
	payload := make([]byte, 4)
	payload[0] = byte(status >> 24)
	payload[1] = byte(status >> 16)
	payload[2] = byte(status >> 8)
	payload[3] = byte(status)
	_, _ = ch.SendRequest("exit-status", false, payload)
}

var _ io.Writer = ssh.Channel(nil)
