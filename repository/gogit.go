package repository

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/MichaelMure/git-bug/util/git"
	gogit "gopkg.in/src-d/go-git.v4"
)

type GoGitRepo struct {
	path string
	r    *gogit.Repository
}

func NewGoGitRepo(path string, witnesser Witnesser) (*GoGitRepo, error) {
	var err error
	if path, err = filepath.Abs(path); err != nil {
		return nil, err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, err
	}

	for {
		_, err := os.Stat(filepath.Join(path, ".git"))
		if err == nil {
			// no error; stop
			break
		}
		if !os.IsNotExist(err) {
			// unknown error; stop
			return nil, err
		}

		fmt.Println(err)

		// try its parent as long as we haven't reached
		// the root dir
		if dir := filepath.Dir(path); dir != path {
			path = dir
			continue
		}
	}

	r, err := gogit.PlainOpen(path)

	if err != nil {
		if err == gogit.ErrRepositoryNotExists {
			return nil, ErrNotARepo
		}
		return nil, err
	}

	return &GoGitRepo{
		path: path,
		r:    r,
	}, nil
}

func (repo *GoGitRepo) GetPath() string {
	return repo.path
}

func (repo *GoGitRepo) GetUserName() (string, error) {
	config, err := repo.r.Config()
	if err != nil {
		return "", err
	}

	// global config is not supported by go-git

	fmt.Println(config)

	panic("implement me")
}

func (repo *GoGitRepo) GetUserEmail() (string, error) {
	// global config is not supported by go-git
	panic("implement me")
}

func (repo *GoGitRepo) GetCoreEditor() (string, error) {
	// global config is not supported by go-git
	panic("implement me")
}

func (repo *GoGitRepo) StoreConfig(key string, value string) error {
	panic("implement me")
}

func (repo *GoGitRepo) ReadConfigs(keyPrefix string) (map[string]string, error) {
	panic("implement me")
}

func (repo *GoGitRepo) RmConfigs(keyPrefix string) error {
	panic("implement me")
}

func (repo *GoGitRepo) FetchRefs(remote string, refSpec string) (string, error) {
	panic("implement me")
}

func (repo *GoGitRepo) PushRefs(remote string, refSpec string) (string, error) {
	panic("implement me")
}

func (repo *GoGitRepo) StoreData(data []byte) (git.Hash, error) {
	panic("implement me")
}

func (repo *GoGitRepo) ReadData(hash git.Hash) ([]byte, error) {
	panic("implement me")
}

func (repo *GoGitRepo) StoreTree(mapping []TreeEntry) (git.Hash, error) {
	panic("implement me")
}

func (repo *GoGitRepo) StoreCommit(treeHash git.Hash) (git.Hash, error) {
	panic("implement me")
}

func (repo *GoGitRepo) StoreCommitWithParent(treeHash git.Hash, parent git.Hash) (git.Hash, error) {
	panic("implement me")
}

func (repo *GoGitRepo) UpdateRef(ref string, hash git.Hash) error {
	panic("implement me")
}

func (repo *GoGitRepo) ListRefs(refspec string) ([]string, error) {
	panic("implement me")
}

func (repo *GoGitRepo) RefExist(ref string) (bool, error) {
	panic("implement me")
}

func (repo *GoGitRepo) CopyRef(source string, dest string) error {
	panic("implement me")
}

func (repo *GoGitRepo) ListCommits(ref string) ([]git.Hash, error) {
	panic("implement me")
}

func (repo *GoGitRepo) ListEntries(hash git.Hash) ([]TreeEntry, error) {
	panic("implement me")
}

func (repo *GoGitRepo) FindCommonAncestor(hash1 git.Hash, hash2 git.Hash) (git.Hash, error) {
	panic("implement me")
}

func (repo *GoGitRepo) GetTreeHash(commit git.Hash) (git.Hash, error) {
	panic("implement me")
}
