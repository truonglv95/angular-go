package imports

type ImportRewriter interface {
	RewriteSymbol(symbol string, specifier string) string
	RewriteSpecifier(specifier string, inContextOfFile string) string
	RewriteNamespaceImportIdentifier(specifier string, moduleName string) string
}

type NoopImportRewriter struct {
}

func (recv *NoopImportRewriter) RewriteSymbol(symbol string, specifier string) string {
	// TODO: stub
	panic("unimplemented")
}

func (recv *NoopImportRewriter) RewriteSpecifier(specifier string, inContextOfFile string) string {
	// TODO: stub
	panic("unimplemented")
}

func (recv *NoopImportRewriter) RewriteNamespaceImportIdentifier(specifier string) string {
	// TODO: stub
	panic("unimplemented")
}

type R3SymbolsImportRewriter struct {
}

func (recv *R3SymbolsImportRewriter) RewriteSymbol(symbol string, specifier string) string {
	// TODO: stub
	panic("unimplemented")
}

func (recv *R3SymbolsImportRewriter) RewriteSpecifier(specifier string, inContextOfFile string) string {
	// TODO: stub
	panic("unimplemented")
}

func (recv *R3SymbolsImportRewriter) RewriteNamespaceImportIdentifier(specifier string) string {
	// TODO: stub
	panic("unimplemented")
}

func ValidateAndRewriteCoreSymbol(name string) string {
	// TODO: stub
	panic("unimplemented")
}
