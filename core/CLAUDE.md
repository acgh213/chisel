# core/ — pure data layer

Zero charmbracelet imports. All types are plain Go structs.
`go list -deps ./core` must return only stdlib + `yaml.v3` + `go-git`.
A future GUI (Wails, Fyne, etc.) reuses this package without touching `tui/`.

## invariants

- `core` never prints, logs, or touches the terminal.
- All errors propagate to the caller; the TUI decides how to display them.
- Frontmatter parse failures degrade gracefully — a bad header never fails a load.
- Best-effort readers (`ListCharacters`, `ListLocations`, `ReadSceneInfo`) skip unreadable files rather than erroring.

---

## file reference

### project.go

**Types:** `FileNode`, `Project`

**Key exports:**
- `NewProject(root string) *Project` — create a project handle; does not scan yet
- `Project.BuildTree() error` — walk the filesystem, populate `Project.Nodes` (excludes `.git/`, `exports/`, `characters/`, `locations/`, `notes/`)
- `Flatten(nodes []FileNode, out *[]FileNode)` — depth-first flatten of the tree for the binder list

`FileNode` carries `Name`, `Path`, `IsDir`, `Expanded`, `Children`, `Depth`, and `Status` (read from frontmatter during tree build).

---

### scene.go

**Types:** `Scene`

**Key exports:**
- `LoadScene(path string) (*Scene, error)` — read file, parse frontmatter + body
- `ParseScene(path, content string) *Scene` — in-memory parse (used in tests)
- `Scene.Save() error` — serialize frontmatter + body back to disk, auto-update `word_count`/`modified`
- `CreateScene(path string) error` — create a new `.md` file with minimal frontmatter stubs
- `WordCount(content string) int` — count prose words (whitespace-split, ignores frontmatter)

`Scene` has `Path string`, `Meta Metadata`, `Body string`.

---

### metadata.go

**Types:** `Status` (string constant type), `Metadata`

**Status constants:** `StatusDraft`, `StatusRevision`, `StatusFinal`, `StatusIdea`

`Metadata` YAML fields (snake_case in file):
`title`, `status`, `synopsis`, `tags`, `draft_order`, `word_target`, `pov`, `word_count`, `created`, `modified`, `timeline_date`, `notes`

**Key exports:**
- `splitFrontmatter(content string) (front, body string)` — private; splits `---` block
- `serializeFrontmatter(m Metadata) string` — private; round-trips yaml without losing unknown keys
- `ReadSceneNotes(path string) string` — read `notes` field from a file's frontmatter without a full scene load

---

### revision.go

**Types:** `Revision`, `RevisionBackend` (interface), `GitBackend`

`RevisionBackend` interface:
```go
Snapshot(msg string) error
Log() ([]Revision, error)
Diff(hash string) (string, error)
Restore(hash, destPath string) error
```

**Key exports:**
- `OpenGitBackend(projectDir string) (*GitBackend, error)` — open or lazily create `.git` inside the project dir
- `ShortHash(hash string) string` — first 7 chars of a commit hash

`GitBackend` treats `go-git`'s `ErrEmptyCommit` as a no-op (no change since last snapshot).
The `.git` directory is created on first `Snapshot`, not at startup.

---

### outline.go

**Types:** `SceneInfo`

`SceneInfo` carries `Path`, `Name`, `Title`, `Synopsis`, `Status`, `WordCount`, `WordTarget`, `DraftOrder`.

**Key exports:**
- `ReadSceneInfo(path string) (*SceneInfo, error)` — cheap metadata read without full scene parse
- `FolderScenes(dir string) ([]SceneInfo, error)` — all `.md` files in one directory (non-recursive)
- `SortScenesForReading(infos []SceneInfo)` — **single authority** for scene order: explicit `draft_order` ascending first, then alphabetical by filename

---

### export.go

**Types:** `ExportResult`

**Key exports:**
- `Project.Export(pandocPath string) (ExportResult, error)` — compile all scenes into `exports/manuscript.md` in reading order; if `pandocPath != ""`, also produce `exports/manuscript.docx` via `os/exec`

`ExportResult` has `ManuscriptPath`, `DocxPath` (empty if pandoc skipped), `SceneCount`, `WordCount`.
The `exports/` directory is excluded from the walk that populates the manuscript.

---

### crud.go

**Key exports:**
- `CreateFolder(dir, name string) error` — `os.MkdirAll` under dir
- `RenameNode(path, newName string) error` — renames file/dir; auto-appends `.md` if target lacks it and source was `.md`
- `DeleteNode(path string) error` — `os.RemoveAll` (works for files and directories)

No confirmation logic — that lives in `tui/prompt.go`.

---

### scaffold.go

**Types:** `Template` (string constant type), `ScaffoldOptions`

**Template constants:** `TemplateMinimal`, `TemplateNovel`, `TemplateShortStories`

**Key exports:**
- `ParseTemplate(s string) (Template, error)` — validate template name from CLI flag
- `Slugify(name string) string` — convert project name to safe directory name (lowercase, hyphens)
- `ScaffoldProject(dir string, opts ScaffoldOptions) error` — write README + sample scene structure for the chosen template

`ScaffoldOptions`: `Name string`, `Template Template`, `Force bool` (overwrite if exists).

---

### character.go

**Types:** `CharacterMeta`, `Character`

`CharacterMeta` YAML fields: `name`, `role`, `description`, `arc`, `voice`, `relationships`, `tags`

**Key exports:**
- `LoadCharacter(path string) (*Character, error)` — parse a single character sheet
- `ListCharacters(root string) ([]*Character, error)` — walk `characters/` dir; returns `nil, nil` if dir missing
- `CharactersDir(root string) string` — canonical path: `<root>/characters/`

---

### location.go

**Types:** `LocationMeta`, `Location`

`LocationMeta` YAML fields: `name`, `type`, `description`, `atmosphere`, `significance`, `tags`

**Key exports:**
- `LoadLocation(path string) (*Location, error)` — parse a single location sheet
- `ListLocations(root string) ([]*Location, error)` — walk `locations/` dir; returns `nil, nil` if dir missing
- `LocationsDir(root string) string` — canonical path: `<root>/locations/`

---

### timeline.go

**Types:** `TimelineEntry`

`TimelineEntry` carries `Path`, `Title`, `TimelineDate` (string, RFC-3339 or freeform), `Status`, `WordCount`.

**Key exports:**
- `BuildTimeline(root string) ([]TimelineEntry, error)` — walk entire project, parse `timeline_date` from frontmatter; sort dated entries ascending, then undated entries alphabetically appended

---

### notes.go

**Key exports:**
- `AppendScratch(root, text string) error` — append a timestamped line to `notes/scratch.md` (creates file/dir if needed); append-only, no frontmatter, no overwrite risk

---

### search.go

**Types:** `SearchResult`

`SearchResult` carries `Path`, `Title`, `LineNum`, `Line` (the matching line text).

**Key exports:**
- `SearchScenes(root, query string) ([]SearchResult, error)` — case-insensitive substring search over scene bodies (not frontmatter); skips `exports/`, `.git/`, `characters/`, `locations/`

---

### config.go

**Types:** `ChiselConfig`

`ChiselConfig` YAML fields: `theme` (string), `daily_goal` (int, words/day)

**Key exports:**
- `LoadConfig(root string) (ChiselConfig, error)` — read `.chisel.yaml`; returns zero-value config (not error) if file missing
- `SaveConfig(root string, cfg ChiselConfig) error` — write `.chisel.yaml`

Config is intentionally minimal — it stores only user preferences, not project content. Gracefully ignored when absent.
