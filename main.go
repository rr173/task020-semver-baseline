// Command go-task-check is a semantic version comparison and range-matching tool.
package main

import (
	"fmt"
	"os"
	"strings"

	"go-task-check/semver"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		usage(os.Stderr)
		os.Exit(2)
	}
	if args[0] == "--smoke-test" {
		os.Exit(smokeTest())
	}
	cmd := args[0]
	rest := args[1:]
	var err error
	switch cmd {
	case "compare":
		err = runCompare(rest)
	case "satisfies":
		err = runSatisfies(rest)
	case "max":
		err = runPick(rest, true)
	case "min":
		err = runPick(rest, false)
	case "sort":
		err = runSort(rest)
	case "valid":
		err = runValid(rest)
	case "-h", "--help", "help":
		usage(os.Stdout)
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", cmd)
		usage(os.Stderr)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func usage(w *os.File) {
	fmt.Fprintln(w, "usage: go-task-check <command> [args...]")
	fmt.Fprintln(w, "commands:")
	fmt.Fprintln(w, "  compare <a> <b>            compare two versions (-1/0/1)")
	fmt.Fprintln(w, "  satisfies <version> <range> check if version satisfies range")
	fmt.Fprintln(w, "  max <v1> [v2 ...]          highest precedence version")
	fmt.Fprintln(w, "  min <v1> [v2 ...]          lowest precedence version")
	fmt.Fprintln(w, "  sort <v1> [v2 ...]         sort versions ascending")
	fmt.Fprintln(w, "  valid <version>            validate a version string")
	fmt.Fprintln(w, "  --smoke-test               run built-in self checks")
}

func parseVersions(args []string) ([]semver.Version, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("missing version argument")
	}
	vs := make([]semver.Version, 0, len(args))
	for _, a := range args {
		v, err := semver.Parse(a)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", a, err)
		}
		vs = append(vs, v)
	}
	return vs, nil
}

func runCompare(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("compare requires exactly two versions")
	}
	a, err := semver.Parse(args[0])
	if err != nil {
		return fmt.Errorf("%s: %w", args[0], err)
	}
	b, err := semver.Parse(args[1])
	if err != nil {
		return fmt.Errorf("%s: %w", args[1], err)
	}
	fmt.Println(a.Compare(b))
	return nil
}

func runSatisfies(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("satisfies requires a version and a range")
	}
	v, err := semver.Parse(args[0])
	if err != nil {
		return fmt.Errorf("version %s: %w", args[0], err)
	}
	r, err := semver.ParseRange(args[1])
	if err != nil {
		return fmt.Errorf("range %q: %w", args[1], err)
	}
	fmt.Println(r.Satisfies(v))
	return nil
}

func runPick(args []string, wantMax bool) error {
	vs, err := parseVersions(args)
	if err != nil {
		return err
	}
	if wantMax {
		v, _ := semver.Max(vs)
		fmt.Println(v.String())
	} else {
		v, _ := semver.Min(vs)
		fmt.Println(v.String())
	}
	return nil
}

func runSort(args []string) error {
	vs, err := parseVersions(args)
	if err != nil {
		return err
	}
	semver.Sort(vs)
	lines := make([]string, len(vs))
	for i, v := range vs {
		lines[i] = v.String()
	}
	fmt.Println(strings.Join(lines, "\n"))
	return nil
}

func runValid(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("valid requires exactly one version")
	}
	if _, err := semver.Parse(args[0]); err != nil {
		fmt.Printf("invalid: %s\n", err)
		os.Exit(1)
	}
	fmt.Println("valid")
	return nil
}

func smokeTest() int {
	fail := 0
	check := func(name string, cond bool) {
		if !cond {
			fmt.Println("FAIL:", name)
			fail++
		}
	}
	mustV := func(s string) semver.Version {
		v, err := semver.Parse(s)
		if err != nil {
			fmt.Printf("FAIL: parse %s: %v\n", s, err)
			fail++
		}
		return v
	}
	mustR := func(s string) semver.Range {
		r, err := semver.ParseRange(s)
		if err != nil {
			fmt.Printf("FAIL: parse range %q: %v\n", s, err)
			fail++
		}
		return r
	}

	// Precedence.
	check("compare 1.0.0<2.0.0", mustV("1.0.0").Compare(mustV("2.0.0")) == -1)
	check("compare 2.0.0>1.0.0", mustV("2.0.0").Compare(mustV("1.0.0")) == 1)
	check("compare build ignored", mustV("1.0.0+build1").Compare(mustV("1.0.0+build2")) == 0)
	check("prerelease < release", mustV("1.0.0-alpha").Compare(mustV("1.0.0")) == -1)
	check("beta.11 > beta.2", mustV("1.0.0-beta.11").Compare(mustV("1.0.0-beta.2")) == 1)
	check("alpha < alpha.1", mustV("1.0.0-alpha").Compare(mustV("1.0.0-alpha.1")) == -1)
	check("alpha.1 < alpha.beta", mustV("1.0.0-alpha.1").Compare(mustV("1.0.0-alpha.beta")) == -1)
	check("numeric < alpha", mustV("1.0.0-1").Compare(mustV("1.0.0-alpha")) == -1)
	check("rc.1 < release", mustV("1.0.0-rc.1").Compare(mustV("1.0.0")) == -1)

	// Validation (strict).
	var err error
	_, err = semver.Parse("1.2")
	check("reject 1.2", err != nil)
	_, err = semver.Parse("01.2.3")
	check("reject leading-zero major", err != nil)
	_, err = semver.Parse("1.2.3-01")
	check("reject leading-zero prerelease", err != nil)
	_, err = semver.Parse("v1.2.3")
	check("reject v-prefix", err != nil)
	_, err = semver.Parse("1.2.3.4")
	check("reject extra component", err != nil)
	_, err = semver.Parse("1.2.3-")
	check("reject empty prerelease", err != nil)
	_, err = semver.Parse("1.2.3+")
	check("reject empty build", err != nil)
	_, err = semver.Parse("1.2.3-alpha..1")
	check("reject empty prerelease id", err != nil)
	_, err = semver.Parse("1.2.3")
	check("accept 1.2.3", err == nil)

	// Range satisfaction.
	sat := func(ver, rng string, want bool) {
		got := mustR(rng).Satisfies(mustV(ver))
		check(fmt.Sprintf("satisfies %s %q == %v", ver, rng, want), got == want)
	}
	sat("1.5.0", ">=1.0.0 <2.0.0", true)
	sat("2.0.0", ">=1.0.0 <2.0.0", false)
	sat("1.5.0", "*", true)
	sat("1.5.0-alpha", "*", false)
	sat("1.5.0-alpha", ">=1.5.0-alpha <2.0.0", true)
	sat("1.5.0-alpha", ">=1.0.0 <2.0.0", false)
	sat("1.5.0-beta", ">=1.5.0-alpha <2.0.0", true)
	sat("1.6.0-rc", ">=1.5.0-alpha <2.0.0", false)
	sat("1.2.3", "^1.0.0", true)
	sat("2.0.0", "^1.0.0", false)
	sat("0.2.5", "^0.2.0", true)
	sat("0.3.0", "^0.2.0", false)
	sat("0.0.3", "^0.0.3", true)
	sat("0.0.4", "^0.0.3", false)
	sat("1.2.9", "~1.2.3", true)
	sat("1.3.0", "~1.2.3", false)
	sat("1.2.3", "1.2.3 - 2.0.0", true)
	sat("2.0.1", "1.2.3 - 2.0.0", false)
	sat("1.5.0", "1.2.3 - 2.3", true)
	sat("2.4.0", "1.2.3 - 2.3", false)
	sat("1.5.0", ">=1.0.0 <2.0.0 || >=3.0.0 <4.0.0", true)
	sat("3.5.0", ">=1.0.0 <2.0.0 || >=3.0.0 <4.0.0", true)
	sat("2.5.0", ">=1.0.0 <2.0.0 || >=3.0.0 <4.0.0", false)
	sat("1.5.0", "1.x", true)
	sat("2.0.0", "1.x", false)
	sat("1.2.3", "=1.2.3", true)

	// Max / min / sort.
	parsed := []semver.Version{mustV("1.0.0"), mustV("1.2.0"), mustV("1.1.0"), mustV("1.0.0-alpha")}
	mx, _ := semver.Max(parsed)
	check("max is 1.2.0", mx.String() == "1.2.0")
	mn, _ := semver.Min(parsed)
	check("min is 1.0.0-alpha", mn.String() == "1.0.0-alpha")
	semver.Sort(parsed)
	check("sort ascending", parsed[0].String() == "1.0.0-alpha" && parsed[3].String() == "1.2.0")

	if fail > 0 {
		fmt.Printf("smoke-test failed: %d assertion(s)\n", fail)
		return 1
	}
	fmt.Println("smoke-test ok")
	return 0
}
