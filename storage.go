package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	bolt "go.etcd.io/bbolt"
)

const (
	bucketRepos = "repos"
)

var ErrNotFound = errors.New("not found")

type Storage struct {
	db *bolt.DB
}

func NewStorage(db *bolt.DB) *Storage {
	st := &Storage{
		db: db,
	}
	return st
}

// Init creates all required buckets and deletes obsolete buckets.
func (st *Storage) Init() error {
	err := st.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketRepos))
		if err != nil {
			return err
		}
		return nil
	})
	return err
}

func (st *Storage) GetRepo(ctx context.Context, userID, owner, repo string) (*Repo, error) {
	r := new(Repo)
	id := makeRepoID(userID, owner, repo)
	err := st.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketRepos))
		data := b.Get([]byte(id))
		if data == nil {
			return ErrNotFound
		}
		err := json.Unmarshal(data, &r)
		return err
	})
	return r, err
}

type UpdateOrCreateRepoParams struct {
	Name   string
	Owner  string
	Token  string
	UserID string
}

func (st *Storage) UpdateOrCreateRepo(ctx context.Context, arg UpdateOrCreateRepoParams) (*Repo, error) {
	id := makeRepoID(arg.UserID, arg.Owner, arg.Name)
	r := &Repo{
		Name:   arg.Name,
		Owner:  arg.Owner,
		Token:  arg.Token,
		UserID: arg.UserID,
	}
	data, err := json.Marshal(*r)
	if err != nil {
		return nil, err
	}
	err = st.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketRepos))
		err := b.Put([]byte(id), data)
		return err
	})
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (st *Storage) ListReposForUser(ctx context.Context, userID string) ([]Repo, error) {
	repos := make([]Repo, 0)
	err := st.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketRepos))
		err := b.ForEach(func(k, data []byte) error {
			var r Repo
			if err := json.Unmarshal(data, &r); err != nil {
				return err
			}
			if r.UserID != userID {
				return nil
			}
			repos = append(repos, r)
			return nil
		})
		return err
	})
	if err != nil {
		return nil, err
	}
	return repos, nil
}

// DeleteRepos deletes all repos.
// This method is mainly intended for tests.
func (st *Storage) DeleteRepos() error {
	err := st.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketRepos))
		return b.ForEach(func(k, v []byte) error {
			if err := b.Delete(k); err != nil {
				return err
			}
			return nil
		})
	})
	return err
}

func makeRepoID(userID, owner, repo string) string {
	return fmt.Sprintf("%s-%s-%s", userID, owner, repo)
}
