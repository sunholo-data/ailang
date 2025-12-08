package link

import (
	"fmt"

	"github.com/sunholo/ailang/internal/builtins"
)

func DebugBuiltinTypes() {
	concatSpec, _ := builtins.GetSpec("concat_List")
	consSpec, _ := builtins.GetSpec("::")

	concatType := concatSpec.Type()
	consType := consSpec.Type()

	fmt.Printf("concat_List type: %T - %s\n", concatType, concatType)
	fmt.Printf(":: type: %T - %s\n", consType, consType)

	concatVars := extractTypeVars(concatType)
	consVars := extractTypeVars(consType)

	fmt.Printf("concat_List type vars: %v\n", concatVars)
	fmt.Printf(":: type vars: %v\n", consVars)
}
