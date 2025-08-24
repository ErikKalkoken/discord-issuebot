package main

import (
	"context"

	"github.com/google/go-github/v74/github"
)

type createGithubIssueParams struct {
	owner  string
	repo   string
	token  string
	title  string
	body   string
	labels []string
}

func createGithubIssue(arg createGithubIssueParams) (*github.Issue, error) {
	client := github.NewClient(nil).WithAuthToken(arg.token)
	issue, _, err := client.Issues.Create(context.Background(), arg.owner, arg.repo, &github.IssueRequest{
		Title:  &arg.title,
		Body:   &arg.body,
		Labels: &arg.labels,
	})
	return issue, err
}
