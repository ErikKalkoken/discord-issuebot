package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
)

const (
	gitLabBaseURL = "https://gitlab.com/api/v4"
	gitHubBaseURL = "https://api.github.com"
)

var ErrHTTPError = errors.New("HTTP error")

// repoAPI allows creating issues on gitlab and github repos.
type repoAPI struct {
	httpClient http.Client
}

func newRepoAPI() *repoAPI {
	r := &repoAPI{}
	return r
}

func (s repoAPI) checkToken(r *Repo) (int, error) {
	if !r.isValid() {
		return 0, fmt.Errorf("checkToken: %+v: %w", r, ErrInvalidArguments)
	}
	switch r.Vendor {
	case gitLab:
		return s.gitLabCheckToken(r)
	case gitHub:
		return s.gitHubCheckInfo(r)
	}
	return 0, ErrInvalidArguments
}

func (s repoAPI) gitLabCheckToken(r *Repo) (int, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("gitLabProjectInfo: %+v: %w", r, err)
	}
	u, err := url.JoinPath(gitLabBaseURL, "projects", url.PathEscape(r.Owner+"/"+r.Repo))
	if err != nil {
		return 0, wrapErr(err)
	}
	v := url.Values{
		"private_token": {r.Token},
	}
	res, err := s.httpClient.Get(u + "?" + v.Encode())
	if err != nil {
		return 0, wrapErr(err)
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return 0, wrapErr(err)
	}
	if res.StatusCode >= 400 {
		return res.StatusCode, wrapErr(fmt.Errorf("%s: %w", res.Status, ErrHTTPError))
	}
	var info any
	if err := json.Unmarshal(data, &info); err != nil {
		return 0, wrapErr(err)
	}
	slog.Debug("Received response from gitlab for check token", "data", info)
	return res.StatusCode, nil
}

func (s repoAPI) gitHubCheckInfo(arg *Repo) (int, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("gitHubCheckInfo: %+v: %w", arg, err)
	}
	u, err := url.JoinPath(gitHubBaseURL, "repos", arg.Owner, arg.Repo)
	if err != nil {
		return 0, wrapErr(err)
	}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return 0, wrapErr(err)
	}
	req.Header.Add("Accept", "application/vnd.github+json")
	req.Header.Add("Authorization", "Bearer "+arg.Token)
	req.Header.Add("X-GitHub-Api-Version", "2022-11-28")
	res, err := s.httpClient.Do(req)
	if err != nil {
		return 0, wrapErr(err)
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return 0, wrapErr(err)
	}
	if res.StatusCode >= 400 {
		return res.StatusCode, fmt.Errorf("%s: %w", res.Status, ErrHTTPError)
	}
	var info any
	if err := json.Unmarshal(data, &info); err != nil {
		return 0, wrapErr(err)
	}
	slog.Debug("Received response from github for check token", "data", info)
	return res.StatusCode, nil
}

type createIssueParams struct {
	body   string
	labels []string
	title  string
}

func (x createIssueParams) isValid() bool {
	return x.title != "" && x.body != ""
}

func (s repoAPI) createIssue(r *Repo, arg createIssueParams) (string, error) {
	if !r.isValid() || !arg.isValid() {
		return "", fmt.Errorf("createIssue: %+v: %+v: %w", r, arg, ErrInvalidArguments)
	}
	switch r.Vendor {
	case gitLab:
		return s.gitLabCreateIssue(r, arg)
	case gitHub:
		return s.gitHubCreateIssue(r, arg)
	}
	return "", ErrInvalidArguments
}

func (s repoAPI) gitLabCreateIssue(r *Repo, arg createIssueParams) (string, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("gitLabCreateIssue: %+v: %w", arg, err)
	}
	u, err := url.JoinPath(gitLabBaseURL, "projects", url.PathEscape(r.Owner+"/"+r.Repo), "issues")
	if err != nil {
		return "", wrapErr(err)
	}
	v := url.Values{
		"private_token": {r.Token},
		"title":         {arg.title},
		"description":   {arg.body},
	}
	res, err := http.Post(u+"?"+v.Encode(), "application/json", nil)
	if err != nil {
		return "", wrapErr(err)
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return "", wrapErr(err)
	}
	if res.StatusCode >= 400 {
		return "", wrapErr(fmt.Errorf("%s: %w", res.Status, ErrHTTPError))
	}
	var info map[string]any
	if err := json.Unmarshal(data, &info); err != nil {
		return "", wrapErr(err)
	}
	slog.Debug("Received response from gitlab for create issue", "data", info)
	htmlURL, ok := info["web_url"].(string)
	if !ok {
		htmlURL = ""
	}
	return htmlURL, nil
}

func (s repoAPI) gitHubCreateIssue(r *Repo, arg createIssueParams) (string, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("gitHubCreateIssue: %+v: %w", arg, err)
	}
	u, err := url.JoinPath(gitHubBaseURL, "repos", r.Owner, r.Repo, "issues")
	if err != nil {
		return "", wrapErr(err)
	}
	body, err := json.Marshal(map[string]any{
		"title": arg.title,
		"body":  arg.body,
	})
	if err != nil {
		return "", wrapErr(err)
	}
	req, err := http.NewRequest("POST", u, bytes.NewBuffer(body))
	if err != nil {
		return "", wrapErr(err)
	}
	req.Header.Add("Accept", "application/vnd.github+json")
	req.Header.Add("Authorization", "Bearer "+r.Token)
	req.Header.Add("X-GitHub-Api-Version", "2022-11-28")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", wrapErr(err)
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return "", wrapErr(err)
	}
	if res.StatusCode >= 400 {
		return "", wrapErr(fmt.Errorf("%s: %w", res.Status, ErrHTTPError))
	}
	var info map[string]any
	if err := json.Unmarshal(data, &info); err != nil {
		return "", wrapErr(err)
	}
	slog.Debug("Received response from github for create issue", "data", info)
	htmlURL, ok := info["htmlURL"].(string)
	if !ok {
		htmlURL = ""
	}
	return htmlURL, nil
}
