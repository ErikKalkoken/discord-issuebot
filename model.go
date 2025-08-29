package main

import (
	"fmt"
	"net/url"
)

type Vendor uint

const (
	undefined Vendor = iota
	gitHub
	gitLab
)

func (v Vendor) Host() string {
	switch v {
	case gitHub:
		return "github.com"
	case gitLab:
		return "gitlab.com"
	}
	return ""
}

type Repo struct {
	ID     int
	Repo   string
	Owner  string
	Token  string
	UserID string
	Vendor Vendor
}

func (r Repo) isValid() bool {
	return r.Owner != "" && r.Repo != "" && r.Token != "" && r.Vendor != undefined && r.UserID != ""
}

func (r Repo) Name() string {
	s, _ := url.JoinPath(r.Vendor.Host(), r.Owner, r.Repo)
	return s
}

func (r Repo) URL() string {
	s, _ := url.JoinPath(r.Vendor.Host(), r.Owner, r.Repo)
	return fmt.Sprintf("https://%s", s)
}
