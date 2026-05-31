package core

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// Revision is one saved snapshot of a scene.
type Revision struct {
	Hash      string
	Timestamp time.Time
	Message   string
}

// Short returns the abbreviated (7-char) revision hash.
func (r Revision) Short() string {
	if len(r.Hash) > 7 {
		return r.Hash[:7]
	}
	return r.Hash
}

// RevisionBackend abstracts the version-control layer so git can be swapped for
// jj later. It is deliberately thin and trigger-agnostic: Snapshot means
// "snapshot the current state now" — the decision of *when* to snapshot (an
// explicit save, an autosave, a structural edit) belongs to the caller, not the
// backend.
type RevisionBackend interface {
	// Snapshot commits the current on-disk state of path with a message. It is a
	// no-op (nil error) when nothing about path has changed since the last
	// snapshot.
	Snapshot(path, message string) error

	// Log returns the snapshots that touched path, newest first.
	Log(path string) ([]Revision, error)

	// Diff returns a unified diff of path between two revisions.
	Diff(path, rev1, rev2 string) (string, error)

	// Restore returns the content of path as it was at rev (it does not write to
	// disk — the caller decides what to do with the content).
	Restore(path, rev string) (string, error)
}

// GitBackend implements RevisionBackend using go-git (pure Go, no git binary)
// on a repository rooted at the project directory.
type GitBackend struct {
	repo *git.Repository
	dir  string
}

// compile-time check that GitBackend satisfies the interface.
var _ RevisionBackend = (*GitBackend)(nil)

// OpenGitBackend opens the git repository at projectDir, initializing one on
// first use if the directory is not yet a repo.
func OpenGitBackend(projectDir string) (*GitBackend, error) {
	repo, err := git.PlainOpen(projectDir)
	if errors.Is(err, git.ErrRepositoryNotExists) {
		repo, err = git.PlainInit(projectDir, false)
	}
	if err != nil {
		return nil, fmt.Errorf("opening git repo at %s: %w", projectDir, err)
	}
	return &GitBackend{repo: repo, dir: projectDir}, nil
}

// rel returns path relative to the repo root, slash-separated (git's form).
func (gb *GitBackend) rel(path string) string {
	r, err := filepath.Rel(gb.dir, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(r)
}

// Snapshot stages and commits a single file. A clean tree (no change since the
// last snapshot) is treated as a no-op rather than an error.
func (gb *GitBackend) Snapshot(path, message string) error {
	wt, err := gb.repo.Worktree()
	if err != nil {
		return fmt.Errorf("worktree: %w", err)
	}

	rel := gb.rel(path)
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
	if errors.Is(err, git.ErrEmptyCommit) {
		return nil // nothing changed — not an error
	}
	if err != nil {
		return fmt.Errorf("committing: %w", err)
	}
	return nil
}

// Log returns commits that touch path, newest first.
func (gb *GitBackend) Log(path string) ([]Revision, error) {
	rel := gb.rel(path)

	iter, err := gb.repo.Log(&git.LogOptions{
		PathFilter: func(p string) bool { return p == rel },
		Order:      git.LogOrderCommitterTime,
	})
	if err != nil {
		return nil, fmt.Errorf("reading log: %w", err)
	}
	defer iter.Close()

	var revs []Revision
	err = iter.ForEach(func(c *object.Commit) error {
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

// Diff returns a unified diff of path between rev1 (older) and rev2 (newer).
func (gb *GitBackend) Diff(path, rev1, rev2 string) (string, error) {
	tree1, err := gb.treeAt(rev1)
	if err != nil {
		return "", err
	}
	tree2, err := gb.treeAt(rev2)
	if err != nil {
		return "", err
	}

	changes, err := tree1.Diff(tree2)
	if err != nil {
		return "", fmt.Errorf("diffing: %w", err)
	}

	rel := gb.rel(path)
	var sb strings.Builder
	for _, ch := range changes {
		// Only the file we care about.
		if ch.From.Name != rel && ch.To.Name != rel {
			continue
		}
		patch, err := ch.Patch()
		if err != nil {
			continue
		}
		sb.WriteString(patch.String())
	}

	if sb.Len() == 0 {
		return "(no changes)", nil
	}
	return sb.String(), nil
}

// Restore returns the content of path at rev.
func (gb *GitBackend) Restore(path, rev string) (string, error) {
	commit, err := gb.repo.CommitObject(plumbing.NewHash(rev))
	if err != nil {
		return "", fmt.Errorf("resolving %s: %w", short(rev), err)
	}
	tree, err := commit.Tree()
	if err != nil {
		return "", fmt.Errorf("tree %s: %w", short(rev), err)
	}
	file, err := tree.File(gb.rel(path))
	if err != nil {
		return "", fmt.Errorf("finding file in %s: %w", short(rev), err)
	}
	content, err := file.Contents()
	if err != nil {
		return "", fmt.Errorf("reading file at %s: %w", short(rev), err)
	}
	return content, nil
}

// treeAt resolves a revision hash to its tree.
func (gb *GitBackend) treeAt(rev string) (*object.Tree, error) {
	commit, err := gb.repo.CommitObject(plumbing.NewHash(rev))
	if err != nil {
		return nil, fmt.Errorf("resolving %s: %w", short(rev), err)
	}
	tree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("tree %s: %w", short(rev), err)
	}
	return tree, nil
}

func short(hash string) string {
	if len(hash) > 7 {
		return hash[:7]
	}
	return hash
}
