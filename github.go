package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/go-github/v74/github"
)

var errInvalid = errors.New("invalid")

type gitHubCreateIssueParams struct {
	owner  string
	repo   string
	token  string
	title  string
	body   string
	labels []string
}

func gitHubCreateIssue(ctx context.Context, arg gitHubCreateIssueParams) (*github.Issue, error) {
	if arg.owner == "" || arg.repo == "" || arg.token == "" || arg.title == "" {
		return nil, fmt.Errorf("%+v: %w", arg, errInvalid)
	}
	client := github.NewClient(nil).WithAuthToken(arg.token)
	issue, _, err := client.Issues.Create(ctx, arg.owner, arg.repo, &github.IssueRequest{
		Title:  &arg.title,
		Body:   &arg.body,
		Labels: &arg.labels,
	})
	if err != nil {
		return nil, err
	}
	slog.Info("Issue created", "repo", fmt.Sprintf("%s/%s", arg.owner, arg.repo), "id", optionalValue(issue.Number), "title", optionalValue(issue.Title))
	return issue, nil
}

type githubGetRepoInfoParams struct {
	owner string
	repo  string
	token string
}

func githubGetRepoInfo(ctx context.Context, arg githubGetRepoInfoParams) (*github.Repository, *github.Response, error) {
	if arg.owner == "" || arg.repo == "" || arg.token == "" {
		return nil, nil, fmt.Errorf("%+v: %w", arg, errInvalid)
	}
	client := github.NewClient(nil).WithAuthToken(arg.token)
	info, r, err := client.Repositories.Get(ctx, arg.owner, arg.repo)
	if err != nil {
		return nil, r, err
	}
	slog.Info("Received repository info from github", "name", optionalValue(info.FullName))
	return info, r, err
}

// optionalValue return the value of a pointer or it's zero value if the pointer is nil.
func optionalValue[T any](s *T) T {
	var z T
	if s == nil {
		return z
	}
	return *s
}
