package linker

import "strings"

const (
	DeclareDirective          = "ɵɵngDeclareDirective"
	DeclareClassMetadata      = "ɵɵngDeclareClassMetadata"
	DeclareComponent          = "ɵɵngDeclareComponent"
	DeclareFactory            = "ɵɵngDeclareFactory"
	DeclareInjectable         = "ɵɵngDeclareInjectable"
	DeclareInjector           = "ɵɵngDeclareInjector"
	DeclareNgModule           = "ɵɵngDeclareNgModule"
	DeclarePipe               = "ɵɵngDeclarePipe"
	DeclareClassMetadataAsync = "ɵɵngDeclareClassMetadataAsync"
	DeclareService            = "ɵɵngDeclareService"
)

var DeclarationFunctions = []string{
	DeclareDirective,
	DeclareClassMetadata,
	DeclareComponent,
	DeclareFactory,
	DeclareInjectable,
	DeclareInjector,
	DeclareNgModule,
	DeclarePipe,
	DeclareClassMetadataAsync,
	DeclareService,
}

// NeedsLinking is the same fast pre-parse filter used by ngtsc's linker. It is
// intentionally conservative: true means a file may need linker processing.
func NeedsLinking(path string, source string) bool {
	_ = path
	for _, fn := range DeclarationFunctions {
		if strings.Contains(source, fn) {
			return true
		}
	}
	return false
}
