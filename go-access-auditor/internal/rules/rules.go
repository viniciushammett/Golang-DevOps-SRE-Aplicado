package rules

import "regexp"

type Rule struct {
	Name  string `yaml:"name"`
	Regex string `yaml:"regex"`
}

type Set struct {
	items []compiled
}
type compiled struct {
	name string
	re   *regexp.Regexp
}

func New(rs []Rule) *Set {
	var out []compiled
	for _, r := range rs {
		if rx, err := regexp.Compile(r.Regex); err == nil {
			out = append(out, compiled{name: r.Name, re: rx})
		}
	}
	return &Set{items: out}
}

func (s *Set) Match(line string) (matched bool, ruleName string) {
	for _, c := range s.items {
		if c.re.MatchString(line) { return true, c.name }
	}
	return false, ""
}