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
	return fmt.Sprintf("%s/%s", r.Owner, r.Repo)
}
