package transform

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/imports"
	"github.com/microsoft/typescript-go/angular-packages/compiler"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/internal/ast"
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
	Handler    DecoratorHandler
	Decorator  *reflection.Decorator
	State      TraitState
	Analysis   any
	Resolution any
	Diagnostics []ast.Diagnostic
}

type ResourceDependencyProvider interface {
	ResourceDependencies(sourceFile *ast.SourceFile) []string
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
}
