package compiler

// ResourceLoader is an interface for retrieving documents by URL that the compiler uses
// to load templates.
//
// This corresponds to the abstract class in TypeScript, which acts as both an abstract class
// and an injection token. In Go we represent it as an interface.
type ResourceLoader interface {
	// Get retrieves the content at the given URL. Returns the content as a string.
	Get(url string) (string, error)
}
