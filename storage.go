package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	bolt "go.etcd.io/bbolt"
)

const (
	bucketRepos       = "repos"
	bucketReposIndex1 = "reposIndex1"
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
		_, err = tx.CreateBucketIfNotExists([]byte(bucketReposIndex1))
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("Init: %w", err)
	}
	return err
}

// DeleteAll deletes all repos.
// This method is mainly intended for tests.
func (st *Storage) DeleteAll() error {
	err := st.db.Update(func(tx *bolt.Tx) error {
		repos := tx.Bucket([]byte(bucketRepos))
		err := repos.ForEach(func(k, v []byte) error {
			if err := repos.Delete(k); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return err
		}
		index := tx.Bucket([]byte(bucketReposIndex1))
		err = index.ForEach(func(k, v []byte) error {
			if err := repos.Delete(k); err != nil {
				return err
			}
			return nil
		})
		return err
	})
	if err != nil {
		return fmt.Errorf("DeleteAll: %w", err)
	}
	return err
}

func (st *Storage) DeleteRepo(id int) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("DeleteRepo: %d: %w", id, err)
	}
	if id == 0 {
		return wrapErr(ErrInvalidArguments)
	}
	err := st.db.Update(func(tx *bolt.Tx) error {
		// delete repo
		repos := tx.Bucket([]byte(bucketRepos))
		bid := itob(id)
		err := repos.Delete(bid)
		if err != nil {
			return err
		}
		// update index
		index := tx.Bucket([]byte(bucketReposIndex1))
		err = index.ForEach(func(k, v []byte) error {
			if !bytes.Equal(bid, v) {
				return nil
			}
			if err := repos.Delete(k); err != nil {
				return err
			}
			return nil
		})
		return err
	})
	if err != nil {
		return wrapErr(err)
	}
	slog.Info("Repo deleted", "id", id)
	return nil
}

func (st *Storage) GetRepo(id int) (*Repo, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetRepo: %d: %w", id, err)
	}
	if id == 0 {
		return nil, wrapErr(ErrInvalidArguments)
	}
	r := new(Repo)
	err := st.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketRepos))
		data := b.Get(itob(id))
		if data == nil {
			return ErrNotFound
		}
		err := json.Unmarshal(data, &r)
		return err
	})
	if err != nil {
		return nil, wrapErr(err)
	}
	return r, err
}

// func (st *Storage) GetRepo(ctx context.Context, userID, owner, repo string) (*Repo, error) {
// 	r := new(Repo)
// 	err := st.db.View(func(tx *bolt.Tx) error {
// 		repos := tx.Bucket([]byte(bucketRepos))
// 		index := tx.Bucket([]byte(bucketReposIndex1))
// 		bid := index.Get(makeUniqueID(userID, owner, repo))
// 		if bid == nil {
// 			return ErrNotFound
// 		}
// 		data := repos.Get([]byte(bid))
// 		if data == nil {
// 			return ErrNotFound
// 		}
// 		err := json.Unmarshal(data, &r)
// 		return err
// 	})
// 	return r, err
// }

func (st *Storage) ListRepoIDs() ([]int, error) {
	ids := make([]int, 0)
	err := st.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketRepos))
		return b.ForEach(func(_, data []byte) error {
			r := new(Repo)
			if err := json.Unmarshal(data, &r); err != nil {
				return err
			}
			ids = append(ids, r.ID)
			return nil
		})

	})
	if err != nil {
		return nil, fmt.Errorf("ListRepoIDs: %w", err)
	}
	return ids, err
}

// ListReposForUser returns the repos of a user ordered by repo name.
func (st *Storage) ListReposForUser(userID string) ([]*Repo, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ListReposForUser: %s: %w", userID, err)
	}
	if userID == "" {
		return nil, wrapErr(ErrInvalidArguments)
	}
	repos := make([]*Repo, 0)
	err := st.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketRepos))
		err := b.ForEach(func(_, data []byte) error {
			r := new(Repo)
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
		return nil, wrapErr(err)
	}
	if len(repos) > 0 {
		slices.SortFunc(repos, func(a, b *Repo) int {
			return strings.Compare(a.Name(), b.Name())
		})
	}
	return repos, nil
}

type UpdateOrCreateRepoParams struct {
	Repo   string
	Owner  string
	Token  string
	UserID string
}

func (st *Storage) UpdateOrCreateRepo(arg UpdateOrCreateRepoParams) (*Repo, bool, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateOrCreateRepo: %+v: %w", arg, err)
	}
	if arg.Repo == "" || arg.Owner == "" || arg.Token == "" || arg.UserID == "" {
		return nil, false, wrapErr(ErrInvalidArguments)
	}
	r := Repo{
		Repo:   arg.Repo,
		Owner:  arg.Owner,
		Token:  arg.Token,
		UserID: arg.UserID,
	}
	var created bool
	err := st.db.Update(func(tx *bolt.Tx) error {
		repos := tx.Bucket([]byte(bucketRepos))
		index := tx.Bucket([]byte(bucketReposIndex1))
		uniqueID := makeUniqueID(r.UserID, r.Owner, r.Repo)
		bid := index.Get([]byte(uniqueID))
		if bid == nil {
			id, _ := repos.NextSequence()
			r.ID = int(id)
			bid = itob(r.ID)
			err := index.Put(uniqueID, bid)
			if err != nil {
				return err
			}
			created = true
		}
		data, err := json.Marshal(r)
		if err != nil {
			return err
		}
		err = repos.Put(bid, data)
		return err
	})
	if err != nil {
		return nil, false, wrapErr(err)
	}
	slog.Info("Repo updated/created", "id", r.ID, "created", created)
	return &r, created, nil
}

// itob returns an 8-byte big endian representation of v.
func itob(v int) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func makeUniqueID(userID, owner, repo string) []byte {
	return fmt.Appendf(nil, "%s-%s-%s", userID, owner, repo)
}
