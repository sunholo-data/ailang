package sharedmem

import (
	"database/sql"
	"time"

	"github.com/sunholo-data/ailang/internal/effects"
)

var _ effects.CheckedSharedCache = (*SQLiteSharedCache)(nil)

func (c *SQLiteSharedCache) GetChecked(key string) ([]byte, bool, error) {
	var value []byte
	err := c.db.QueryRow(`SELECT value FROM brain_frames WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return value, true, nil
}

// nonNilBytes binds empty bytes as a BLOB, rather than SQL NULL.
func nonNilBytes(value []byte) []byte {
	if value == nil {
		return []byte{}
	}
	return value
}

func (c *SQLiteSharedCache) PutChecked(key string, value []byte) error {
	now := time.Now().UnixMilli()
	_, err := c.db.Exec(`INSERT INTO brain_frames(key,value,created_at,updated_at) VALUES(?,?,?,?)
 ON CONFLICT(key) DO UPDATE SET value=excluded.value,updated_at=excluded.updated_at`, key, nonNilBytes(value), now, now)
	return err
}
func (c *SQLiteSharedCache) DeleteChecked(key string) error {
	_, err := c.db.Exec(`DELETE FROM brain_frames WHERE key=?`, key)
	return err
}

// CASChecked uses a single conditional write. SQLite serializes the condition
// and mutation together even when callers use separate processes/connections.
func (c *SQLiteSharedCache) CASChecked(key string, oldValue, newValue []byte) (bool, error) {
	now := time.Now().UnixMilli()
	var result sql.Result
	var err error
	if oldValue == nil {
		result, err = c.db.Exec(`INSERT INTO brain_frames(key,value,created_at,updated_at) VALUES(?,?,?,?) ON CONFLICT(key) DO NOTHING`, key, nonNilBytes(newValue), now, now)
	} else {
		result, err = c.db.Exec(`UPDATE brain_frames SET value=?,updated_at=? WHERE key=? AND value=?`, nonNilBytes(newValue), now, key, oldValue)
	}
	if err != nil {
		return false, err
	}
	count, err := result.RowsAffected()
	return count == 1, err
}
func (c *SQLiteSharedCache) KeysChecked() ([]string, error) {
	rows, err := c.db.Query(`SELECT key FROM brain_frames ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	keys := []string{}
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, rows.Err()
}
