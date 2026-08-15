package semver

import "testing"

func sliceEq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestParse(t *testing.T) {
	valid := map[string]Version{
		"1.2.3":             {Major: 1, Minor: 2, Patch: 3},
		"0.0.0":             {},
		"1.0.0-alpha":       {Major: 1, Prerelease: []string{"alpha"}},
		"1.0.0-alpha.1":     {Major: 1, Prerelease: []string{"alpha", "1"}},
		"1.0.0-alpha.beta":  {Major: 1, Prerelease: []string{"alpha", "beta"}},
		"1.0.0+build.123":   {Major: 1, Build: []string{"build", "123"}},
		"1.0.0-alpha+build": {Major: 1, Prerelease: []string{"alpha"}, Build: []string{"build"}},
		"10.20.30":          {Major: 10, Minor: 20, Patch: 30},
		"1.2.3-0":           {Major: 1, Minor: 2, Patch: 3, Prerelease: []string{"0"}},
	}
	for s, want := range valid {
		got, err := Parse(s)
		if err != nil {
			t.Errorf("Parse(%q) error: %v", s, err)
			continue
		}
		if got.Major != want.Major || got.Minor != want.Minor || got.Patch != want.Patch {
			t.Errorf("Parse(%q) core = %d.%d.%d, want %d.%d.%d", s, got.Major, got.Minor, got.Patch, want.Major, want.Minor, want.Patch)
		}
		if !sliceEq(got.Prerelease, want.Prerelease) {
			t.Errorf("Parse(%q) prerelease = %v, want %v", s, got.Prerelease, want.Prerelease)
		}
		if !sliceEq(got.Build, want.Build) {
			t.Errorf("Parse(%q) build = %v, want %v", s, got.Build, want.Build)
		}
	}
}

func TestParseInvalid(t *testing.T) {
	invalid := []string{
		"", "1", "1.2", "1.2.3.4", "01.2.3", "1.02.3", "1.2.03",
		"v1.2.3", "1.2.3-", "1.2.3+", "1.2.3-alpha..1", "1.2.3-01",
		"1.2.3-alpha.01", "1.2.3-alpha.", "abc", "1.2.x", "1.2.3-α",
	}
	for _, s := range invalid {
		if _, err := Parse(s); err == nil {
			t.Errorf("Parse(%q) expected error, got nil", s)
		}
	}
}

func TestCompare(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "2.0.0", -1},
		{"2.0.0", "1.0.0", 1},
		{"2.1.0", "2.0.0", 1},
		{"2.1.1", "2.1.0", 1},
		{"1.0.0", "1.0.0", 0},
		{"1.0.0+build1", "1.0.0+build2", 0},
		{"1.0.0-alpha", "1.0.0", -1},
		{"1.0.0", "1.0.0-alpha", 1},
		{"1.0.0-alpha", "1.0.0-alpha.1", -1},
		{"1.0.0-alpha.1", "1.0.0-alpha.beta", -1},
		{"1.0.0-alpha.beta", "1.0.0-beta", -1},
		{"1.0.0-beta", "1.0.0-beta.2", -1},
		{"1.0.0-beta.2", "1.0.0-beta.11", -1},
		{"1.0.0-beta.11", "1.0.0-rc.1", -1},
		{"1.0.0-rc.1", "1.0.0", -1},
		{"1.0.0-1", "1.0.0-alpha", -1},
		{"1.0.0-9", "1.0.0-10", -1},
	}
	for _, c := range cases {
		a, errA := Parse(c.a)
		b, errB := Parse(c.b)
		if errA != nil || errB != nil {
			t.Errorf("Parse(%q)=%v Parse(%q)=%v", c.a, errA, c.b, errB)
			continue
		}
		if got := a.Compare(b); got != c.want {
			t.Errorf("Compare(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestSatisfies(t *testing.T) {
	cases := []struct {
		ver, rng string
		want     bool
	}{
		{"1.2.3", "1.2.3", true},
		{"1.2.4", "1.2.3", false},
		{"1.5.0", ">=1.0.0 <2.0.0", true},
		{"2.0.0", ">=1.0.0 <2.0.0", false},
		{"1.5.0", "*", true},
		{"1.5.0-alpha", "*", false},
		{"1.5.0-alpha", ">=1.5.0-alpha <2.0.0", true},
		{"1.5.0-alpha", ">=1.0.0 <2.0.0", false},
		{"1.5.0-beta", ">=1.5.0-alpha <2.0.0", true},
		{"1.6.0-rc", ">=1.5.0-alpha <2.0.0", false},
		{"1.2.3", "^1.0.0", true},
		{"2.0.0", "^1.0.0", false},
		{"0.2.5", "^0.2.0", true},
		{"0.3.0", "^0.2.0", false},
		{"0.0.3", "^0.0.3", true},
		{"0.0.4", "^0.0.3", false},
		{"0.0.0", "^0.0", true},
		{"0.0.9", "^0.0", true},
		{"0.1.0", "^0.0", false},
		{"1.0.0", "^1", true},
		{"2.0.0", "^1", false},
		{"1.2.9", "~1.2.3", true},
		{"1.3.0", "~1.2.3", false},
		{"1.2.3", "~1.2", true},
		{"1.3.0", "~1.2", false},
		{"1.2.3", "~1", true},
		{"2.0.0", "~1", false},
		{"1.2.3", "1.2.3 - 2.0.0", true},
		{"2.0.1", "1.2.3 - 2.0.0", false},
		{"1.5.0", "1.2.3 - 2.3", true},
		{"2.4.0", "1.2.3 - 2.3", false},
		{"2.9.0", "1.2.3 - 2", true},
		{"3.0.0", "1.2.3 - 2", false},
		{"1.5.0", ">=1.0.0 <2.0.0 || >=3.0.0 <4.0.0", true},
		{"3.5.0", ">=1.0.0 <2.0.0 || >=3.0.0 <4.0.0", true},
		{"2.5.0", ">=1.0.0 <2.0.0 || >=3.0.0 <4.0.0", false},
		{"1.5.0", "1.x", true},
		{"2.0.0", "1.x", false},
		{"1.2.0", "1.2.x", true},
		{"1.3.0", "1.2.x", false},
		{"1.0.0", "1", true},
		{"2.0.0", "1", false},
		{"1.2.0", "1.2", true},
		{"1.3.0", "1.2", false},
		{"1.2.3", "=1.2.3", true},
		{"1.2.3-alpha", "=1.2.3", false},
		{"1.2.3-beta", "=1.2.3-beta", true},
		{"1.3.0", ">1.2", true},
		{"1.2.5", ">1.2", false},
		{"1.2.0", ">1.2", false},
		{"2.0.0", ">1", true},
		{"1.5.0", ">1", false},
		{"1.2.4", ">1.2.3", true},
		{"1.2.3", ">1.2.3", false},
		{"1.2.0", "<=1.2", true},
		{"1.2.9", "<=1.2", true},
		{"1.3.0", "<=1.2", false},
		{"1.2.5", "<=1.2.5", true},
		{"1.5.0", "<=1.2.5", false},
		{"1.1.0", "<1.2", true},
		{"1.2.0", "<1.2", false},
		{"0.9.0", "<1", true},
		{"1.0.0", "<1", false},
		{"1.2.3", ">=1.2", true},
		{"1.1.0", ">=1.2", false},
	}
	for _, c := range cases {
		v, errV := Parse(c.ver)
		r, errR := ParseRange(c.rng)
		if errV != nil {
			t.Errorf("Parse(%q) error: %v", c.ver, errV)
			continue
		}
		if errR != nil {
			t.Errorf("ParseRange(%q) error: %v", c.rng, errR)
			continue
		}
		if got := r.Satisfies(v); got != c.want {
			t.Errorf("Satisfies(%q, %q) = %v, want %v", c.ver, c.rng, got, c.want)
		}
	}
}

func TestParseRangeInvalid(t *testing.T) {
	bad := []string{
		"1.2.3 -", "1.2.3 - 2.0.0 - 3.0.0", "=1.2.3-01", ">=01.0.0", "1.2.3.4",
	}
	for _, s := range bad {
		if _, err := ParseRange(s); err == nil {
			t.Errorf("ParseRange(%q) expected error", s)
		}
	}
}

func TestMaxMinSort(t *testing.T) {
	vs := []string{"1.2.0", "1.0.0", "1.1.0", "1.0.0-alpha"}
	parsed := make([]Version, len(vs))
	for i, s := range vs {
		v, err := Parse(s)
		if err != nil {
			t.Fatalf("Parse(%q) error: %v", s, err)
		}
		parsed[i] = v
	}
	if mx, ok := Max(parsed); !ok || mx.String() != "1.2.0" {
		t.Errorf("Max = %v ok=%v, want 1.2.0", mx, ok)
	}
	if mn, ok := Min(parsed); !ok || mn.String() != "1.0.0-alpha" {
		t.Errorf("Min = %v ok=%v, want 1.0.0-alpha", mn, ok)
	}
	Sort(parsed)
	want := []string{"1.0.0-alpha", "1.0.0", "1.1.0", "1.2.0"}
	for i, v := range parsed {
		if v.String() != want[i] {
			t.Errorf("Sort[%d] = %s, want %s", i, v.String(), want[i])
		}
	}
	if _, ok := Max(nil); ok {
		t.Error("Max(nil) ok should be false")
	}
}
