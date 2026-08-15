package semver

import "testing"

// Regression probe for BUG1: Min must handle a non-nil empty slice without
// panicking, returning (zero, false) just like it does for nil.
func TestBug1_MinEmptySlice(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Min([]Version{}) panicked: %v", r)
		}
	}()
	_, ok := Min([]Version{})
	if ok {
		t.Fatal("Min([]Version{}) should return false")
	}
}

// Regression probe for BUG2: Sort must be stable — versions with equal
// precedence (differing only in build metadata) must retain input order.
func TestBug2_SortStable(t *testing.T) {
	a, _ := Parse("2.0.0+first")
	b, _ := Parse("2.0.0+second")
	vs := []Version{a, b}
	Sort(vs)
	if vs[0].Build[0] != "first" || vs[1].Build[0] != "second" {
		t.Fatalf("Sort not stable for equal-precedence versions: got [%s, %s]", vs[0], vs[1])
	}
}

// Regression probe for BUG3: a prerelease version must only satisfy a range
// set when a comparator in that set carries a prerelease on the exact same
// major.minor.patch triple, not merely the same major.minor.
func TestBug3_PrereleasePatchFilter(t *testing.T) {
	r, _ := ParseRange(">=1.5.0-alpha <1.5.9")
	v, _ := Parse("1.5.3-beta")
	// v is 1.5.3-beta; the comparator prerelease is on 1.5.0, not 1.5.3.
	// Same major.minor but different patch: must NOT satisfy.
	if r.Satisfies(v) {
		t.Fatal("1.5.3-beta should NOT satisfy '>=1.5.0-alpha <1.5.9' (prerelease filter requires same major.minor.patch)")
	}
}

// Regression probe for BUG4: range expressions containing build metadata with
// a dash in the build tag (e.g. ">=1.0.0+build-info") must parse correctly,
// not confuse the build dash with a prerelease separator.
func TestBug4_RangeWithBuildMeta(t *testing.T) {
	r, err := ParseRange(">=1.0.0+build-info <2.0.0")
	if err != nil {
		t.Fatalf("ParseRange with build-info metadata should succeed: %v", err)
	}
	v, _ := Parse("1.5.0")
	if !r.Satisfies(v) {
		t.Fatal("1.5.0 should satisfy '>=1.0.0+build-info <2.0.0'")
	}
}

// Regression probe for BUG5: Version.String() must join build metadata
// identifiers with dots, not dashes, matching the SemVer 2.0.0 BNF.
func TestBug5_StringBuildSeparator(t *testing.T) {
	v, _ := Parse("1.2.3+build.456")
	got := v.String()
	if got != "1.2.3+build.456" {
		t.Fatalf("String() = %q, want %q", got, "1.2.3+build.456")
	}
}
