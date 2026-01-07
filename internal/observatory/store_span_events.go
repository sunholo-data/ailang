// Package observatory provides a unified observability platform for AILANG.
package observatory

import (
	"database/sql"
)

// CreateSpanEvent inserts a new span event.
func (s *Store) CreateSpanEvent(e *SpanEvent) error {
	result, err := s.db.Exec(`
		INSERT INTO span_events (span_id, name, timestamp, event_type,
		                         approval_status, tool_name, error_message, attributes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, e.SpanID, e.Name, e.Timestamp, e.EventType,
		e.ApprovalStatus, e.ToolName, e.ErrorMessage, e.AttributesJSON())
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	e.ID = id
	return nil
}

// GetSpanEvents retrieves all events for a span.
func (s *Store) GetSpanEvents(spanID string) ([]SpanEvent, error) {
	rows, err := s.db.Query(`
		SELECT id, span_id, name, timestamp, event_type,
		       approval_status, tool_name, error_message, attributes
		FROM span_events WHERE span_id = ?
		ORDER BY timestamp ASC
	`, spanID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []SpanEvent
	for rows.Next() {
		e := SpanEvent{}
		var eventType, approvalStatus, toolName, errorMessage sql.NullString
		var attributesJSON string
		if err := rows.Scan(&e.ID, &e.SpanID, &e.Name, &e.Timestamp, &eventType,
			&approvalStatus, &toolName, &errorMessage, &attributesJSON); err != nil {
			return nil, err
		}
		if eventType.Valid {
			e.EventType = EventType(eventType.String)
		}
		if approvalStatus.Valid {
			e.ApprovalStatus = ApprovalStatus(approvalStatus.String)
		}
		if toolName.Valid {
			e.ToolName = toolName.String
		}
		if errorMessage.Valid {
			e.ErrorMessage = errorMessage.String
		}
		e.ParseAttributes(attributesJSON)
		events = append(events, e)
	}
	return events, rows.Err()
}

// DeleteSpanEvent removes a span event by ID.
func (s *Store) DeleteSpanEvent(id int64) error {
	_, err := s.db.Exec("DELETE FROM span_events WHERE id = ?", id)
	return err
}
