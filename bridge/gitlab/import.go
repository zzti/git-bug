package gitlab

import (
	"fmt"
	"strconv"
	"time"

	"github.com/xanzy/go-gitlab"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/util/git"
	"github.com/MichaelMure/git-bug/util/text"
)

// gitlabImporter implement the Importer interface
type gitlabImporter struct {
	conf core.Configuration

	// iterator
	iterator *iterator

	// number of imported issues
	importedIssues int

	// number of imported identities
	importedIdentities int
}

func (gi *gitlabImporter) Init(conf core.Configuration) error {
	gi.conf = conf
	return nil
}

// ImportAll iterate over all the configured repository issues (notes) and ensure the creation
// of the missing issues / comments / label events / title changes ...
func (gi *gitlabImporter) ImportAll(repo *cache.RepoCache, since time.Time) error {
	gi.iterator = NewIterator(gi.conf[keyProjectID], gi.conf[keyToken], since)

	// Loop over all matching issues
	for gi.iterator.NextIssue() {
		issue := gi.iterator.IssueValue()
		fmt.Printf("importing issue: %v\n", issue.Title)

		// create issue
		b, err := gi.ensureIssue(repo, issue)
		if err != nil {
			return fmt.Errorf("issue creation: %v", err)
		}

		// Loop over all notes
		for gi.iterator.NextNote() {
			note := gi.iterator.NoteValue()
			if err := gi.ensureNote(repo, b, note); err != nil {
				return fmt.Errorf("note creation: %v", err)
			}
		}

		// Loop over all label events
		for gi.iterator.NextLabelEvent() {
			labelEvent := gi.iterator.LabelEventValue()
			if err := gi.ensureLabelEvent(repo, b, labelEvent); err != nil {
				return fmt.Errorf("label event creation: %v", err)
			}
		}

		if err := gi.iterator.Error(); err != nil {
			fmt.Printf("import error: %v\n", err)
			return err
		}

		// commit bug state
		if err := b.CommitAsNeeded(); err != nil {
			return fmt.Errorf("bug commit: %v", err)
		}
	}

	fmt.Printf("Successfully imported %d issues and %d identities from Gitlab\n", gi.importedIssues, gi.importedIdentities)
	return nil
}

func (gi *gitlabImporter) ensureIssue(repo *cache.RepoCache, issue *gitlab.Issue) (*cache.BugCache, error) {
	// ensure issue author
	author, err := gi.ensurePerson(repo, issue.Author.ID)
	if err != nil {
		return nil, err
	}

	// resolve bug
	b, err := repo.ResolveBugCreateMetadata(keyGitlabUrl, issue.WebURL)
	if err != nil && err != bug.ErrBugNotExist {
		return nil, err
	}

	if err == nil {
		return b, nil
	}

	// if bug was never imported
	cleanText, err := text.Cleanup(issue.Description)
	if err != nil {
		return nil, err
	}

	// create bug
	b, _, err = repo.NewBugRaw(
		author,
		issue.CreatedAt.Unix(),
		issue.Title,
		cleanText,
		nil,
		map[string]string{
			core.KeyOrigin:   target,
			keyGitlabId:      parseID(issue.ID),
			keyGitlabUrl:     issue.WebURL,
			keyGitlabProject: gi.conf[keyProjectID],
		},
	)

	if err != nil {
		return nil, err
	}

	// importing a new bug
	gi.importedIssues++

	return b, nil
}

func (gi *gitlabImporter) ensureNote(repo *cache.RepoCache, b *cache.BugCache, note *gitlab.Note) error {
	id := parseID(note.ID)

	hash, errResolve := b.ResolveOperationWithMetadata(keyGitlabId, id)
	if errResolve != nil && errResolve != cache.ErrNoMatchingOp {
		return errResolve
	}

	// ensure issue author
	author, err := gi.ensurePerson(repo, note.Author.ID)
	if err != nil {
		return err
	}

	noteType, body := GetNoteType(note)
	switch noteType {
	case NOTE_CLOSED:
		if errResolve == nil {
			return nil
		}

		_, err = b.CloseRaw(
			author,
			note.CreatedAt.Unix(),
			map[string]string{
				keyGitlabId: id,
			},
		)
		return err

	case NOTE_REOPENED:
		if errResolve == nil {
			return nil
		}

		_, err = b.OpenRaw(
			author,
			note.CreatedAt.Unix(),
			map[string]string{
				keyGitlabId: id,
			},
		)
		return err

	case NOTE_DESCRIPTION_CHANGED:
		issue := gi.iterator.IssueValue()

		firstComment := b.Snapshot().Comments[0]
		// since gitlab doesn't provide the issue history
		// we should check for "changed the description" notes and compare issue texts
		// TODO: Check only one time and ignore next 'description change' within one issue
		if errResolve == cache.ErrNoMatchingOp && issue.Description != firstComment.Message {
			// comment edition
			_, err = b.EditCommentRaw(
				author,
				note.UpdatedAt.Unix(),
				git.Hash(firstComment.Id()),
				issue.Description,
				map[string]string{
					keyGitlabId: id,
				},
			)

			return err
		}

	case NOTE_COMMENT:
		cleanText, err := text.Cleanup(body)
		if err != nil {
			return err
		}

		// if we didn't import the comment
		if errResolve == cache.ErrNoMatchingOp {

			// add comment operation
			_, err = b.AddCommentRaw(
				author,
				note.CreatedAt.Unix(),
				cleanText,
				nil,
				map[string]string{
					keyGitlabId: id,
				},
			)

			return err
		}

		// if comment was already exported

		// search for last comment update
		comment, err := b.Snapshot().SearchComment(hash)
		if err != nil {
			return err
		}

		// compare local bug comment with the new note body
		if comment.Message != cleanText {
			// comment edition
			_, err = b.EditCommentRaw(
				author,
				note.UpdatedAt.Unix(),
				git.Hash(comment.Id()),
				cleanText,
				nil,
			)

			return err
		}

		return nil

	case NOTE_TITLE_CHANGED:
		// title change events are given new notes
		if errResolve == nil {
			return nil
		}

		_, err = b.SetTitleRaw(
			author,
			note.CreatedAt.Unix(),
			body,
			map[string]string{
				keyGitlabId: id,
			},
		)

		return err

	case NOTE_UNKNOWN,
		NOTE_ASSIGNED,
		NOTE_UNASSIGNED,
		NOTE_CHANGED_MILESTONE,
		NOTE_REMOVED_MILESTONE,
		NOTE_CHANGED_DUEDATE,
		NOTE_REMOVED_DUEDATE,
		NOTE_LOCKED,
		NOTE_UNLOCKED:
		return nil

	default:
		panic("unhandled note type")
	}

	return nil
}

func (gi *gitlabImporter) ensureLabelEvent(repo *cache.RepoCache, b *cache.BugCache, labelEvent *gitlab.LabelEvent) error {
	_, err := b.ResolveOperationWithMetadata(keyGitlabId, parseID(labelEvent.ID))
	if err != cache.ErrNoMatchingOp {
		return err
	}

	// ensure issue author
	author, err := gi.ensurePerson(repo, labelEvent.User.ID)
	if err != nil {
		return err
	}

	switch labelEvent.Action {
	case "add":
		_, err = b.ForceChangeLabelsRaw(
			author,
			labelEvent.CreatedAt.Unix(),
			[]string{labelEvent.Label.Name},
			nil,
			map[string]string{
				keyGitlabId: parseID(labelEvent.ID),
			},
		)

	case "remove":
		_, err = b.ForceChangeLabelsRaw(
			author,
			labelEvent.CreatedAt.Unix(),
			nil,
			[]string{labelEvent.Label.Name},
			map[string]string{
				keyGitlabId: parseID(labelEvent.ID),
			},
		)

	default:
		err = fmt.Errorf("unexpected label event action")
	}

	return err
}

func (gi *gitlabImporter) ensurePerson(repo *cache.RepoCache, id int) (*cache.IdentityCache, error) {
	// Look first in the cache
	i, err := repo.ResolveIdentityImmutableMetadata(keyGitlabId, strconv.Itoa(id))
	if err == nil {
		return i, nil
	}
	if _, ok := err.(identity.ErrMultipleMatch); ok {
		return nil, err
	}

	client := buildClient(gi.conf["token"])

	user, _, err := client.Users.GetUser(id)
	if err != nil {
		return nil, err
	}

	// importing a new identity
	gi.importedIdentities++

	return repo.NewIdentityRaw(
		user.Name,
		user.PublicEmail,
		user.Username,
		user.AvatarURL,
		map[string]string{
			// because Gitlab
			keyGitlabId:    strconv.Itoa(id),
			keyGitlabLogin: user.Username,
		},
	)
}

func parseID(id int) string {
	return fmt.Sprintf("%d", id)
}