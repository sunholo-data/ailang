package pipeline

import (
	"strings"
	"testing"
)

func TestQualifiedTypes(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantErr bool
		errSub  string
	}{
		{
			name: "std_set_qualified_type_app",
			src: `module qual_set
import std/set as S

export pure func addOne(s: S.Set[int]) -> S.Set[int] = S.insert(s, 1)
export pure func emptySet() -> S.Set[(int, int)] = S.empty()`,
			wantErr: false,
		},
		{
			name: "std_map_qualified_type_app",
			src: `module qual_map
import std/map as M

export pure func put(m: M.Map[string, int], k: string, v: int) -> M.Map[string, int] =
  M.insert(m, k, v)`,
			wantErr: false,
		},
		{
			name: "std_json_qualified_simple_type",
			src: `module qual_json
import std/json as J

export pure func render(j: J.Json) -> string = J.encode(j)`,
			wantErr: false,
		},
		{
			name: "qualified_and_unqualified_unify_identically",
			src: `module qual_unify
import std/set as S
import std/set (Set)

export pure func id(s: S.Set[int]) -> Set[int] = s`,
			wantErr: false,
		},
		{
			name: "unknown_qualified_type_rejected",
			src: `module qual_unknown
import std/set as S

export pure func bad(x: S.NoSuchType) -> int = 1`,
			wantErr: true,
			errSub:  "TC_UNKNOWN_QUALIFIED_TYPE",
		},
		{
			name: "unknown_module_alias_rejected",
			src: `module qual_no_alias

export pure func bad(x: Unknown.Set[int]) -> int = 1`,
			wantErr: true,
			errSub:  "TC_UNKNOWN_QUALIFIED_TYPE",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := check386(t, tc.name, tc.src)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.errSub)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantErr && tc.errSub != "" && !strings.Contains(err.Error(), tc.errSub) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.errSub)
			}
		})
	}
}
