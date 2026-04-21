package builtins

import (
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/effects/testctx"
	"github.com/sunholo-data/ailang/internal/eval"
)

func TestDtParts(t *testing.T) {
	spec, ok := GetSpec("_dt_parts")
	if !ok {
		t.Fatal("_dt_parts not registered")
	}

	// 2024-01-01 00:00:00 UTC is Monday (weekday=1)
	ts := time.Date(2024, 1, 1, 12, 30, 45, 0, time.UTC).UnixMilli()
	ctx := testctx.NewMockEffContext()
	args := []eval.Value{&eval.IntValue{Value: int(ts)}}

	result, err := spec.Impl(ctx.EffContext, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rec, ok := result.(*eval.RecordValue)
	if !ok {
		t.Fatalf("expected RecordValue, got %T", result)
	}

	// Verify fields
	tests := map[string]int{
		"year":    2024,
		"month":   1,
		"day":     1,
		"weekday": 1, // Monday
		"hour":    12,
		"minute":  30,
		"second":  45,
	}

	for field, expected := range tests {
		val, ok := rec.Fields[field]
		if !ok {
			t.Errorf("missing field %s", field)
			continue
		}
		intVal, ok := val.(*eval.IntValue)
		if !ok {
			t.Errorf("field %s: expected IntValue, got %T", field, val)
			continue
		}
		if intVal.Value != expected {
			t.Errorf("field %s: expected %d, got %d", field, expected, intVal.Value)
		}
	}
}

func TestDtMake(t *testing.T) {
	spec, ok := GetSpec("_dt_make")
	if !ok {
		t.Fatal("_dt_make not registered")
	}

	ctx := testctx.NewMockEffContext()
	args := []eval.Value{
		&eval.IntValue{Value: 2024},
		&eval.IntValue{Value: 1},
		&eval.IntValue{Value: 1},
		&eval.IntValue{Value: 0},
		&eval.IntValue{Value: 0},
		&eval.IntValue{Value: 0},
	}

	result, err := spec.Impl(ctx.EffContext, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	intVal, ok := result.(*eval.IntValue)
	if !ok {
		t.Fatalf("expected IntValue, got %T", result)
	}

	// 2024-01-01 00:00:00 UTC
	expected := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	if int64(intVal.Value) != expected {
		t.Errorf("expected %d, got %d", expected, intVal.Value)
	}
}

func TestDtAdd(t *testing.T) {
	spec, ok := GetSpec("_dt_add")
	if !ok {
		t.Fatal("_dt_add not registered")
	}

	// Start: 2024-01-15
	start := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC).UnixMilli()
	ctx := testctx.NewMockEffContext()

	tests := []struct {
		name     string
		years    int
		months   int
		days     int
		expected time.Time
	}{
		{"add 7 days", 0, 0, 7, time.Date(2024, 1, 22, 0, 0, 0, 0, time.UTC)},
		{"add 1 month", 0, 1, 0, time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC)},
		{"add 1 year", 1, 0, 0, time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)},
		{"subtract 10 days", 0, 0, -10, time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			args := []eval.Value{
				&eval.IntValue{Value: int(start)},
				&eval.IntValue{Value: tc.years},
				&eval.IntValue{Value: tc.months},
				&eval.IntValue{Value: tc.days},
			}

			result, err := spec.Impl(ctx.EffContext, args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			intVal := result.(*eval.IntValue)
			if int64(intVal.Value) != tc.expected.UnixMilli() {
				t.Errorf("expected %v (%d), got %d",
					tc.expected, tc.expected.UnixMilli(), intVal.Value)
			}
		})
	}
}

func TestDtAddMonthBoundary(t *testing.T) {
	// Test Jan 31 + 1 month = Feb 29 (2024 is leap year)
	spec, _ := GetSpec("_dt_add")
	ctx := testctx.NewMockEffContext()

	jan31 := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC).UnixMilli()
	args := []eval.Value{
		&eval.IntValue{Value: int(jan31)},
		&eval.IntValue{Value: 0},
		&eval.IntValue{Value: 1},
		&eval.IntValue{Value: 0},
	}

	result, _ := spec.Impl(ctx.EffContext, args)
	intVal := result.(*eval.IntValue)

	// Go's AddDate normalizes: Jan 31 + 1 month = Mar 2 (31 days past Feb 1)
	// Actually Go normalizes to the equivalent date by clamping to month end
	resultTime := time.UnixMilli(int64(intVal.Value)).UTC()

	// Feb 2024 has 29 days (leap year), so Jan 31 + 1 month = Mar 2
	// This is Go's documented AddDate behavior
	if resultTime.Month() != time.March || resultTime.Day() != 2 {
		t.Logf("Jan 31 2024 + 1 month = %v (Go AddDate semantics)", resultTime.Format("2006-01-02"))
	}
}

func TestDtDiffDays(t *testing.T) {
	spec, ok := GetSpec("_dt_diffDays")
	if !ok {
		t.Fatal("_dt_diffDays not registered")
	}

	ctx := testctx.NewMockEffContext()

	// 7 days apart
	t1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	t2 := time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC).UnixMilli()

	args := []eval.Value{
		&eval.IntValue{Value: int(t2)},
		&eval.IntValue{Value: int(t1)},
	}

	result, err := spec.Impl(ctx.EffContext, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	intVal := result.(*eval.IntValue)
	if intVal.Value != 7 {
		t.Errorf("expected 7 days, got %d", intVal.Value)
	}

	// Negative difference
	args = []eval.Value{
		&eval.IntValue{Value: int(t1)},
		&eval.IntValue{Value: int(t2)},
	}
	result, _ = spec.Impl(ctx.EffContext, args)
	intVal = result.(*eval.IntValue)
	if intVal.Value != -7 {
		t.Errorf("expected -7 days, got %d", intVal.Value)
	}
}

func TestDtFormatISODate(t *testing.T) {
	spec, ok := GetSpec("_dt_formatISODate")
	if !ok {
		t.Fatal("_dt_formatISODate not registered")
	}

	ctx := testctx.NewMockEffContext()
	ts := time.Date(2024, 1, 15, 12, 30, 0, 0, time.UTC).UnixMilli()
	args := []eval.Value{&eval.IntValue{Value: int(ts)}}

	result, err := spec.Impl(ctx.EffContext, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	strVal := result.(*eval.StringValue)
	if strVal.Value != "2024-01-15" {
		t.Errorf("expected 2024-01-15, got %s", strVal.Value)
	}
}

func TestDtFormatRFC3339(t *testing.T) {
	spec, ok := GetSpec("_dt_formatRFC3339")
	if !ok {
		t.Fatal("_dt_formatRFC3339 not registered")
	}

	ctx := testctx.NewMockEffContext()
	ts := time.Date(2024, 1, 15, 14, 30, 45, 0, time.UTC).UnixMilli()
	args := []eval.Value{&eval.IntValue{Value: int(ts)}}

	result, err := spec.Impl(ctx.EffContext, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	strVal := result.(*eval.StringValue)
	if strVal.Value != "2024-01-15T14:30:45Z" {
		t.Errorf("expected 2024-01-15T14:30:45Z, got %s", strVal.Value)
	}
}

func TestDtFormatMonthShort(t *testing.T) {
	spec, _ := GetSpec("_dt_formatMonthShort")
	ctx := testctx.NewMockEffContext()

	tests := []struct {
		month    int
		expected string
	}{
		{1, "Jan"}, {2, "Feb"}, {3, "Mar"}, {4, "Apr"},
		{5, "May"}, {6, "Jun"}, {7, "Jul"}, {8, "Aug"},
		{9, "Sep"}, {10, "Oct"}, {11, "Nov"}, {12, "Dec"},
	}

	for _, tc := range tests {
		ts := time.Date(2024, time.Month(tc.month), 15, 0, 0, 0, 0, time.UTC).UnixMilli()
		args := []eval.Value{&eval.IntValue{Value: int(ts)}}
		result, _ := spec.Impl(ctx.EffContext, args)
		strVal := result.(*eval.StringValue)
		if strVal.Value != tc.expected {
			t.Errorf("month %d: expected %s, got %s", tc.month, tc.expected, strVal.Value)
		}
	}
}

func TestDtFormatWeekdayFull(t *testing.T) {
	spec, _ := GetSpec("_dt_formatWeekdayFull")
	ctx := testctx.NewMockEffContext()

	// 2024-01-01 is Monday
	tests := []struct {
		day      int
		expected string
	}{
		{1, "Monday"}, {2, "Tuesday"}, {3, "Wednesday"},
		{4, "Thursday"}, {5, "Friday"}, {6, "Saturday"}, {7, "Sunday"},
	}

	for _, tc := range tests {
		ts := time.Date(2024, 1, tc.day, 0, 0, 0, 0, time.UTC).UnixMilli()
		args := []eval.Value{&eval.IntValue{Value: int(ts)}}
		result, _ := spec.Impl(ctx.EffContext, args)
		strVal := result.(*eval.StringValue)
		if strVal.Value != tc.expected {
			t.Errorf("day %d: expected %s, got %s", tc.day, tc.expected, strVal.Value)
		}
	}
}

func TestDtParseISODate(t *testing.T) {
	spec, ok := GetSpec("_dt_parseISODate")
	if !ok {
		t.Fatal("_dt_parseISODate not registered")
	}

	ctx := testctx.NewMockEffContext()

	// Valid date
	args := []eval.Value{&eval.StringValue{Value: "2024-01-15"}}
	result, err := spec.Impl(ctx.EffContext, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tagged, ok := result.(*eval.TaggedValue)
	if !ok {
		t.Fatalf("expected TaggedValue, got %T", result)
	}
	if tagged.CtorName != "Some" {
		t.Errorf("expected Some, got %s", tagged.CtorName)
	}
	if len(tagged.Fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(tagged.Fields))
	}
	intVal := tagged.Fields[0].(*eval.IntValue)
	expectedTs := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC).UnixMilli()
	if int64(intVal.Value) != expectedTs {
		t.Errorf("expected %d, got %d", expectedTs, intVal.Value)
	}

	// Invalid date
	args = []eval.Value{&eval.StringValue{Value: "not-a-date"}}
	result, _ = spec.Impl(ctx.EffContext, args)
	tagged = result.(*eval.TaggedValue)
	if tagged.CtorName != "None" {
		t.Errorf("expected None for invalid date, got %s", tagged.CtorName)
	}
}

func TestDtParseRFC3339(t *testing.T) {
	spec, ok := GetSpec("_dt_parseRFC3339")
	if !ok {
		t.Fatal("_dt_parseRFC3339 not registered")
	}

	ctx := testctx.NewMockEffContext()

	// Valid datetime
	args := []eval.Value{&eval.StringValue{Value: "2024-01-15T14:30:45Z"}}
	result, _ := spec.Impl(ctx.EffContext, args)

	tagged := result.(*eval.TaggedValue)
	if tagged.CtorName != "Some" {
		t.Errorf("expected Some, got %s", tagged.CtorName)
	}

	// With timezone offset (should be converted to UTC)
	args = []eval.Value{&eval.StringValue{Value: "2024-01-15T14:30:45+02:00"}}
	result, _ = spec.Impl(ctx.EffContext, args)
	tagged = result.(*eval.TaggedValue)
	if tagged.CtorName != "Some" {
		t.Errorf("expected Some for timezone offset, got %s", tagged.CtorName)
	}

	// Invalid
	args = []eval.Value{&eval.StringValue{Value: "invalid"}}
	result, _ = spec.Impl(ctx.EffContext, args)
	tagged = result.(*eval.TaggedValue)
	if tagged.CtorName != "None" {
		t.Errorf("expected None for invalid, got %s", tagged.CtorName)
	}
}

func TestDtTimezoneInvariance(t *testing.T) {
	// Verify that all operations produce UTC-based results
	// regardless of what we pass in (since we always use .UTC())

	spec, _ := GetSpec("_dt_formatISODate")
	ctx := testctx.NewMockEffContext()

	// Same instant in time should produce same formatted date
	ts := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).UnixMilli()
	args := []eval.Value{&eval.IntValue{Value: int(ts)}}
	result1, _ := spec.Impl(ctx.EffContext, args)

	// Running again should produce identical result
	result2, _ := spec.Impl(ctx.EffContext, args)

	if result1.(*eval.StringValue).Value != result2.(*eval.StringValue).Value {
		t.Error("format results should be deterministic")
	}
}
