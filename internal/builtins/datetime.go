package builtins

import (
	"time"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

func init() {
	registerDtParts()
	registerDtMake()
	registerDtAdd()
	registerDtDiffDays()
	registerDtFormatISODate()
	registerDtFormatRFC3339()
	registerDtFormatMonthShort()
	registerDtFormatWeekdayFull()
	registerDtParseISODate()
	registerDtParseRFC3339()
}

// =============================================================================
// _dt_parts: Extract all date components from timestamp
// =============================================================================

func registerDtParts() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/datetime",
		Name:    "_dt_parts",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeDtPartsType,
		Impl:    dtPartsImpl,
		Metadata: &BuiltinMetadata{
			Description: "Extract all date/time components from a UTC timestamp",
			Params: []ParamDoc{
				{Name: "ts", Description: "Unix timestamp in milliseconds (UTC)"},
			},
			Returns: "Record with year, month, day, weekday, hour, minute, second",
			Examples: []Example{
				{Code: "_dt_parts(1704067200000)", Description: "Returns {year: 2024, month: 1, day: 1, weekday: 1, hour: 0, minute: 0, second: 0}"},
			},
			Since:     "v0.7.0",
			Stability: StabilityStable,
			Tags:      []string{"datetime", "extraction", "pure"},
			Category:  "datetime",
		},
	})
	if err != nil {
		panic("failed to register _dt_parts builtin: " + err.Error())
	}
}

func makeDtPartsType() types.Type {
	T := types.NewBuilder()
	// int -> {year: int, month: int, day: int, weekday: int, hour: int, minute: int, second: int}
	return T.Func(T.Int()).Returns(
		T.Record(
			types.Field("year", T.Int()),
			types.Field("month", T.Int()),
			types.Field("day", T.Int()),
			types.Field("weekday", T.Int()),
			types.Field("hour", T.Int()),
			types.Field("minute", T.Int()),
			types.Field("second", T.Int()),
		),
	).Build()
}

func dtPartsImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	ts := args[0].(*eval.IntValue).Value
	t := time.UnixMilli(int64(ts)).UTC()

	return &eval.RecordValue{
		Fields: map[string]eval.Value{
			"year":    &eval.IntValue{Value: t.Year()},
			"month":   &eval.IntValue{Value: int(t.Month())},
			"day":     &eval.IntValue{Value: t.Day()},
			"weekday": &eval.IntValue{Value: int(t.Weekday())}, // 0=Sun, 6=Sat
			"hour":    &eval.IntValue{Value: t.Hour()},
			"minute":  &eval.IntValue{Value: t.Minute()},
			"second":  &eval.IntValue{Value: t.Second()},
		},
	}, nil
}

// =============================================================================
// _dt_make: Construct timestamp from components
// =============================================================================

func registerDtMake() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/datetime",
		Name:    "_dt_make",
		NumArgs: 6,
		IsPure:  true,
		Effect:  "",
		Type:    makeDtMakeType,
		Impl:    dtMakeImpl,
		Metadata: &BuiltinMetadata{
			Description: "Construct UTC timestamp from date/time components",
			Params: []ParamDoc{
				{Name: "year", Description: "Year (e.g., 2024)"},
				{Name: "month", Description: "Month (1-12)"},
				{Name: "day", Description: "Day of month (1-31)"},
				{Name: "hour", Description: "Hour (0-23)"},
				{Name: "minute", Description: "Minute (0-59)"},
				{Name: "second", Description: "Second (0-59)"},
			},
			Returns: "Unix timestamp in milliseconds (UTC)",
			Examples: []Example{
				{Code: "_dt_make(2024, 1, 1, 0, 0, 0)", Description: "Returns 1704067200000"},
			},
			Since:     "v0.7.0",
			Stability: StabilityStable,
			Tags:      []string{"datetime", "construction", "pure"},
			Category:  "datetime",
		},
	})
	if err != nil {
		panic("failed to register _dt_make builtin: " + err.Error())
	}
}

func makeDtMakeType() types.Type {
	T := types.NewBuilder()
	// (int, int, int, int, int, int) -> int
	return T.Func(T.Int(), T.Int(), T.Int(), T.Int(), T.Int(), T.Int()).Returns(T.Int()).Build()
}

func dtMakeImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	year := args[0].(*eval.IntValue).Value
	month := args[1].(*eval.IntValue).Value
	day := args[2].(*eval.IntValue).Value
	hour := args[3].(*eval.IntValue).Value
	minute := args[4].(*eval.IntValue).Value
	second := args[5].(*eval.IntValue).Value

	t := time.Date(year, time.Month(month), day, hour, minute, second, 0, time.UTC)
	return &eval.IntValue{Value: int(t.UnixMilli())}, nil
}

// =============================================================================
// _dt_add: Date arithmetic (add years, months, days)
// =============================================================================

func registerDtAdd() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/datetime",
		Name:    "_dt_add",
		NumArgs: 4,
		IsPure:  true,
		Effect:  "",
		Type:    makeDtAddType,
		Impl:    dtAddImpl,
		Metadata: &BuiltinMetadata{
			Description: "Add years, months, and days to a timestamp (uses Go time.AddDate semantics)",
			Params: []ParamDoc{
				{Name: "ts", Description: "Unix timestamp in milliseconds (UTC)"},
				{Name: "years", Description: "Years to add (can be negative)"},
				{Name: "months", Description: "Months to add (can be negative)"},
				{Name: "days", Description: "Days to add (can be negative)"},
			},
			Returns: "New timestamp in milliseconds (UTC)",
			Examples: []Example{
				{Code: "_dt_add(ts, 0, 0, 7)", Description: "Add 7 days"},
				{Code: "_dt_add(ts, 1, 0, 0)", Description: "Add 1 year"},
				{Code: "_dt_add(ts, 0, 0, -30)", Description: "Subtract 30 days"},
			},
			Since:     "v0.7.0",
			Stability: StabilityStable,
			Tags:      []string{"datetime", "arithmetic", "pure"},
			Category:  "datetime",
		},
	})
	if err != nil {
		panic("failed to register _dt_add builtin: " + err.Error())
	}
}

func makeDtAddType() types.Type {
	T := types.NewBuilder()
	// (int, int, int, int) -> int
	return T.Func(T.Int(), T.Int(), T.Int(), T.Int()).Returns(T.Int()).Build()
}

func dtAddImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	ts := args[0].(*eval.IntValue).Value
	years := args[1].(*eval.IntValue).Value
	months := args[2].(*eval.IntValue).Value
	days := args[3].(*eval.IntValue).Value

	t := time.UnixMilli(int64(ts)).UTC()
	t = t.AddDate(years, months, days)
	return &eval.IntValue{Value: int(t.UnixMilli())}, nil
}

// =============================================================================
// _dt_diffDays: Calculate difference in whole days
// =============================================================================

func registerDtDiffDays() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/datetime",
		Name:    "_dt_diffDays",
		NumArgs: 2,
		IsPure:  true,
		Effect:  "",
		Type:    makeDtDiffDaysType,
		Impl:    dtDiffDaysImpl,
		Metadata: &BuiltinMetadata{
			Description: "Calculate difference in whole days between two timestamps (a - b)",
			Params: []ParamDoc{
				{Name: "a", Description: "First timestamp in milliseconds"},
				{Name: "b", Description: "Second timestamp in milliseconds"},
			},
			Returns: "Number of whole days (can be negative)",
			Examples: []Example{
				{Code: "_dt_diffDays(t2, t1)", Description: "Days between t1 and t2"},
			},
			Since:     "v0.7.0",
			Stability: StabilityStable,
			Tags:      []string{"datetime", "arithmetic", "pure"},
			Category:  "datetime",
		},
	})
	if err != nil {
		panic("failed to register _dt_diffDays builtin: " + err.Error())
	}
}

func makeDtDiffDaysType() types.Type {
	T := types.NewBuilder()
	// (int, int) -> int
	return T.Func(T.Int(), T.Int()).Returns(T.Int()).Build()
}

func dtDiffDaysImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	a := args[0].(*eval.IntValue).Value
	b := args[1].(*eval.IntValue).Value

	// Milliseconds per day = 24 * 60 * 60 * 1000 = 86400000
	const msPerDay = 86400000
	diff := (a - b) / msPerDay
	return &eval.IntValue{Value: diff}, nil
}

// =============================================================================
// _dt_formatISODate: Format timestamp as ISO 8601 date
// =============================================================================

func registerDtFormatISODate() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/datetime",
		Name:    "_dt_formatISODate",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeDtFormatStringType,
		Impl:    dtFormatISODateImpl,
		Metadata: &BuiltinMetadata{
			Description: "Format timestamp as ISO 8601 date (YYYY-MM-DD)",
			Params: []ParamDoc{
				{Name: "ts", Description: "Unix timestamp in milliseconds (UTC)"},
			},
			Returns: "Date string in ISO 8601 format",
			Examples: []Example{
				{Code: "_dt_formatISODate(1704067200000)", Description: "Returns \"2024-01-01\""},
			},
			Since:     "v0.7.0",
			Stability: StabilityStable,
			Tags:      []string{"datetime", "format", "pure"},
			Category:  "datetime",
		},
	})
	if err != nil {
		panic("failed to register _dt_formatISODate builtin: " + err.Error())
	}
}

func makeDtFormatStringType() types.Type {
	T := types.NewBuilder()
	// int -> string
	return T.Func(T.Int()).Returns(T.String()).Build()
}

func dtFormatISODateImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	ts := args[0].(*eval.IntValue).Value
	t := time.UnixMilli(int64(ts)).UTC()
	return &eval.StringValue{Value: t.Format("2006-01-02")}, nil
}

// =============================================================================
// _dt_formatRFC3339: Format timestamp as RFC 3339
// =============================================================================

func registerDtFormatRFC3339() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/datetime",
		Name:    "_dt_formatRFC3339",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeDtFormatStringType,
		Impl:    dtFormatRFC3339Impl,
		Metadata: &BuiltinMetadata{
			Description: "Format timestamp as RFC 3339 (YYYY-MM-DDTHH:MM:SSZ)",
			Params: []ParamDoc{
				{Name: "ts", Description: "Unix timestamp in milliseconds (UTC)"},
			},
			Returns: "DateTime string in RFC 3339 format with Z suffix",
			Examples: []Example{
				{Code: "_dt_formatRFC3339(1704067200000)", Description: "Returns \"2024-01-01T00:00:00Z\""},
			},
			Since:     "v0.7.0",
			Stability: StabilityStable,
			Tags:      []string{"datetime", "format", "pure"},
			Category:  "datetime",
		},
	})
	if err != nil {
		panic("failed to register _dt_formatRFC3339 builtin: " + err.Error())
	}
}

func dtFormatRFC3339Impl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	ts := args[0].(*eval.IntValue).Value
	t := time.UnixMilli(int64(ts)).UTC()
	return &eval.StringValue{Value: t.Format(time.RFC3339)}, nil
}

// =============================================================================
// _dt_formatMonthShort: Format timestamp as short month name
// =============================================================================

func registerDtFormatMonthShort() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/datetime",
		Name:    "_dt_formatMonthShort",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeDtFormatStringType,
		Impl:    dtFormatMonthShortImpl,
		Metadata: &BuiltinMetadata{
			Description: "Format timestamp as short month name (Jan, Feb, ...)",
			Params: []ParamDoc{
				{Name: "ts", Description: "Unix timestamp in milliseconds (UTC)"},
			},
			Returns: "Short month name",
			Examples: []Example{
				{Code: "_dt_formatMonthShort(1704067200000)", Description: "Returns \"Jan\""},
			},
			Since:     "v0.7.0",
			Stability: StabilityStable,
			Tags:      []string{"datetime", "format", "pure"},
			Category:  "datetime",
		},
	})
	if err != nil {
		panic("failed to register _dt_formatMonthShort builtin: " + err.Error())
	}
}

func dtFormatMonthShortImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	ts := args[0].(*eval.IntValue).Value
	t := time.UnixMilli(int64(ts)).UTC()
	return &eval.StringValue{Value: t.Format("Jan")}, nil
}

// =============================================================================
// _dt_formatWeekdayFull: Format timestamp as full weekday name
// =============================================================================

func registerDtFormatWeekdayFull() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/datetime",
		Name:    "_dt_formatWeekdayFull",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeDtFormatStringType,
		Impl:    dtFormatWeekdayFullImpl,
		Metadata: &BuiltinMetadata{
			Description: "Format timestamp as full weekday name (Monday, Tuesday, ...)",
			Params: []ParamDoc{
				{Name: "ts", Description: "Unix timestamp in milliseconds (UTC)"},
			},
			Returns: "Full weekday name",
			Examples: []Example{
				{Code: "_dt_formatWeekdayFull(1704067200000)", Description: "Returns \"Monday\""},
			},
			Since:     "v0.7.0",
			Stability: StabilityStable,
			Tags:      []string{"datetime", "format", "pure"},
			Category:  "datetime",
		},
	})
	if err != nil {
		panic("failed to register _dt_formatWeekdayFull builtin: " + err.Error())
	}
}

func dtFormatWeekdayFullImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	ts := args[0].(*eval.IntValue).Value
	t := time.UnixMilli(int64(ts)).UTC()
	return &eval.StringValue{Value: t.Format("Monday")}, nil
}

// =============================================================================
// _dt_parseISODate: Parse ISO 8601 date string
// =============================================================================

func registerDtParseISODate() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/datetime",
		Name:    "_dt_parseISODate",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeDtParseOptionType,
		Impl:    dtParseISODateImpl,
		Metadata: &BuiltinMetadata{
			Description: "Parse ISO 8601 date string (YYYY-MM-DD) to timestamp",
			Params: []ParamDoc{
				{Name: "s", Description: "Date string in ISO 8601 format"},
			},
			Returns: "Option[int]: Some(timestamp) if valid, None if invalid",
			Examples: []Example{
				{Code: "_dt_parseISODate(\"2024-01-01\")", Description: "Returns Some(1704067200000)"},
				{Code: "_dt_parseISODate(\"invalid\")", Description: "Returns None"},
			},
			Since:     "v0.7.0",
			Stability: StabilityStable,
			Tags:      []string{"datetime", "parse", "pure"},
			Category:  "datetime",
		},
	})
	if err != nil {
		panic("failed to register _dt_parseISODate builtin: " + err.Error())
	}
}

func makeDtParseOptionType() types.Type {
	T := types.NewBuilder()
	// string -> Option[int]
	return T.Func(T.String()).Returns(T.App("Option", T.Int())).Build()
}

func dtParseISODateImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	s := args[0].(*eval.StringValue).Value

	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		// Return None
		return &eval.TaggedValue{
			ModulePath: "std/option",
			TypeName:   "Option",
			CtorName:   "None",
			Fields:     []eval.Value{},
		}, nil
	}

	// Return Some(timestamp)
	return &eval.TaggedValue{
		ModulePath: "std/option",
		TypeName:   "Option",
		CtorName:   "Some",
		Fields:     []eval.Value{&eval.IntValue{Value: int(t.UTC().UnixMilli())}},
	}, nil
}

// =============================================================================
// _dt_parseRFC3339: Parse RFC 3339 datetime string
// =============================================================================

func registerDtParseRFC3339() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/datetime",
		Name:    "_dt_parseRFC3339",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeDtParseOptionType,
		Impl:    dtParseRFC3339Impl,
		Metadata: &BuiltinMetadata{
			Description: "Parse RFC 3339 datetime string to timestamp",
			Params: []ParamDoc{
				{Name: "s", Description: "DateTime string in RFC 3339 format"},
			},
			Returns: "Option[int]: Some(timestamp) if valid, None if invalid",
			Examples: []Example{
				{Code: "_dt_parseRFC3339(\"2024-01-01T00:00:00Z\")", Description: "Returns Some(1704067200000)"},
				{Code: "_dt_parseRFC3339(\"invalid\")", Description: "Returns None"},
			},
			Since:     "v0.7.0",
			Stability: StabilityStable,
			Tags:      []string{"datetime", "parse", "pure"},
			Category:  "datetime",
		},
	})
	if err != nil {
		panic("failed to register _dt_parseRFC3339 builtin: " + err.Error())
	}
}

func dtParseRFC3339Impl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	s := args[0].(*eval.StringValue).Value

	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		// Return None
		return &eval.TaggedValue{
			ModulePath: "std/option",
			TypeName:   "Option",
			CtorName:   "None",
			Fields:     []eval.Value{},
		}, nil
	}

	// Return Some(timestamp) - always convert to UTC
	return &eval.TaggedValue{
		ModulePath: "std/option",
		TypeName:   "Option",
		CtorName:   "Some",
		Fields:     []eval.Value{&eval.IntValue{Value: int(t.UTC().UnixMilli())}},
	}, nil
}
