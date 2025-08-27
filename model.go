package main

import "fmt"

type Repo struct {
	ID     int
	Repo   string
	Owner  string
	Token  string
	UserID string
}

func (r Repo) Name() string {
	return fmt.Sprintf("github.com/%s/%s", r.Owner, r.Repo)
}

func (r Repo) URL() string {
	return fmt.Sprintf("https://github.com/%s/%s", r.Owner, r.Repo)
}
