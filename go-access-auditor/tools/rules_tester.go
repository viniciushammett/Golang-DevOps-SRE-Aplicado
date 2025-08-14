package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

type Rule struct {
	Name  string `yaml:"name"`
	Regex string `yaml:"regex"`
}
type Example struct {
	Line     string   `json:"line"`
	Expected []string `json:"expected"` // nomes das regras que devem casar
}

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: rules-tester <rules.yaml> <examples.jsonl>")
		os.Exit(2)
	}
	rulesPath := os.Args[1]
	examplesPath := os.Args[2]

	// load rules
	var rules []Rule
	b, err := os.ReadFile(rulesPath); if err != nil { panic(err) }
	if err := yaml.Unmarshal(b, &rules); err != nil { panic(err) }

	compiled := make([]struct{
		Name string
		Re   *regexp.Regexp
	}, 0, len(rules))
	for _, r := range rules {
		re, err := regexp.Compile(r.Regex)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid regex in rule %q: %v\n", r.Name, err)
			os.Exit(1)
		}
		compiled = append(compiled, struct{
			Name string; Re *regexp.Regexp
		}{r.Name, re})
	}

	// open examples (JSONL)
	f, err := os.Open(examplesPath); if err != nil { panic(err) }
	defer f.Close()

	s := bufio.NewScanner(f)
	lineNo := 0
	failed := 0
	for s.Scan() {
		lineNo++
		var ex Example
		if err := json.Unmarshal([]byte(s.Text()), &ex); err != nil {
			fmt.Fprintf(os.Stderr, "invalid JSONL line %d: %v\n", lineNo, err)
			failed++
			continue
		}
		seen := map[string]bool{}
		for _, c := range compiled {
			if c.Re.MatchString(ex.Line) {
				seen[c.Name] = true
			}
		}
		// compare with expected
		ok := true
		for _, want := range ex.Expected {
			if !seen[want] { ok = false }
		}
		if !ok {
			fmt.Printf("❌ line %d: %q\n  expected: %v\n  got: %v\n", lineNo, ex.Line, ex.Expected, keys(seen))
			failed++
		} else {
			fmt.Printf("✅ %q -> %v\n", ex.Line, keys(seen))
		}
	}
	if failed > 0 {
		fmt.Fprintf(os.Stderr, "\nFAILED: %d example(s) mismatched\n", failed)
		os.Exit(1)
	}
}

func keys(m map[string]bool) []string {
	out := make([]string,0,len(m))
	for k := range m { out = append(out, k) }
	return out
}