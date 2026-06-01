package linker

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
)

type PartialComponentLinkerVersion1 struct {
	sourceURL string
	code      string
}

func NewPartialComponentLinkerVersion1(sourceURL string, code string) *PartialComponentLinkerVersion1 {
	return &PartialComponentLinkerVersion1{sourceURL: sourceURL, code: code}
}

func (l *PartialComponentLinkerVersion1) LinkPartialDeclaration(constantPool render3.ConstantPool, metaObj *AstObject, version string) (LinkedDefinition, error) {
	meta, err := l.ToR3ComponentMeta(metaObj, version)
	if err != nil {
		return LinkedDefinition{}, err
	}
	compiled := render3.CompileComponentFromMetadata(meta, constantPool, render3.MakeBindingParser(false))
	return linkedDefinitionFromCompiled(compiled), nil
}

func (l *PartialComponentLinkerVersion1) ToR3ComponentMeta(metaObj *AstObject, version string) (render3.R3ComponentMetadata[render3.R3TemplateDependency], error) {
	baseMeta, err := ToR3DirectiveMeta(metaObj, l.code, l.sourceURL, version)
	if err != nil {
		return render3.R3ComponentMetadata[render3.R3TemplateDependency]{}, err
	}
	templateValue, err := metaObj.GetValue("template")
	if err != nil {
		return render3.R3ComponentMetadata[render3.R3TemplateDependency]{}, err
	}
	templateText, err := templateValue.GetString()
	if err != nil {
		return render3.R3ComponentMetadata[render3.R3TemplateDependency]{}, err
	}
	parsed := render3.ParseTemplate(templateText, l.sourceURL, &render3.ParseTemplateOptions{})
	if len(parsed.Errors) > 0 {
		return render3.R3ComponentMetadata[render3.R3TemplateDependency]{}, linkerError(templateValue.Node(), "Errors found in the template: %v", parsed.Errors)
	}

	declarations, hasDirectiveDeps, err := toTemplateDependencies(metaObj)
	if err != nil {
		return render3.R3ComponentMetadata[render3.R3TemplateDependency]{}, err
	}

	styles, err := toStringArray(metaObj, "styles")
	if err != nil {
		return render3.R3ComponentMetadata[render3.R3TemplateDependency]{}, err
	}

	var viewProviders output.Expression
	if metaObj.Has("viewProviders") {
		viewProviders, err = metaObj.GetOpaque("viewProviders")
		if err != nil {
			return render3.R3ComponentMetadata[render3.R3TemplateDependency]{}, err
		}
	}
	var animations output.Expression
	if metaObj.Has("animations") {
		animations, err = metaObj.GetOpaque("animations")
		if err != nil {
			return render3.R3ComponentMetadata[render3.R3TemplateDependency]{}, err
		}
	}

	encapsulation := 0
	if metaObj.Has("encapsulation") {
		encapsulation, err = toComponentEnumValue(metaObj, "encapsulation", map[string]int{
			"Emulated":                      0,
			"None":                          2,
			"ShadowDom":                     3,
			"ExperimentalIsolatedShadowDom": 4,
		})
		if err != nil {
			return render3.R3ComponentMetadata[render3.R3TemplateDependency]{}, err
		}
	}

	var changeDetection render3.Expression
	if metaObj.Has("changeDetection") {
		cd, err := toComponentEnumValue(metaObj, "changeDetection", map[string]int{
			"OnPush":  0,
			"Default": 1,
			"Eager":   1,
		})
		if err != nil {
			return render3.R3ComponentMetadata[render3.R3TemplateDependency]{}, err
		}
		changeDetection = cd
	}

	return render3.R3ComponentMetadata[render3.R3TemplateDependency]{
		R3DirectiveMetadata:      baseMeta,
		Template:                 render3.Template{Children: parsed.Nodes},
		Declarations:             declarations,
		DeclarationListEmitMode:  render3.DeclarationListEmitMode_Direct,
		Styles:                   styles,
		Encapsulation:            encapsulation,
		Animations:               animations,
		ViewProviders:            viewProviders,
		RelativeContextFilePath:  l.sourceURL,
		I18nUseExternalIds:       false,
		ChangeDetection:          changeDetection,
		RelativeTemplatePath:     nil,
		HasDirectiveDependencies: hasDirectiveDeps,
		ForeignImports:           nil,
	}, nil
}

func toComponentEnumValue(metaObj *AstObject, field string, members map[string]int) (int, error) {
	value, err := metaObj.GetValue(field)
	if err != nil {
		return 0, err
	}
	if value.IsNumber() {
		number, err := value.GetNumber()
		if err != nil {
			return 0, err
		}
		return int(number), nil
	}
	memberName := value.GetSymbolName()
	if enumValue, ok := members[memberName]; ok {
		return enumValue, nil
	}
	return 0, linkerError(value.Node(), "Unsupported component metadata enum value for %q.", field)
}

func toTemplateDependencies(metaObj *AstObject) ([]render3.R3TemplateDependency, bool, error) {
	var declarations []render3.R3TemplateDependency
	hasDirectiveDeps := false

	if metaObj.Has("dependencies") {
		deps, err := metaObj.GetArray("dependencies")
		if err != nil {
			return nil, false, err
		}
		for _, dep := range deps {
			obj, err := dep.GetObject()
			if err != nil {
				return nil, false, err
			}
			kind, err := obj.GetString("kind")
			if err != nil {
				return nil, false, err
			}
			typeNode, err := obj.GetNode("type")
			if err != nil {
				return nil, false, err
			}
			switch kind {
			case "directive", "component":
				hasDirectiveDeps = true
				declarations = append(declarations, render3.R3TemplateDependency{
					Kind: render3.R3TemplateDependencyKind_Directive,
					Type: output.NewWrappedNodeExpr(typeNode, nil, nil, nil),
				})
			case "pipe":
				declarations = append(declarations, render3.R3TemplateDependency{
					Kind: render3.R3TemplateDependencyKind_Pipe,
					Type: output.NewWrappedNodeExpr(typeNode, nil, nil, nil),
				})
			case "ngmodule":
				hasDirectiveDeps = true
				declarations = append(declarations, render3.R3TemplateDependency{
					Kind: render3.R3TemplateDependencyKind_NgModule,
					Type: output.NewWrappedNodeExpr(typeNode, nil, nil, nil),
				})
			}
		}
	}

	return declarations, hasDirectiveDeps, nil
}
