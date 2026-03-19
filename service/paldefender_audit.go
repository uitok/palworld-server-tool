package service

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/zaigie/palworld-server-tool/internal/database"
	"go.etcd.io/bbolt"
)

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
	if limit <= 0 {
		limit = 20
	}
	logs := make([]database.PalDefenderAuditLog, 0, limit)
	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("paldefender_audit_logs"))
		if b == nil {
			return nil
		}
		cursor := b.Cursor()
		for key, value := cursor.Last(); key != nil && len(logs) < limit; key, value = cursor.Prev() {
			var log database.PalDefenderAuditLog
			if err := json.Unmarshal(value, &log); err != nil {
				return err
			}
			logs = append(logs, log)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.SliceStable(logs, func(i, j int) bool {
		return logs[i].CreatedAt.After(logs[j].CreatedAt)
	})
	return logs, nil
}
