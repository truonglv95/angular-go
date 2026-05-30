package compiler

// Some of the code comes from WebComponents.JS
// https://github.com/webcomponents/webcomponentsjs/blob/master/src/HTMLImports/path.js

import "regexp"

// urlWithSchemaRegexp matches URLs that have a schema prefix.
var urlWithSchemaRegexp = regexp.MustCompile(`^([^:/?#]+):`)

// IsStyleUrlResolvable returns true if the given URL is resolvable as a style URL.
// Returns false if url is empty, starts with '/', or has a schema other than "package" or "asset".
func IsStyleUrlResolvable(url string) bool {
	if url == "" || url[0] == '/' {
		return false
	}
	matches := urlWithSchemaRegexp.FindStringSubmatch(url)
	if matches == nil {
		return true
	}
	schema := matches[1]
	return schema == "package" || schema == "asset"
}
