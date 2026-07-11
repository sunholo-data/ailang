package types

import "testing"

// M-EFFECT-MODE-VALIDATION M1 — schema/defaults consistency guard.

// TestEffectSchemaDefaultsConsistent enforces the M1 invariant: every registered
// default in defaultEffectModes must itself be a legal member of effectSchema.
// A default that is not a schema value would be rejected the moment it was
// applied — a self-inconsistent table. This is the compile-time cross-check the
// sprint plan requires (M1 acceptance).
func TestEffectSchemaDefaultsConsistent(t *testing.T) {
	for effect, def := range defaultEffectModes {
		schema, ok := effectSchema[effect]
		if !ok {
			t.Errorf("effect %q has a default (%s=%s) but no effectSchema entry", effect, def.Key, def.Value)
			continue
		}
		allowed, keyOK := schema[def.Key]
		if !keyOK {
			t.Errorf("effect %q default key %q is not a schema key (keys: %v)", effect, def.Key, sortedSetKeys(schema))
			continue
		}
		if _, valueOK := allowed[def.Value]; !valueOK {
			t.Errorf("effect %q default value %q=%q is not in the allowed set %v",
				effect, def.Key, def.Value, sortedValues(allowed))
		}
	}
}
