package builtins

// registerBytesCodegenSpecs supplies the same byte representation as the
// interpreter: owned []byte buffers, with int64 values at generated-Go boundaries.
// Run after effect specs, whose historical string bridges claimed these exports.
func registerBytesCodegenSpecs() {
	for _, oldName := range []string{"_str_fromString", "_str_toString"} {
		if meta := Registry[oldName]; meta != nil && meta.GoCodegen != nil {
			// Retain direct alias lookup for compatibility, but hand qualified
			// std/bytes claims to the actual bytes implementations below.
			meta.GoCodegen.StdlibName = ""
			meta.GoCodegen.StdlibModule = ""
		}
	}
	registerIfMissing("_bytes_from_ints", 1, true, &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName: "BytesFromInts", Signature: "func BytesFromInts(xs interface{}) interface{}",
			Body: `if xs == nil { return make([]byte, 0) }
	list := reflect.ValueOf(xs)
	if list.Kind() != reflect.Slice { panic(fmt.Sprintf("_bytes_from_ints: expected List, got %T", xs)) }
	out := make([]byte, list.Len())
	for i := 0; i < list.Len(); i++ {
		var n int64
		switch x := list.Index(i).Interface().(type) {
		case int64: n = x
		case int: n = int64(x)
		default: panic(fmt.Sprintf("_bytes_from_ints: expected Int element at index %d", i))
		}
		if n < 0 || n > 255 { panic(fmt.Sprintf("_bytes_from_ints: value %d at index %d out of byte range (0-255)", n, i)) }
		out[i] = byte(n)
	}
	return out`,
		},
		Imports: []string{"fmt", "reflect"}, StdlibName: "fromInts", StdlibModule: "std/bytes",
	})
	registerIfMissing("_bytes_to_ints", 1, true, &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName: "BytesToInts", Signature: "func BytesToInts(v interface{}) interface{}",
			Body: `b := v.([]byte)
	out := make([]interface{}, len(b))
	for i, x := range b { out[i] = int64(x) }
	return out`,
		},
		StdlibName: "toInts", StdlibModule: "std/bytes",
	})
	registerIfMissing("_bytes_length", 1, true, &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName: "BytesLength", Signature: "func BytesLength(v interface{}) interface{}",
			Body: `return int64(len(v.([]byte)))`,
		},
		StdlibName: "length", StdlibModule: "std/bytes",
	})
	registerIfMissing("_bytes_slice", 3, true, &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName: "BytesSlice", Signature: "func BytesSlice(v interface{}, start interface{}, length interface{}) interface{}",
			Body: `b := v.([]byte)
	s, n, total := toInt64(start), toInt64(length), int64(len(b))
	if s < 0 || n < 0 || s > total || n > total-s { return NewOptionNone() }
	out := make([]byte, int(n))
	copy(out, b[int(s):int(s+n)])
	return NewOptionSome(out)`,
		},
		StdlibName: "slice", StdlibModule: "std/bytes",
	})
	registerIfMissing("_bytes_byte_at", 2, true, &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName: "BytesByteAt", Signature: "func BytesByteAt(v interface{}, index interface{}) interface{}",
			Body: `b := v.([]byte)
	i := toInt64(index)
	if i < 0 || i >= int64(len(b)) { return NewOptionNone() }
	return NewOptionSome(int64(b[int(i)]))`,
		},
		StdlibName: "byteAt", StdlibModule: "std/bytes",
	})
	registerIfMissing("_bytes_concat", 2, true, &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName: "BytesConcat", Signature: "func BytesConcat(x interface{}, y interface{}) interface{}",
			Body: `a, b := x.([]byte), y.([]byte)
	if len(b) > int(^uint(0)>>1)-len(a) { panic("_bytes_concat: result exceeds host size") }
	out := make([]byte, len(a)+len(b))
	copy(out, a)
	copy(out[len(a):], b)
	return out`,
		},
		StdlibName: "concat", StdlibModule: "std/bytes",
	})
	registerIfMissing("_bytes_from_string", 1, true, &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName: "BytesFromString", Signature: "func BytesFromString(s interface{}) interface{}",
			Body: `return []byte(s.(string))`,
		},
		StdlibName: "fromString", StdlibModule: "std/bytes",
	})
	registerIfMissing("_bytes_to_string", 1, true, &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName: "BytesToString", Signature: "func BytesToString(b interface{}) interface{}",
			Body: `return string(b.([]byte))`,
		},
		StdlibName: "toString", StdlibModule: "std/bytes",
	})
}
