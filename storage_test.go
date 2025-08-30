package main

import (
	"fmt"
	"math/rand/v2"
	"path/filepath"
	"testing"

	"github.com/icrowley/fake"
	"github.com/stretchr/testify/assert"
	bolt "go.etcd.io/bbolt"
)

func TestStorage(t *testing.T) {
	p := filepath.Join(t.TempDir(), "test.db")
	db, err := bolt.Open(p, 0600, nil)
	if err != nil {
		t.Fatalf("Failed to open DB: %s", err)
	}
	defer db.Close()
	st := NewStorage(db)
	if err = st.Init(); err != nil {
		t.Fatal(err)
	}
	t.Run("can create new repo", func(t *testing.T) {
		if err := st.DeleteAll(); err != nil {
			t.Fatal(err)
		}
		r1, created, err := st.UpdateOrCreateRepo(UpdateOrCreateRepoParams{
			Owner:  "owner",
			Repo:   "repo",
			UserID: "user",
			Token:  "token",
			Vendor: gitHub,
		})
		if assert.NoError(t, err) {
			assert.True(t, created)
			assert.Equal(t, "owner", r1.Owner)
			assert.Equal(t, "repo", r1.Repo)
			assert.Equal(t, "user", r1.UserID)
			assert.Equal(t, "token", r1.Token)
			assert.Equal(t, gitHub, r1.Vendor)
			r2, err := st.GetRepo(r1.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, r1, r2)
			}
		}
	})
	t.Run("can update existing repo", func(t *testing.T) {
		if err := st.DeleteAll(); err != nil {
			t.Fatal(err)
		}
		r2 := createRepo(t, st)
		r1, created, err := st.UpdateOrCreateRepo(UpdateOrCreateRepoParams{
			Owner:  r2.Owner,
			Repo:   r2.Repo,
			UserID: r2.UserID,
			Token:  "token",
			Vendor: gitHub,
		})
		if assert.NoError(t, err) {
			assert.False(t, created)
			assert.Equal(t, "token", r1.Token)
		}
	})
	t.Run("can get a repo", func(t *testing.T) {
		if err := st.DeleteAll(); err != nil {
			t.Fatal(err)
		}
		r1 := createRepo(t, st)
		r2, err := st.GetRepo(r1.ID)
		if assert.NoError(t, err) {
			assert.Equal(t, r1, r2)
		}
	})
	t.Run("should return when not found error", func(t *testing.T) {
		if err := st.DeleteAll(); err != nil {
			t.Fatal(err)
		}
		_, err := st.GetRepo(42)
		assert.ErrorIs(t, err, ErrNotFound)
	})
	t.Run("can list repo IDs", func(t *testing.T) {
		if err := st.DeleteAll(); err != nil {
			t.Fatal(err)
		}
		r1 := createRepo(t, st)
		r2 := createRepo(t, st)
		if assert.NoError(t, err) {
			got, err := st.ListRepoIDs()
			if assert.NoError(t, err) {
				want := []int{r2.ID, r1.ID}
				assert.ElementsMatch(t, want, got)
			}
		}
	})
	t.Run("can return ordered list of repos for user", func(t *testing.T) {
		if err := st.DeleteAll(); err != nil {
			t.Fatal(err)
		}
		user1 := "user1"
		r1 := createRepo(t, st, UpdateOrCreateRepoParams{
			UserID: user1,
			Owner:  "alpha",
			Repo:   "two",
		})
		r2 := createRepo(t, st, UpdateOrCreateRepoParams{
			UserID: user1,
			Owner:  "alpha",
			Repo:   "first",
		})
		createRepo(t, st)
		if assert.NoError(t, err) {
			xx, err := st.ListReposForUser(user1)
			if assert.NoError(t, err) {
				want := []int{r2.ID, r1.ID}
				var got []int
				for _, x := range xx {
					got = append(got, x.ID)
				}
				assert.Equal(t, want, got)
			}
		}
	})

	t.Run("can count repos for a user", func(t *testing.T) {
		if err := st.DeleteAll(); err != nil {
			t.Fatal(err)
		}
		user1 := "user1"
		createRepo(t, st, UpdateOrCreateRepoParams{
			UserID: user1,
			Owner:  "alpha",
			Repo:   "two",
		})
		createRepo(t, st, UpdateOrCreateRepoParams{
			UserID: user1,
			Owner:  "alpha",
			Repo:   "first",
		})
		createRepo(t, st)
		if assert.NoError(t, err) {
			got, err := st.CountReposForUser(user1)
			if assert.NoError(t, err) {
				assert.Equal(t, 2, got)
			}
		}
	})

	t.Run("can delete a repo", func(t *testing.T) {
		if err := st.DeleteAll(); err != nil {
			t.Fatal(err)
		}
		r1 := createRepo(t, st)
		r2 := createRepo(t, st)
		if assert.NoError(t, err) {
			err := st.DeleteRepo(r2.ID)
			if assert.NoError(t, err) {
				want := []int{r1.ID}
				got, err := st.ListRepoIDs()
				if err != nil {
					t.Fatal(err)
				}
				assert.ElementsMatch(t, want, got)
			}
		}
	})
}

func createRepo(t *testing.T, st *Storage, args ...UpdateOrCreateRepoParams) *Repo {
	var arg UpdateOrCreateRepoParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.UserID == "" {
		arg.UserID = fmt.Sprintf("%s%d", fake.UserName(), rand.IntN(10_000))
	}
	if arg.Owner == "" {
		arg.Owner = arg.UserID
	}
	if arg.Repo == "" {
		arg.Repo = fmt.Sprintf("%s%d", fake.Color(), rand.IntN(10_000))
	}
	if arg.Token == "" {
		arg.Token = fake.SimplePassword()
	}
	if arg.Vendor == "" {
		arg.Vendor = gitHub
	}
	r, _, err := st.UpdateOrCreateRepo(arg)
	if err != nil {
		t.Fatal(err)
		return nil
	}
	return r
}
