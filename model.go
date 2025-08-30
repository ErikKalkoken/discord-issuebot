package main

import (
	"fmt"
	"net/url"
)

// Vendor represents a vendor that provides git repositories like GitHub.
type Vendor string

const (
	gitHub Vendor = "github"
	gitLab Vendor = "gitlab"
)

func (v Vendor) String() string {
	return string(v)
}

func (v Vendor) Host() string {
	switch v {
	case gitHub:
		return "github.com"
	case gitLab:
		return "gitlab.com"
	}
	return ""
}

// Repo represents a repository for creating issues.
type Repo struct {
	ID     int    `json:"id"`
	Repo   string `json:"repo"`
	Owner  string `json:"owner"`
	Token  string `json:"token"`
	UserID string `json:"user_id"` // Discord user ID
	Vendor Vendor `json:"vendor"`
}

func (r Repo) isValid() bool {
	return r.Owner != "" && r.Repo != "" && r.Token != "" && r.Vendor != "" && r.UserID != ""
}

func (r Repo) Name() string {
	s, _ := url.JoinPath(r.Vendor.Host(), r.Owner, r.Repo)
	return s
}

func (r Repo) URL() string {
	s, _ := url.JoinPath(r.Vendor.Host(), r.Owner, r.Repo)
	return fmt.Sprintf("https://%s", s)
}
