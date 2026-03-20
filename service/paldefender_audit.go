package service

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/zaigie/palworld-server-tool/internal/database"
	"go.etcd.io/bbolt"
)

type PalDefenderAuditLogFilter struct {
	Limit     int
	Action    string
	BatchID   string
	PlayerUID string
	UserID    string
	Success   *bool
	ErrorCode string
}

func AddPalDefenderAuditLog(db *bbolt.DB, log database.PalDefenderAuditLog) error {
	if log.ID == "" {
		log.ID = fmt.Sprintf("%020d", time.Now().UTC().UnixNano())
	}
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now().UTC()
	}
	return db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("paldefender_audit_logs"))
		value, err := json.Marshal(log)
		if err != nil {
			return err
		}
		return b.Put([]byte(log.ID), value)
	})
}

func ListPalDefenderAuditLogs(db *bbolt.DB, limit int) ([]database.PalDefenderAuditLog, error) {
	return ListPalDefenderAuditLogsByFilter(db, PalDefenderAuditLogFilter{Limit: limit})
}

func ListPalDefenderAuditLogsByFilter(db *bbolt.DB, filter PalDefenderAuditLogFilter) ([]database.PalDefenderAuditLog, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	logs := make([]database.PalDefenderAuditLog, 0, filter.Limit)
	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("paldefender_audit_logs"))
		if b == nil {
			return nil
		}
		return b.ForEach(func(_, value []byte) error {
			var log database.PalDefenderAuditLog
			if err := json.Unmarshal(value, &log); err != nil {
				return err
			}
			if !matchesPalDefenderAuditLogFilter(log, filter) {
				return nil
			}
			logs = append(logs, log)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	sort.SliceStable(logs, func(i, j int) bool {
		return logs[i].CreatedAt.After(logs[j].CreatedAt)
	})
	if len(logs) > filter.Limit {
		logs = logs[:filter.Limit]
	}
	return logs, nil
}

func matchesPalDefenderAuditLogFilter(log database.PalDefenderAuditLog, filter PalDefenderAuditLogFilter) bool {
	if filter.Action != "" && !strings.EqualFold(strings.TrimSpace(filter.Action), strings.TrimSpace(log.Action)) {
		return false
	}
	if filter.BatchID != "" && !strings.EqualFold(strings.TrimSpace(filter.BatchID), strings.TrimSpace(log.BatchID)) {
		return false
	}
	if filter.PlayerUID != "" && !strings.EqualFold(strings.TrimSpace(filter.PlayerUID), strings.TrimSpace(log.PlayerUID)) {
		return false
	}
	if filter.UserID != "" && !strings.EqualFold(strings.TrimSpace(filter.UserID), strings.TrimSpace(log.UserID)) {
		return false
	}
	if filter.Success != nil && log.Success != *filter.Success {
		return false
	}
	if filter.ErrorCode != "" && !strings.EqualFold(strings.TrimSpace(filter.ErrorCode), strings.TrimSpace(log.ErrorCode)) {
		return false
	}
	return true
}
