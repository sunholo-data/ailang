package vm_test

import "testing"

func TestJSONSchemaBothEngines(t *testing.T) {
	source := `module jsonschema_parity
import std/json (decode, Json, JNull)
import std/result (Ok, Err)
import std/jsonschema (validate, validateLenient, validateSchema)
func parse(s: string) -> Json = match decode(s) { Ok(j) => j, Err(_) => JNull }
export func main() -> string {
 let schema = parse("{\"type\":\"object\",\"required\":[\"x\"],\"properties\":{\"x\":{\"minimum\":2}}}");
 let good = validate(schema, parse("{\"x\":3}"));
 let bad = validate(schema, parse("{\"x\":1}"));
 let strict = validate(parse("{\"unknown\":true}"), JNull);
 let lenient = validateLenient(parse("{\"unknown\":true}"), JNull);
 let invalid = validateSchema(parse("{\"type\":42}"));
 "${show(good)} ${show(bad)} ${show(strict)} ${show(lenient)} ${show(invalid)}"
}`
	e, ee, v, ve := runBothEngines(t, "jsonschema_parity", source)
	if ee != nil || ve != nil {
		t.Fatalf("eval %v; VM %v", ee, ve)
	}
	if e != v {
		t.Fatalf("eval %s; VM %s", e, v)
	}
}
