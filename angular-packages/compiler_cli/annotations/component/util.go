package component

import (
	"fmt"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/partial_evaluator"
)

func flattenStyles(value partial_evaluator.ResolvedValue, dest *[]string) {
	if strVal, ok := value.(string); ok {
		*dest = append(*dest, strVal)
	} else if arrVal, ok := value.(partial_evaluator.ResolvedValueArray); ok {
		for _, v := range arrVal {
			flattenStyles(v, dest)
		}
	}
}

func validateAndFlattenImports(value partial_evaluator.ResolvedValue, dest *[]partial_evaluator.ResolvedValue) error {
	if arrVal, ok := value.(partial_evaluator.ResolvedValueArray); ok {
		for _, v := range arrVal {
			if _, isArr := v.(partial_evaluator.ResolvedValueArray); isArr {
				if err := validateAndFlattenImports(v, dest); err != nil {
					return err
				}
			} else {
				*dest = append(*dest, v)
			}
		}
		return nil
	}
	return fmt.Errorf("'imports' must be an array of components, directives, pipes, or NgModules")
}
