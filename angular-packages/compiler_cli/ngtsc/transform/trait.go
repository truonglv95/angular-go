package transform

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/imports"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/incremental/semantic_graph"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/internal/ast"
	"sync"
)

type TraitState int

const (
	TraitStatePending TraitState = iota
	TraitStateAnalyzed
	TraitStateResolved
	TraitStateSkipped
	TraitStateErrored
)

// Trait đại diện cho một Decorator đã được detect trên một Class (vd: @Component)
// và lưu trữ toàn bộ trạng thái của nó qua các Phase biên dịch.
type Trait struct {
	Handler     DecoratorHandler
	Decorator   *reflection.Decorator
	State       TraitState
	Analysis    any
	Resolution  any
	resolveOnce sync.Once
	Diagnostics []ast.Diagnostic
}

func (t *Trait) GetResolution(c *ast.ClassDeclaration) any {
	t.resolveOnce.Do(func() {
		if t.State == TraitStateAnalyzed && t.Analysis != nil {
			resolution, diags := t.Handler.Resolve(c, t.Analysis)
			t.Resolution = resolution
			if len(diags) > 0 {
				t.State = TraitStateErrored
				t.Diagnostics = append(t.Diagnostics, diags...)
			} else {
				t.State = TraitStateResolved
			}
		}
	})
	return t.Resolution
}

type ResourceDependencyProvider interface {
	ResourceDependencies(sourceFile *ast.SourceFile) []string
}

type SemanticSymbolProvider interface {
	GetSemanticSymbol(node *ast.ClassDeclaration, analysis any) *semantic_graph.SemanticSymbol
}

type SemanticReferenceProvider interface {
	GetSemanticReferenceKeys(node *ast.ClassDeclaration, analysis any, resolution any) []string
}

type SemanticReferenceSymbolProvider interface {
	GetSemanticReferenceSymbols(node *ast.ClassDeclaration, analysis any, resolution any) []semantic_graph.SemanticSymbol
}

// DecoratorHandler định nghĩa vòng đời biên dịch của một Angular Decorator.
// Các class như ComponentDecoratorHandler, DirectiveDecoratorHandler sẽ implement interface này.
type DecoratorHandler interface {
	Name() string

	// Detect kiểm tra xem class này có chứa decorator mà handler quan tâm hay không.
	Detect(node *ast.ClassDeclaration, decorators []reflection.Decorator) *reflection.Decorator

	// Analyze đọc metadata từ decorator bằng AST (không dùng regex).
	Analyze(node *ast.ClassDeclaration, decorator *reflection.Decorator) (any, []ast.Diagnostic)

	// Resolve giải quyết các tham chiếu (ví dụ tìm dependencies, imports).
	Resolve(node *ast.ClassDeclaration, analysis any) (any, []ast.Diagnostic)

	// CompileFull sinh ra các hàm Ivy như ɵcmp, ɵfac, ɵdir...
	CompileFull(node *ast.ClassDeclaration, analysis any, resolution any, pool *compiler.ConstantPool, importMgr *imports.ImportManager, factory *ast.NodeFactory) ([]CompileResult, []ast.Diagnostic)
}

// CompileResult chứa kết quả của quá trình Compile, ví dụ một thuộc tính `ɵcmp`
// cần được gắn vào class.
type CompileResult struct {
	PropertyName string
	Initializer  *ast.Node
	Type         *ast.Node
	Statements   []*ast.Node
	// HmrUpdateNodes holds the export-default function declaration for the HMR
	// virtual module (e.g. `export default function App_UpdateMetadata(App) {...}`).
	HmrUpdateNodes []*ast.Node
	// HmrImports holds the namespace import declarations produced by the
	// HMR-specific ImportManager (e.g. `import * as i0 from '@angular/core'`).
	// These must be prepended to HmrUpdateNodes when building the virtual source
	// file so that the function body can reference i0, i1, etc.
	HmrImports []*ast.Node
}
