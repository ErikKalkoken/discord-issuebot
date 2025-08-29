package main

import (
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestGitHub(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	t.Run("can check token", func(t *testing.T) {
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://api.github.com/repos/owner/repo",
			func(req *http.Request) (*http.Response, error) {
				if req.Header.Get("Authorization") != "Bearer token" {
					return httpmock.NewStringResponse(401, ""), nil
				}
				if req.Header.Get("X-GitHub-Api-Version") != "2022-11-28" {
					return httpmock.NewStringResponse(500, ""), nil
				}
				return httpmock.NewJsonResponse(200, map[string]any{
					"id":   "123",
					"name": "name",
				})
			})
		a := newRepoAPI()
		got, err := a.checkToken(&Repo{
			Owner:  "owner",
			Repo:   "repo",
			Token:  "token",
			Vendor: gitHub,
			UserID: "user",
		})
		if assert.NoError(t, err) {
			assert.Equal(t, http.StatusOK, got)
		}
	})

	t.Run("can create issue", func(t *testing.T) {
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			"https://api.github.com/repos/owner/repo/issues",
			func(req *http.Request) (*http.Response, error) {
				if req.Header.Get("Authorization") != "Bearer token" {
					return httpmock.NewStringResponse(401, ""), nil
				}
				if req.Header.Get("X-GitHub-Api-Version") != "2022-11-28" {
					return httpmock.NewStringResponse(500, ""), nil
				}
				return httpmock.NewJsonResponse(200, map[string]any{
					"id":      "123",
					"htmlURL": "url",
				})
			})
		a := newRepoAPI()
		got, err := a.createIssue(&Repo{
			Owner:  "owner",
			Repo:   "repo",
			Token:  "token",
			Vendor: gitHub,
			UserID: "user",
		}, createIssueParams{
			title: "title",
			body:  "body",
		})
		if assert.NoError(t, err) {
			assert.Equal(t, "url", got)
		}
	})
}

func TestGitLab(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	t.Run("can check token", func(t *testing.T) {
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://gitlab.com/api/v4/projects/owner%2Frepo",
			func(req *http.Request) (*http.Response, error) {
				v := req.URL.Query()
				if v.Get("private_token") != "token" {
					return httpmock.NewStringResponse(401, ""), nil
				}
				return httpmock.NewJsonResponse(200, map[string]any{
					"id":   "123",
					"name": "name",
				})
			})
		a := newRepoAPI()
		got, err := a.checkToken(&Repo{
			Owner:  "owner",
			Repo:   "repo",
			Token:  "token",
			Vendor: gitLab,
			UserID: "user",
		})
		if assert.NoError(t, err) {
			assert.Equal(t, http.StatusOK, got)
		}
	})

	t.Run("can create issue", func(t *testing.T) {
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			"https://gitlab.com/api/v4/projects/owner%2Frepo/issues",
			func(req *http.Request) (*http.Response, error) {
				v := req.URL.Query()
				if v.Get("private_token") != "token" {
					return httpmock.NewStringResponse(401, ""), nil
				}
				return httpmock.NewJsonResponse(200, map[string]any{
					"id":      "123",
					"web_url": "url",
				})
			})
		a := newRepoAPI()
		got, err := a.createIssue(&Repo{
			Owner:  "owner",
			Repo:   "repo",
			Token:  "token",
			Vendor: gitLab,
			UserID: "user",
		}, createIssueParams{
			title: "title",
			body:  "body",
		})
		if assert.NoError(t, err) {
			assert.Equal(t, "url", got)
		}
	})
}
