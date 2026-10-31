package fizz

import "regexp"

var ginPathParamRe = regexp.MustCompile(`\/:([^\/]*)`)

// openAPIPath converts a Gin route path that uses colons to declare
// path parameters to the OpenAPI syntax that uses curly braces.
// Wildcards ("*name") are left untouched.
func openAPIPath(ginPath string) string {
	return ginPathParamRe.ReplaceAllString(ginPath, "/{$1}")
}
