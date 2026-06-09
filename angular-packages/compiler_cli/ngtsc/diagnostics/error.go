package diagnostics

import (
	"fmt"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/astnav"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/diagnostics"
)

type FatalDiagnosticError struct {
	Code                   ErrorCode
	Node                   *ast.Node
	DiagnosticMessage      interface{} // string | *ast.Diagnostic
	RelatedInformation     []*ast.Diagnostic
	isFatalDiagnosticError bool
}

func (e *FatalDiagnosticError) Error() string {
	msgText := ""
	switch v := e.DiagnosticMessage.(type) {
	case string:
		msgText = v
	case *ast.Diagnostic:
		// Simplified flattening
		msgText = v.String()
	}
	return fmt.Sprintf("FatalDiagnosticError: Code: %d, Message: %s", e.Code, msgText)
}

func NewFatalDiagnosticError(code ErrorCode, node *ast.Node, diagnosticMessage interface{}, relatedInformation []*ast.Diagnostic) *FatalDiagnosticError {
	return &FatalDiagnosticError{
		Code:                   code,
		Node:                   node,
		DiagnosticMessage:      diagnosticMessage,
		RelatedInformation:     relatedInformation,
		isFatalDiagnosticError: true,
	}
}

func (e *FatalDiagnosticError) ToDiagnostic() *ast.Diagnostic {
	return MakeDiagnostic(e.Code, e.Node, e.DiagnosticMessage, e.RelatedInformation, diagnostics.CategoryError)
}

func MakeDiagnostic(code ErrorCode, node *ast.Node, messageText interface{}, relatedInformation []*ast.Diagnostic, category diagnostics.Category) *ast.Diagnostic {
	var file *ast.SourceFile
	loc := core.UndefinedTextRange()
	if node != nil {
		file = ast.GetSourceFileOfNode(node)
		if file != nil {
			start := astnav.GetStartOfNode(node, file, false)
			loc = core.NewTextRange(start, node.End())
		} else {
			loc = node.Loc
		}
	}

	var d *ast.Diagnostic
	msgArgs := []string{}
	var chain []*ast.Diagnostic
	switch v := messageText.(type) {
	case string:
		msgArgs = append(msgArgs, v)
	case *ast.Diagnostic:
		msgArgs = v.MessageArgs()
		chain = v.MessageChain()
	}

	d = ast.NewDiagnosticFromSerialized(
		file,
		loc,
		int32(NgErrorCode(code)),
		category,
		diagnostics.Key(""),
		msgArgs,
		chain, // msgChain
		relatedInformation,
		false,
		false,
		false,
	)

	return d
}

func MakeDiagnosticChain(messageText string, next []*ast.Diagnostic) *ast.Diagnostic {
	return ast.NewDiagnosticFromSerialized(
		nil,
		core.UndefinedTextRange(),
		0,
		diagnostics.CategoryMessage,
		diagnostics.Key(""),
		[]string{messageText},
		next,
		nil,
		false,
		false,
		false,
	)
}

func MakeRelatedInformation(node *ast.Node, messageText string) *ast.Diagnostic {
	var file *ast.SourceFile
	loc := core.UndefinedTextRange()
	if node != nil {
		file = ast.GetSourceFileOfNode(node)
		if file != nil {
			start := astnav.GetStartOfNode(node, file, false)
			loc = core.NewTextRange(start, node.End())
		} else {
			loc = node.Loc
		}
	}

	return ast.NewDiagnosticFromSerialized(
		file,
		loc,
		0,
		diagnostics.CategoryMessage,
		diagnostics.Key(""),
		[]string{messageText},
		nil,
		nil,
		false,
		false,
		false,
	)
}

func AddDiagnosticChain(messageText interface{}, add []*ast.Diagnostic) *ast.Diagnostic {
	switch v := messageText.(type) {
	case string:
		return MakeDiagnosticChain(v, add)
	case *ast.Diagnostic:
		for _, d := range add {
			v.AddMessageChain(d)
		}
		return v
	}
	return nil
}

func IsFatalDiagnosticError(err error) bool {
	if fde, ok := err.(*FatalDiagnosticError); ok {
		return fde.isFatalDiagnosticError
	}
	return false
}

func IsLocalCompilationDiagnostics(diagnostic *ast.Diagnostic) bool {
	if diagnostic == nil {
		return false
	}
	return int(diagnostic.Code()) == NgErrorCode(ErrorCode_LOCAL_COMPILATION_UNRESOLVED_CONST) ||
		int(diagnostic.Code()) == NgErrorCode(ErrorCode_LOCAL_COMPILATION_UNSUPPORTED_EXPRESSION)
}
