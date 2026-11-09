package coordinator

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

// sqliteCascadeEnvelope reads back the cascade envelope columns CreateTask
// writes (M-PKG-CASCADE-DETERMINISTIC-FIRST).
//
// M-CASCADE-ENVELOPE-HYDRATION: the SQLite store wrote all eleven fields and
// read none of them, so a task listed for dispatch from SQLite kept its
// Source but lost its envelope — the same EMPTY dispatch the inbox routes
// produced, one hop later. The Firestore store read them all; this is that
// read for the other backend.
type sqliteCascadeEnvelope struct {
	rootPackage, rootChangeClass, fromVersion, toVersion sql.NullString
	fromInterfaceHash, toInterfaceHash                   sql.NullString
	fromContentHash, toContentHash                       sql.NullString
	effectsWidened                                       sql.NullBool
	prevEffectCeiling, newEffectCeiling                  sql.NullString // JSON-encoded []string
}

func (e *sqliteCascadeEnvelope) apply(t *TaskRecord) error {
	t.RootPackage = e.rootPackage.String
	t.RootChangeClass = e.rootChangeClass.String
	t.FromVersion = e.fromVersion.String
	t.ToVersion = e.toVersion.String
	t.FromInterfaceHash = e.fromInterfaceHash.String
	t.ToInterfaceHash = e.toInterfaceHash.String
	t.FromContentHash = e.fromContentHash.String
	t.ToContentHash = e.toContentHash.String
	t.EffectsWidened = e.effectsWidened.Bool
	var err error
	if t.PrevEffectCeiling, err = decodeEffectCeiling(e.prevEffectCeiling); err != nil {
		return fmt.Errorf("task %s: prev_effect_ceiling: %w", t.ID, err)
	}
	if t.NewEffectCeiling, err = decodeEffectCeiling(e.newEffectCeiling); err != nil {
		return fmt.Errorf("task %s: new_effect_ceiling: %w", t.ID, err)
	}
	return nil
}

// decodeEffectCeiling returns nil for an absent ceiling and an error for an
// unparseable one: the ceiling decides whether effects widened, so a corrupt
// row must not read as "no ceiling".
func decodeEffectCeiling(col sql.NullString) ([]string, error) {
	if !col.Valid || col.String == "" {
		return nil, nil
	}
	var out []string
	if err := json.Unmarshal([]byte(col.String), &out); err != nil {
		return nil, err
	}
	return out, nil
}
