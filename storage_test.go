package main

import (
	"context"
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
	ctx := context.Background()
	t.Run("can create new repo", func(t *testing.T) {
		if err := st.DeleteRepos(); err != nil {
			t.Fatal(err)
		}
		r1, err := st.UpdateOrCreateRepo(ctx, UpdateOrCreateRepoParams{
			Owner:  "owner",
			Name:   "repo",
			UserID: "user",
			Token:  "token",
		})
		if assert.NoError(t, err) {
			assert.Equal(t, "owner", r1.Owner)
			assert.Equal(t, "repo", r1.Name)
			assert.Equal(t, "user", r1.UserID)
			assert.Equal(t, "token", r1.Token)
			r2, err := st.GetRepo(ctx, "user", "owner", "repo")
			if assert.NoError(t, err) {
				assert.Equal(t, r1, r2)
			}
		}
	})
	t.Run("should return not found error", func(t *testing.T) {
		if err := st.DeleteRepos(); err != nil {
			t.Fatal(err)
		}
		_, err := st.GetRepo(ctx, "user", "owner", "repo")
		assert.ErrorIs(t, err, ErrNotFound)
	})
	t.Run("can list repos for user", func(t *testing.T) {
		if err := st.DeleteRepos(); err != nil {
			t.Fatal(err)
		}
		user1 := "user1"
		r1 := createRepo(t, st, UpdateOrCreateRepoParams{UserID: user1})
		r2 := createRepo(t, st, UpdateOrCreateRepoParams{UserID: user1})
		createRepo(t, st)
		if assert.NoError(t, err) {
			xx, err := st.ListReposForUser(ctx, user1)
			if assert.NoError(t, err) {
				want := []string{r1.Token, r2.Token}
				var got []string
				for _, x := range xx {
					got = append(got, x.Token)
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
	if arg.Name == "" {
		arg.Name = fmt.Sprintf("%s%d", fake.Color(), rand.IntN(10_000))
	}
	if arg.Token == "" {
		arg.Token = fake.SimplePassword()
	}
	r, err := st.UpdateOrCreateRepo(context.Background(), arg)
	if err != nil {
		t.Fatal(err)
		return nil
	}
	return r
}
