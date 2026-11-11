package golang

// emittedShowRuntime is the `show` / `showPreview` runtime written into every
// compiled program (writeRuntimeMiscHelpers). It ports the spec of
// internal/eval/show.go (M-V1-SHOW-TOTAL): total — no width, depth or byte
// limit; a top-level string bare and a nested one double-quoted with the
// escaper of internal/strutil/quote.go; floats in shortest round-trip digits
// with ".0"; records in field-name order; bytes as <bytes:HEX> in full.
//
// Differences from the interpreters, both deliberate for an Experimental
// backend: the walk is recursive (one strings.Builder, so linear in output,
// but a value nested deeper than the goroutine stack allows still overflows),
// and ShowPreview renders in full before cutting.
//
// A runtime helper cannot add imports (the runtime import block is fmt, math,
// reflect, sort, strconv, strings), so an invalid UTF-8 byte is recognised as
// a U+FFFD from `range` whose source bytes are not the encoding of U+FFFD.
// TestEmittedShowMatchesEvaluator compiles this source and compares it with
// eval.Show on every shape the emitter can represent.
const emittedShowRuntime = `// Show converts any value to its string form, exactly as the show builtin does.
func Show(v interface{}) string {
	var b strings.Builder
	showInto(&b, v, false)
	return b.String()
}

// ShowPreview is std/string.showPreview: the first maxChars code points of
// Show(v), then "…(+N chars)" when anything was left out.
func ShowPreview(v interface{}, maxChars interface{}) interface{} {
	s := Show(v)
	n := int(toInt64(maxChars))
	if n < 0 {
		n = 0
	}
	kept, i := 0, 0
	for i < len(s) && kept < n {
		_, size := showDecodeRune(s, i)
		i += size
		kept++
	}
	if i == len(s) {
		return s
	}
	rest := 0
	for j := i; j < len(s); {
		_, size := showDecodeRune(s, j)
		j += size
		rest++
	}
	return s[:i] + "…(+" + strconv.Itoa(rest) + " chars)"
}

// showDecodeRune decodes the rune at s[i:]; an invalid byte is one rune of size 1.
func showDecodeRune(s string, i int) (rune, int) {
	for _, r := range s[i:] {
		if r == '�' && !strings.HasPrefix(s[i:], "�") {
			return r, 1
		}
		return r, len(string(r))
	}
	return 0, 0
}

func showQuote(b *strings.Builder, s string) {
	b.WriteByte('"')
	for i := 0; i < len(s); {
		r, size := showDecodeRune(s, i)
		switch {
		case r == '�' && size == 1:
			b.WriteString("\\x" + strings.ToUpper(fmt.Sprintf("%02x", s[i])))
		case r == '"':
			b.WriteString("\\\"")
		case r == '\\':
			b.WriteString("\\\\")
		case r == '\n':
			b.WriteString("\\n")
		case r == '\t':
			b.WriteString("\\t")
		case r == '\r':
			b.WriteString("\\r")
		case r == '$' && i+1 < len(s) && s[i+1] == '{':
			b.WriteString("\\$")
		case r != ' ' && (r < 0x20 || r == 0x7f || !strconv.IsPrint(r)):
			b.WriteString("\\u{" + strings.ToUpper(strconv.FormatInt(int64(r), 16)) + "}")
		default:
			b.WriteString(s[i : i+size])
		}
		i += size
	}
	b.WriteByte('"')
}

func showFloat(x float64) string {
	if math.IsNaN(x) {
		return "NaN"
	}
	if math.IsInf(x, 1) {
		return "Inf"
	}
	if math.IsInf(x, -1) {
		return "-Inf"
	}
	s := strconv.FormatFloat(x, 'f', -1, 64)
	if !strings.Contains(s, ".") {
		s += ".0"
	}
	return s
}

func showInto(b *strings.Builder, v interface{}, nested bool) {
	switch x := v.(type) {
	case string:
		if nested {
			showQuote(b, x)
		} else {
			b.WriteString(x)
		}
	case int64:
		b.WriteString(strconv.FormatInt(x, 10))
	case int:
		b.WriteString(strconv.Itoa(x))
	case float64:
		b.WriteString(showFloat(x))
	case bool:
		b.WriteString(strconv.FormatBool(x))
	case nil:
		b.WriteString("()")
	case []byte:
		b.WriteString("<bytes:" + fmt.Sprintf("%x", x) + ">")
	case ArrayVal, FloatArrayVal:
		showSequence(b, reflect.ValueOf(x), "#[", "]")
	case Tuple:
		showSequence(b, reflect.ValueOf(x), "(", ")")
	default:
		showReflect(b, reflect.ValueOf(x), nested)
	}
}

func showReflect(b *strings.Builder, v reflect.Value, nested bool) {
	if !v.IsValid() {
		b.WriteString("()")
		return
	}
	switch v.Kind() {
	case reflect.Interface:
		if v.IsNil() {
			b.WriteString("()")
			return
		}
		showInto(b, v.Elem().Interface(), nested)
	case reflect.Ptr:
		if v.IsNil() {
			b.WriteString("()")
			return
		}
		showReflect(b, v.Elem(), nested)
	case reflect.Slice, reflect.Array:
		showSequence(b, v, "[", "]")
	case reflect.Map:
		showMap(b, v)
	case reflect.Struct:
		showStruct(b, v)
	case reflect.Func:
		b.WriteString("<function>")
	default:
		b.WriteString("<unknown>")
	}
}

func showSequence(b *strings.Builder, v reflect.Value, open, close string) {
	b.WriteString(open)
	for i := 0; i < v.Len(); i++ {
		if i > 0 {
			b.WriteString(", ")
		}
		showInto(b, v.Index(i).Interface(), true)
	}
	b.WriteString(close)
}

func showMap(b *strings.Builder, v reflect.Value) {
	if v.Type().Key().Kind() != reflect.String {
		b.WriteString("<map>")
		return
	}
	tag := v.MapIndex(reflect.ValueOf("_tag"))
	keys := make([]string, 0, v.Len())
	for _, key := range v.MapKeys() {
		if !tag.IsValid() || key.String() != "_tag" {
			keys = append(keys, key.String())
		}
	}
	sort.Strings(keys)
	if tag.IsValid() {
		ctor, ok := tag.Interface().(string)
		if !ok {
			b.WriteString("<unknown>")
			return
		}
		b.WriteString(ctor)
		if len(keys) == 0 {
			return
		}
		b.WriteString("(")
		for i, key := range keys {
			if i > 0 {
				b.WriteString(", ")
			}
			showInto(b, v.MapIndex(reflect.ValueOf(key)).Interface(), true)
		}
		b.WriteString(")")
		return
	}
	// Go maps also back inline records, so this renders record syntax.
	b.WriteString("{")
	for i, key := range keys {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(key + ": ")
		showInto(b, v.MapIndex(reflect.ValueOf(key)).Interface(), true)
	}
	b.WriteString("}")
}

func showStruct(b *strings.Builder, v reflect.Value) {
	if v.NumField() == 0 {
		b.WriteString("()")
		return
	}
	if kind := v.FieldByName("Kind"); kind.IsValid() {
		for i := 0; i < v.NumField(); i++ {
			fieldInfo := v.Type().Field(i)
			field := v.Field(i)
			if fieldInfo.Name == "Kind" || field.Kind() != reflect.Ptr || field.IsNil() {
				continue
			}
			variant := field.Elem()
			b.WriteString(fieldInfo.Name)
			if variant.NumField() == 0 {
				return
			}
			b.WriteString("(")
			for j := 0; j < variant.NumField(); j++ {
				if j > 0 {
					b.WriteString(", ")
				}
				showInto(b, variant.Field(j).Interface(), true)
			}
			b.WriteString(")")
			return
		}
		b.WriteString("<unknown>")
		return
	}
	fields := make(map[string]reflect.Value, v.NumField())
	keys := make([]string, 0, v.NumField())
	for i := 0; i < v.NumField(); i++ {
		fieldInfo := v.Type().Field(i)
		if fieldInfo.PkgPath != "" {
			continue
		}
		name := strings.ToLower(fieldInfo.Name[:1]) + fieldInfo.Name[1:]
		keys = append(keys, name)
		fields[name] = v.Field(i)
	}
	sort.Strings(keys)
	b.WriteString("{")
	for i, key := range keys {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(key + ": ")
		showInto(b, fields[key].Interface(), true)
	}
	b.WriteString("}")
}

`
