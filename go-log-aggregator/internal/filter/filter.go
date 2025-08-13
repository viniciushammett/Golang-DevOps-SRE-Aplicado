package filter

import "regexp"

func CompileOrNil(expr string) (*regexp.Regexp, error) {
	if expr == "" { return nil, nil }
	return regexp.Compile(expr)
}