package apiserver

import (
	"net/http"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

func TestResultErrStatus(t *testing.T) {
	tests := []struct {
		name       string
		value      eval.Value
		wantStatus int
		wantIsErr  bool
	}{
		{
			name:       "Err(string) returns 400",
			value:      &eval.TaggedValue{TypeName: "Result", CtorName: "Err", Fields: []eval.Value{&eval.StringValue{Value: "bad input"}}},
			wantStatus: http.StatusBadRequest,
			wantIsErr:  true,
		},
		{
			name: "Err({_status: 404, message: ...}) returns 404",
			value: &eval.TaggedValue{TypeName: "Result", CtorName: "Err", Fields: []eval.Value{
				&eval.RecordValue{Fields: map[string]eval.Value{
					"_status": &eval.IntValue{Value: 404},
					"message": &eval.StringValue{Value: "not found"},
				}},
			}},
			wantStatus: http.StatusNotFound,
			wantIsErr:  true,
		},
		{
			name: "Err({_status: 503, message: ...}) returns 503",
			value: &eval.TaggedValue{TypeName: "Result", CtorName: "Err", Fields: []eval.Value{
				&eval.RecordValue{Fields: map[string]eval.Value{
					"_status": &eval.IntValue{Value: 503},
					"message": &eval.StringValue{Value: "upstream down"},
				}},
			}},
			wantStatus: http.StatusServiceUnavailable,
			wantIsErr:  true,
		},
		{
			name: "Err(record without _status) returns 400",
			value: &eval.TaggedValue{TypeName: "Result", CtorName: "Err", Fields: []eval.Value{
				&eval.RecordValue{Fields: map[string]eval.Value{
					"message": &eval.StringValue{Value: "validation failed"},
				}},
			}},
			wantStatus: http.StatusBadRequest,
			wantIsErr:  true,
		},
		{
			name:       "Err(int) returns 400",
			value:      &eval.TaggedValue{TypeName: "Result", CtorName: "Err", Fields: []eval.Value{&eval.IntValue{Value: 42}}},
			wantStatus: http.StatusBadRequest,
			wantIsErr:  true,
		},
		{
			name:       "Err() with no fields returns 400",
			value:      &eval.TaggedValue{TypeName: "Result", CtorName: "Err", Fields: []eval.Value{}},
			wantStatus: http.StatusBadRequest,
			wantIsErr:  true,
		},
		{
			name:      "Ok(value) is not an error",
			value:     &eval.TaggedValue{TypeName: "Result", CtorName: "Ok", Fields: []eval.Value{&eval.StringValue{Value: "success"}}},
			wantIsErr: false,
		},
		{
			name:      "non-Result TaggedValue is not an error",
			value:     &eval.TaggedValue{TypeName: "Option", CtorName: "None", Fields: []eval.Value{}},
			wantIsErr: false,
		},
		{
			name:      "StringValue is not an error",
			value:     &eval.StringValue{Value: "hello"},
			wantIsErr: false,
		},
		{
			name:      "RecordValue is not an error",
			value:     &eval.RecordValue{Fields: map[string]eval.Value{"x": &eval.IntValue{Value: 1}}},
			wantIsErr: false,
		},
		{
			name:      "IntValue is not an error",
			value:     &eval.IntValue{Value: 42},
			wantIsErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, payload, isErr := resultErrStatus(tt.value)
			if isErr != tt.wantIsErr {
				t.Fatalf("isErr = %v, want %v", isErr, tt.wantIsErr)
			}
			if !isErr {
				return
			}
			if status != tt.wantStatus {
				t.Errorf("status = %d, want %d", status, tt.wantStatus)
			}
			if payload == nil {
				t.Error("payload is nil for Err result")
			}
		})
	}
}

func TestResultErrStatus_StripStatus(t *testing.T) {
	// Verify that _status is stripped from the payload record
	value := &eval.TaggedValue{TypeName: "Result", CtorName: "Err", Fields: []eval.Value{
		&eval.RecordValue{Fields: map[string]eval.Value{
			"_status": &eval.IntValue{Value: 404},
			"message": &eval.StringValue{Value: "not found"},
			"code":    &eval.StringValue{Value: "USER_NOT_FOUND"},
		}},
	}}

	status, payload, isErr := resultErrStatus(value)
	if !isErr {
		t.Fatal("expected isErr=true")
	}
	if status != 404 {
		t.Errorf("status = %d, want 404", status)
	}

	rec, ok := payload.(*eval.RecordValue)
	if !ok {
		t.Fatalf("payload type = %T, want *eval.RecordValue", payload)
	}
	if _, has := rec.Fields["_status"]; has {
		t.Error("_status should be stripped from payload")
	}
	if _, has := rec.Fields["message"]; !has {
		t.Error("message field should be preserved")
	}
	if _, has := rec.Fields["code"]; !has {
		t.Error("code field should be preserved")
	}
}
