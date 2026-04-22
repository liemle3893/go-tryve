package sandbox

import "testing"

// releaseVersionRE is indirectly tested here to pin the format.
func TestReleaseVersionRE(t *testing.T) {
	cases := map[string]bool{
		"v1.2.3":        true,
		"v10.20.30":     true,
		"v1.2.3-rc1":    false,
		"1.2.3":         false,
		"dev":           false,
		"":              false,
	}
	for in, want := range cases {
		if got := releaseVersionRE.MatchString(in); got != want {
			t.Errorf("%q: got %v want %v", in, got, want)
		}
	}
}
