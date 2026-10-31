package fizz

import "regexp"

var (
	ginPathParamRe = regexp.MustCompile(`\/:([^\/]*)`)
	// A gin catch-all is always the last segment of the path.
	ginWildcardRe = regexp.MustCompile(`\/\*([^\/]*)$`)
)

// openAPIPath converts a Gin route path to the OpenAPI syntax:
// path parameters (":name") and catch-all wildcards ("*name") both
// become "{name}". Use wildcardParams to tell them apart.
func openAPIPath(ginPath string) string {
	ginPath = ginWildcardRe.ReplaceAllString(ginPath, "/{$1}")
	return ginPathParamRe.ReplaceAllString(ginPath, "/{$1}")
}

// wildcardParams returns the names of the catch-all wildcards of a
// Gin route path (at most one).
func wildcardParams(ginPath string) []string {
	if m := ginWildcardRe.FindStringSubmatch(ginPath); m != nil {
		return []string{m[1]}
	}
	return nil
}
