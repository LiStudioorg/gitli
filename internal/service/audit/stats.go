// SPDX-License-Identifier: AGPL-3.0
package audit

import (
	"context"
	"time"
)

// DayCount is a per-day aggregation row.
type DayCount struct {
	Day   string `json:"day"`
	Count int    `json:"count"`
}

// KeyCount is a generic aggregation row keyed by a label.
type KeyCount struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
}

// Stats aggregates audit activity.
type Stats struct {
	ByDay   []DayCount `json:"by_day"`
	ByEvent []KeyCount `json:"by_event"`
	ByActor []KeyCount `json:"by_actor"`
}

// StatsSince computes aggregated statistics for events since the given time.
func (s *Service) StatsSince(ctx context.Context, since time.Time) (*Stats, error) {
	out := &Stats{}
	// By day.
	q := s.st.Rebind(`SELECT SUBSTR(CAST(created_at AS CHAR), 1, 10) AS d, COUNT(*) FROM audit_logs WHERE created_at >= ? GROUP BY d ORDER BY d`)
	rows, err := s.st.DB.QueryContext(ctx, q, dbTime(since))
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var d DayCount
		if err := rows.Scan(&d.Day, &d.Count); err != nil {
			_ = rows.Close()
			return nil, err
		}
		out.ByDay = append(out.ByDay, d)
	}
	_ = rows.Close()

	// By event.
	ev, err := s.groupCount(ctx, `SELECT event, COUNT(*) FROM audit_logs WHERE created_at >= ? GROUP BY event ORDER BY COUNT(*) DESC LIMIT 50`, since)
	if err != nil {
		return nil, err
	}
	out.ByEvent = ev

	// By actor.
	ac, err := s.groupCount(ctx, `SELECT CAST(actor_id AS CHAR), COUNT(*) FROM audit_logs WHERE created_at >= ? AND actor_id IS NOT NULL GROUP BY actor_id ORDER BY COUNT(*) DESC LIMIT 50`, since)
	if err != nil {
		return nil, err
	}
	out.ByActor = ac
	return out, nil
}

func (s *Service) groupCount(ctx context.Context, query string, since time.Time) ([]KeyCount, error) {
	rows, err := s.st.DB.QueryContext(ctx, s.st.Rebind(query), dbTime(since))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []KeyCount
	for rows.Next() {
		var k KeyCount
		if err := rows.Scan(&k.Key, &k.Count); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// Archive moves audit logs older than the retention cutoff into
// audit_logs_archive and deletes them from the live table. Returns the number
// of rows archived.
func (s *Service) Archive(ctx context.Context, retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		return 0, nil
	}
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	cols := `orig_id, actor_id, actor_type, actor_ip, actor_user_agent, event, resource, action,
		target_type, target_id, target_name, old_value, new_value, result, error_message,
		request_id, metadata, created_at`
	ins := s.st.Rebind(`INSERT INTO audit_logs_archive (` + cols + `)
		SELECT id, actor_id, actor_type, actor_ip, actor_user_agent, event, resource, action,
		target_type, target_id, target_name, old_value, new_value, result, error_message,
		request_id, metadata, created_at
		FROM audit_logs WHERE created_at < ?`)
	if _, err := s.st.DB.ExecContext(ctx, ins, dbTime(cutoff)); err != nil {
		return 0, err
	}
	del := s.st.Rebind(`DELETE FROM audit_logs WHERE created_at < ?`)
	res, err := s.st.DB.ExecContext(ctx, del, dbTime(cutoff))
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// Count returns the number of live audit rows matching the filter.
func (s *Service) Count(ctx context.Context, f Query) (int, error) {
	rows, err := s.Query(ctx, func() Query { f.Limit = 200; f.Offset = 0; return f }())
	if err != nil {
		return 0, err
	}
	return len(rows), nil
}
