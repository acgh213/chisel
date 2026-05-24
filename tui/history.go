package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// ---------------------------------------------------------------------------
// revision backend interface
// ---------------------------------------------------------------------------

// Revision represents one saved snapshot of a file.
type Revision struct {
	Hash      string
	Timestamp time.Time
	Message   string
}

// RevisionBackend abstracts the VCS layer so git can be swapped for jj later.
type RevisionBackend interface {
	// Save commits the current state of path with the given message.
	Save(path string, message string) error

	// Log returns the revision history for a path (newest first).
	Log(path string) ([]Revision, error)

	// Diff returns a unified diff between two revisions of a path.
	Diff(path string, rev1, rev2 string) (string, error)

	// Restore restores a path to the content at the given revision.
	Restore(path string, rev string) (string, error)
}

// ---------------------------------------------------------------------------
// git backend
// ---------------------------------------------------------------------------

// GitBackend implements RevisionBackend using go-git on the project repo.
type GitBackend struct {
	repo *git.Repository
	dir  string // project root
}

// NewGitBackend opens the git repository at projectDir.
func NewGitBackend(projectDir string) (*GitBackend, error) {
	repo, err := git.PlainOpen(projectDir)
	if err != nil {
		return nil, fmt.Errorf("opening git repo: %w", err)
	}
	return &GitBackend{repo: repo, dir: projectDir}, nil
}

// Save stages and commits a single file with a structured message.
func (gb *GitBackend) Save(path string, message string) error {
	wt, err := gb.repo.Worktree()
	if err != nil {
		return fmt.Errorf("worktree: %w", err)
	}

	// Stage only the changed file.
	rel, err := filepath.Rel(gb.dir, path)
	if err != nil {
		rel = path
	}
	rel = filepath.ToSlash(rel)

	if _, err := wt.Add(rel); err != nil {
		return fmt.Errorf("staging %s: %w", rel, err)
	}

	_, err = wt.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "chisel",
			Email: "chisel@local",
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("committing: %w", err)
	}

	return nil
}

// Log returns commits that touch the given path, newest first.
func (gb *GitBackend) Log(path string) ([]Revision, error) {
	rel, err := filepath.Rel(gb.dir, path)
	if err != nil {
		rel = path
	}
	rel = filepath.ToSlash(rel)

	commitIter, err := gb.repo.Log(&git.LogOptions{
		PathFilter: func(p string) bool {
			return p == rel
		},
		Order: git.LogOrderCommitterTime,
	})
	if err != nil {
		return nil, fmt.Errorf("reading log: %w", err)
	}

	var revs []Revision
	err = commitIter.ForEach(func(c *object.Commit) error {
		revs = append(revs, Revision{
			Hash:      c.Hash.String(),
			Timestamp: c.Author.When,
			Message:   strings.TrimSpace(c.Message),
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("iterating commits: %w", err)
	}

	return revs, nil
}

// Diff returns a unified diff between two revisions of a path.
func (gb *GitBackend) Diff(path string, rev1, rev2 string) (string, error) {
	rel, err := filepath.Rel(gb.dir, path)
	if err != nil {
		rel = path
	}
	rel = filepath.ToSlash(rel)

	hash1 := plumbing.NewHash(rev1)
	hash2 := plumbing.NewHash(rev2)

	commit1, err := gb.repo.CommitObject(hash1)
	if err != nil {
		return "", fmt.Errorf("resolving %s: %w", shortHash(rev1), err)
	}
	commit2, err := gb.repo.CommitObject(hash2)
	if err != nil {
		return "", fmt.Errorf("resolving %s: %w", shortHash(rev2), err)
	}

	tree1, err := commit1.Tree()
	if err != nil {
		return "", fmt.Errorf("tree %s: %w", shortHash(rev1), err)
	}
	tree2, err := commit2.Tree()
	if err != nil {
		return "", fmt.Errorf("tree %s: %w", shortHash(rev2), err)
	}

	changes, err := tree1.Diff(tree2)
	if err != nil {
		return "", fmt.Errorf("diffing: %w", err)
	}

	var sb strings.Builder
	for _, change := range changes {
		patch, err := change.Patch()
		if err != nil {
			continue
		}
		sb.WriteString(patch.String())
		sb.WriteRune('\n')
	}

	if sb.Len() == 0 {
		return "(no changes)", nil
	}

	return sb.String(), nil
}

// Restore returns the content of a path at the given revision.
func (gb *GitBackend) Restore(path string, rev string) (string, error) {
	rel, err := filepath.Rel(gb.dir, path)
	if err != nil {
		rel = path
	}
	rel = filepath.ToSlash(rel)

	hash := plumbing.NewHash(rev)
	commit, err := gb.repo.CommitObject(hash)
	if err != nil {
		return "", fmt.Errorf("resolving %s: %w", shortHash(rev), err)
	}

	tree, err := commit.Tree()
	if err != nil {
		return "", fmt.Errorf("tree: %w", err)
	}

	file, err := tree.File(rel)
	if err != nil {
		// The file might not exist in this commit — try parent.
		return "", fmt.Errorf("finding %s in %s: %w", rel, shortHash(rev), err)
	}

	content, err := file.Contents()
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", rel, err)
	}

	return content, nil
}
