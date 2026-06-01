package linker

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
)

type PartialFactoryLinkerVersion1 struct {
}

func NewPartialFactoryLinkerVersion1() *PartialFactoryLinkerVersion1 {
	return &PartialFactoryLinkerVersion1{}
}

func (l *PartialFactoryLinkerVersion1) LinkPartialDeclaration(constantPool render3.ConstantPool, metaObj *AstObject, version string) (LinkedDefinition, error) {
	meta, err := l.ToR3FactoryMeta(metaObj, version)
	if err != nil {
		return LinkedDefinition{}, err
	}
	compiled := render3.CompileFactoryFunction(meta)
	return linkedDefinitionFromCompiled(compiled), nil
}

func (l *PartialFactoryLinkerVersion1) ToR3FactoryMeta(metaObj *AstObject, version string) (render3.R3FactoryMetadata, error) {
	typeExpr, err := metaObj.GetOpaque("type")
	if err != nil {
		return nil, err
	}

	name := "unknown"
	if typeValue, err := metaObj.GetValue("type"); err == nil {
		if typeName := typeValue.GetSymbolName(); typeName != "" {
			name = typeName
		}
	}

	targetValue, err := metaObj.GetValue("target")
	var target render3.FactoryTarget
	if err == nil {
		targetName := targetValue.GetSymbolName()
		switch targetName {
		case "Directive":
			target = render3.FactoryTargetDirective
		case "Component":
			target = render3.FactoryTargetComponent
		case "Injectable":
			target = render3.FactoryTargetInjectable
		case "Pipe":
			target = render3.FactoryTargetPipe
		case "NgModule":
			target = render3.FactoryTargetNgModule
		}
	}

	var deps interface{}
	if metaObj.Has("deps") {
		depsValue, err := metaObj.GetValue("deps")
		if err != nil {
			return nil, err
		}
		if !depsValue.IsNull() {
			if depsValue.IsString() {
				depsString, err := depsValue.GetString()
				if err != nil {
					return nil, err
				}
				if depsString != "invalid" {
					return nil, linkerError(depsValue.Node(), "Unsupported deps string %q.", depsString)
				}
				deps = "invalid"
			} else {
				depsArray, err := depsValue.GetArray()
				if err != nil {
					return nil, err
				}
				parsedDeps := make([]render3.R3DependencyMetadata, 0, len(depsArray))
				for _, d := range depsArray {
					depObj, err := d.GetObject()
					if err == nil {
						token, _ := depObj.GetOpaque("token")
						host := depObj.Has("host")
						optional := depObj.Has("optional")
						self := depObj.Has("self")
						skipSelf := depObj.Has("skipSelf")

						parsedDeps = append(parsedDeps, render3.R3DependencyMetadata{
							Token:    token,
							Host:     host,
							Optional: optional,
							Self:     self,
							SkipSelf: skipSelf,
						})
					}
				}
				deps = parsedDeps
			}
		}
	}

	return render3.R3ConstructorFactoryMetadata{
		Name:     name,
		Type:     render3.R3Reference{Value: typeExpr},
		Deps:     deps,
		Target:   target,
	}, nil
}
