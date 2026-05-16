package output

import "testing"

func TestFormatWithCommasNegatives(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0"},
		{123, "123"},
		{-123, "-123"},                  // 3 digits — was "-,123"
		{1000, "1,000"},
		{-1000, "-1,000"},
		{-468000000, "-468,000,000"},    // 9 digits — was "-,468,000,000"
		{-1377000000, "-1,377,000,000"}, // 10 digits — was correct
		{-5490462000000, "-5,490,462,000,000"},
	}
	for _, c := range cases {
		got := formatWithCommas(c.in)
		if got != c.want {
			t.Errorf("formatWithCommas(%d) = %q; want %q", c.in, got, c.want)
		}
	}
}
