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
		isValid bool
	}{
		{"happy case", "https://github.com/ErikKalkoken/evebuddy", "ErikKalkoken", "evebuddy", true},
		{"wrong host", "https://gitlab.com/ErikKalkoken/evebuddy", "", "", false},
		{"path too short", "https://gitlab.com/ErikKalkoken", "", "", false},
		{"path too long", "https://gitlab.com/ErikKalkoken/x/y", "", "", false},
		{"invalid URL", "xyz", "", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			owner, repo, err := parseRepoURL(tc.rawURL)
			if tc.isValid {
				assert.Equal(t, tc.owner, owner)
				assert.Equal(t, tc.repo, repo)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
