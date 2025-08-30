package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseURL(t *testing.T) {
	cases := []struct {
		name    string
		rawURL  string
		owner   string
		repo    string
		vendor  Vendor
		isValid bool
	}{
		{"github happy case", "https://github.com/ErikKalkoken/evebuddy", "ErikKalkoken", "evebuddy", gitHub, true},
		{"gitlab happy case", "https://gitlab.com/ErikKalkoken/evebuddy", "ErikKalkoken", "evebuddy", gitLab, true},
		{"invalid host", "https://bitbucket.com/ErikKalkoken/evebuddy", "", "", "", false},
		{"path too short", "https://gitlab.com/ErikKalkoken", "", "", gitHub, false},
		{"path too long", "https://gitlab.com/ErikKalkoken/x/y", "", "", gitHub, false},
		{"invalid URL", "xyz", "", "", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			owner, repo, vendor, err := parseRepoURL(tc.rawURL)
			if tc.isValid {
				assert.Equal(t, tc.owner, owner)
				assert.Equal(t, tc.repo, repo)
				assert.Equal(t, tc.vendor, vendor)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
