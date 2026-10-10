package trace

// cloneTraceEvent separates all mutable payloads from caller, observer and
// snapshot ownership. Recorded strings are immutable; only headers are copied.
func cloneTraceEvent(event TraceEvent) TraceEvent {
	if event.Module != nil {
		value := *event.Module
		value.Caps = cloneStrings(value.Caps)
		event.Module = &value
	}
	if event.Function != nil {
		value := *event.Function
		value.Args = cloneStrings(value.Args)
		event.Function = &value
	}
	if event.Effect != nil {
		value := *event.Effect
		value.Args = cloneStrings(value.Args)
		if value.Deterministic != nil {
			deterministic := *value.Deterministic
			value.Deterministic = &deterministic
		}
		if value.Route != nil {
			route := *value.Route
			route.FallbackChain = cloneStrings(route.FallbackChain)
			value.Route = &route
		}
		event.Effect = &value
	}
	if event.Contract != nil {
		value := *event.Contract
		event.Contract = &value
	}
	if event.Budget != nil {
		value := *event.Budget
		event.Budget = &value
	}
	if event.Error != nil {
		value := *event.Error
		event.Error = &value
	}
	if event.Truncation != nil {
		value := *event.Truncation
		event.Truncation = &value
	}
	return event
}

func cloneStrings(values []string) []string {
	if values == nil {
		return nil
	}
	return append(make([]string, 0, len(values)), values...)
}
