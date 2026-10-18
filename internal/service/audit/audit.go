// SPDX-License-Identifier: AGPL-3.0
// Package audit records security-relevant audit events with an async writer,
// querying, export, statistics, retention/archive and alert rules.
package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/LiStudioorg/gitli/internal/domain"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

// Result values for an audit event.
const (
	ResultSuccess = "success"
	ResultFailure = "failure"
	ResultDenied  = "denied"
)

// Entry describes an audit record. Event should follow the "resource.action"
// convention (e.g. "repo.delete", "auth.login.success").
type Entry struct {
	ActorID    *int64
	ActorType  string // user | system | admin | runner
	Action     string // legacy short action; derived from Event when empty
	TargetType string
	TargetID   *int64
	TargetName string
	IP         string
	Detail     string

	// Enterprise fields.
	Event        string
	Resource     string
	OldValue     string
	NewValue     string
	Result       string
	ErrorMessage string
	RequestID    string
	UserAgent    string
	Metadata     string
}

// Service writes and queries audit logs.
type Service struct {
	st    *store.Store
	log   *slog.Logger
	queue chan Entry
	wg    sync.WaitGroup
	stop  chan struct{}
	once  sync.Once
}

// New builds an audit Service with an async writer.
func New(st *store.Store) *Service {
	return &Service{
		st:    st,
		log:   slog.Default(),
		queue: make(chan Entry, 1024),
		stop:  make(chan struct{}),
	}
}

// SetLogger sets the logger used for write failures.
func (s *Service) SetLogger(l *slog.Logger) {
	if l != nil {
		s.log = l
	}
}

// Start launches the async writer goroutines.
func (s *Service) Start() {
	for i := 0; i < 2; i++ {
		s.wg.Add(1)
		go s.worker()
	}
}

// Stop drains and stops the async writer.
func (s *Service) Stop() {
	s.once.Do(func() { close(s.stop) })
	s.wg.Wait()
}

func (s *Service) worker() {
	defer s.wg.Done()
	for {
		select {
		case <-s.stop:
			return
		case e := <-s.queue:
			if err := s.write(context.Background(), e); err != nil {
				s.log.Warn("audit write failed", "event", e.Event, "error", err)
			}
		}
	}
}

// Log enqueues an audit event for async persistence. It never blocks the
// caller for long: if the buffer is full it falls back to a synchronous write.
func (s *Service) Log(e Entry) {
	select {
	case s.queue <- e:
	default:
		if err := s.write(context.Background(), e); err != nil {
			s.log.Warn("audit write (fallback) failed", "event", e.Event, "error", err)
		}
	}
}

// Record is the synchronous, backwards-compatible write used by existing
// callers. New code should prefer Log for non-blocking writes.
func (s *Service) Record(ctx context.Context, e Entry) error {
	return s.write(ctx, e)
}

func (s *Service) write(ctx context.Context, e Entry) error {
	if e.Result == "" {
		e.Result = ResultSuccess
	}
	if e.ActorType == "" {
		e.ActorType = "user"
	}
	if e.Event == "" {
		e.Event = e.Action
	}
	if e.Action == "" {
		e.Action = e.Event
	}
	if e.Resource == "" {
		e.Resource = strings.SplitN(e.Event, ".", 2)[0]
	}
	q := s.st.Rebind(`INSERT INTO audit_logs
		(actor_id, actor_type, actor_ip, actor_user_agent, event, resource, action,
		 target_type, target_id, target_name, old_value, new_value, result, error_message,
		 request_id, metadata, ip, detail)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	_, err := s.st.DB.ExecContext(ctx, q,
		e.ActorID, e.ActorType, e.IP, e.UserAgent, e.Event, e.Resource, e.Action,
		e.TargetType, e.TargetID, e.TargetName, e.OldValue, e.NewValue, e.Result, e.ErrorMessage,
		e.RequestID, e.Metadata, e.IP, e.Detail)
	return err
}

// List returns recent audit logs (legacy signature).
func (s *Service) List(ctx context.Context, limit, offset int) ([]*domain.AuditLog, error) {
	return s.Query(ctx, Query{Limit: limit, Offset: offset})
}

// Query filters audit logs.
type Query struct {
	Event   string
	ActorID *int64
	Target  string
	Result  string
	From    *time.Time
	To      *time.Time
	Limit   int
	Offset  int
}

// Query returns audit logs matching the filter, newest first.
func (s *Service) Query(ctx context.Context, f Query) ([]*domain.AuditLog, error) {
	if f.Limit <= 0 || f.Limit > 200 {
		f.Limit = 50
	}
	where := []string{"1=1"}
	args := []any{}
	if f.Event != "" {
		where = append(where, `event LIKE ? ESCAPE '\'`)
		args = append(args, store.PrefixPattern(f.Event))
	}
	if f.ActorID != nil {
		where = append(where, "actor_id = ?")
		args = append(args, *f.ActorID)
	}
	if f.Target != "" {
		where = append(where, `(target_type = ? OR target_name LIKE ? ESCAPE '\')`)
		args = append(args, f.Target, store.ContainsPattern(f.Target))
	}
	if f.Result != "" {
		where = append(where, "result = ?")
		args = append(args, f.Result)
	}
	if f.From != nil {
		where = append(where, "created_at >= ?")
		args = append(args, dbTime(*f.From))
	}
	if f.To != nil {
		where = append(where, "created_at <= ?")
		args = append(args, dbTime(*f.To))
	}
	args = append(args, f.Limit, f.Offset)
	q := s.st.Rebind(`SELECT id, actor_id, actor_type, actor_ip, event, resource, action,
		target_type, target_id, target_name, old_value, new_value, result, error_message,
		request_id, metadata, actor_user_agent, created_at
		FROM audit_logs WHERE ` + strings.Join(where, " AND ") + ` ORDER BY id DESC LIMIT ? OFFSET ?`)
	rows, err := s.st.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanLogs(rows)
}

// Get returns a single audit log by id.
func (s *Service) Get(ctx context.Context, id int64) (*domain.AuditLog, error) {
	q := s.st.Rebind(`SELECT id, actor_id, actor_type, actor_ip, event, resource, action,
		target_type, target_id, target_name, old_value, new_value, result, error_message,
		request_id, metadata, actor_user_agent, created_at
		FROM audit_logs WHERE id = ?`)
	rows, err := s.st.DB.QueryContext(ctx, q, id)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	logs, err := scanLogs(rows)
	if err != nil {
		return nil, err
	}
	if len(logs) == 0 {
		return nil, domain.ErrAuditNotFound
	}
	return logs[0], nil
}

func scanLogs(rows *sql.Rows) ([]*domain.AuditLog, error) {
	var out []*domain.AuditLog
	for rows.Next() {
		var a domain.AuditLog
		var actorID, targetID sql.NullInt64
		if err := rows.Scan(&a.ID, &actorID, &a.ActorType, &a.ActorIP, &a.Event, &a.Resource, &a.Action,
			&a.TargetType, &targetID, &a.TargetName, &a.OldValue, &a.NewValue, &a.Result, &a.ErrorMessage,
			&a.RequestID, &a.Metadata, &a.ActorUserAgent, &a.CreatedAt); err != nil {
			return nil, err
		}
		if actorID.Valid {
			v := actorID.Int64
			a.ActorID = &v
		}
		if targetID.Valid {
			v := targetID.Int64
			a.TargetID = &v
		}
		out = append(out, &a)
	}
	return out, rows.Err()
}

// Stream writes matching audit logs as CSV or JSON to the provided writer,
// row by row (no full in-memory materialization).
func (s *Service) Stream(ctx context.Context, f Query, format string, w interface{ Write([]byte) (int, error) }) (int, error) {
	f.Limit = 0 // unbounded export; pagination handled by chunked reads
	if format == "json" {
		return s.streamJSON(ctx, f, w)
	}
	return s.streamCSV(ctx, f, w)
}

func (s *Service) streamCSV(ctx context.Context, f Query, w interface{ Write([]byte) (int, error) }) (int, error) {
	header := "id,created_at,actor_id,actor_type,event,resource,action,target_type,target_id,result,actor_ip\n"
	if _, err := w.Write([]byte(header)); err != nil {
		return 0, err
	}
	count := 0
	offset := 0
	for {
		f.Offset = offset
		f.Limit = 500
		logs, err := s.Query(ctx, f)
		if err != nil {
			return count, err
		}
		if len(logs) == 0 {
			break
		}
		for _, a := range logs {
			line := fmt.Sprintf("%d,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s\n",
				a.ID, a.CreatedAt.UTC().Format(time.RFC3339), actorStr(a.ActorID), a.ActorType,
				a.Event, a.Resource, a.Action, a.TargetType, targetStr(a.TargetID), a.Result, a.ActorIP)
			if _, err := w.Write([]byte(line)); err != nil {
				return count, err
			}
			count++
		}
		offset += len(logs)
	}
	return count, nil
}

func (s *Service) streamJSON(ctx context.Context, f Query, w interface{ Write([]byte) (int, error) }) (int, error) {
	if _, err := w.Write([]byte("[\n")); err != nil {
		return 0, err
	}
	count := 0
	offset := 0
	first := true
	for {
		f.Offset = offset
		f.Limit = 500
		logs, err := s.Query(ctx, f)
		if err != nil {
			return count, err
		}
		if len(logs) == 0 {
			break
		}
		for _, a := range logs {
			b, _ := json.Marshal(a)
			if !first {
				if _, err := w.Write([]byte(",\n")); err != nil {
					return count, err
				}
			}
			first = false
			if _, err := w.Write(b); err != nil {
				return count, err
			}
			count++
		}
		offset += len(logs)
	}
	if _, err := w.Write([]byte("\n]\n")); err != nil {
		return count, err
	}
	return count, nil
}

func actorStr(id *int64) string {
	if id == nil {
		return ""
	}
	return fmt.Sprintf("%d", *id)
}

func targetStr(id *int64) string {
	if id == nil {
		return ""
	}
	return fmt.Sprintf("%d", *id)
}

// dbTime formats a timestamp to match the SQLite/MySQL DATETIME text format
// (UTC, "2006-01-02 15:04:05") so string comparison against created_at works
// consistently across drivers.
func dbTime(t time.Time) string {
	return t.UTC().Format("2006-01-02 15:04:05")
}
