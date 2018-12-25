package repository

import (
	"fmt"
	"testing"

	"gotest.tools/assert"
)

func TestNewGoGitRepo(t *testing.T) {
	repo, err := NewGoGitRepo("/tmp/blah/pwet", func(repo ClockedRepo) error {
		return nil
	})

	assert.NilError(t, err)

	fmt.Println(repo.GetPath())

	repo.GetUserName()
}
