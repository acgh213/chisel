# changelog

all notable changes to chisel will be documented in this file.

format based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
chisel uses [Semantic Versioning](https://semver.org/).

---

## [unreleased]

### added
- project vision, design document, goals, and implementation plan
- repo scaffolded at [github.com/acgh213/chisel](https://github.com/acgh213/chisel)

---

## versioning convention

| phase | version | what ships |
|-------|---------|-----------|
| scaffolding | 0.0.1 | project creation, config, manifest I/O |
| binder + editor | 0.1.0 | writing experience, revision history (git) |
| llm integration | 0.2.0 | rewrite, generate, summarize, ask |
| mirror + research | 0.3.0 | style analysis, research gathering |
| export + polish | 1.0.0 | manuscript export, themes, corkboard, outline |
| character notes | 1.1.0 | character sheets, scene notes |
| jj backend | 1.2.0 | jj revision history, git→jj migration |

---

## [0.1.0] — 2026-05-24

### added
- **phase 0 (scaffolding):** `chisel new` command scaffolds a project with `scenes/`, `research/`, `exports/`, `.gitignore`, `config.json`, `manifest.jsonl`, and git init via go-git
- **phase 0 (scaffolding):** config loading (`LoadConfig`/`SaveConfig`/`DefaultConfig`) with llm, mirror, history, and editor slots
- **phase 0 (scaffolding):** manifest I/O (`LoadManifest`/`SaveManifest`/`AppendEntry`) with JSONL read/write and append-only semantics
- **phase 0 (scaffolding):** style colour tokens (peach theme default) as lipgloss.Color constants
- **phase 1 (binder + editor):** binder tree with recursive `scenes/` directory scan, expand/collapse folders, status indicators (draft · revised · done)
- **phase 1 (binder + editor):** markdown editor wrapping `bubbles/textarea` with word count, file load/save, modified tracking
- **phase 1 (binder + editor):** three pane modes: Ctrl+1 (editor only), Ctrl+2 (binder + editor), Ctrl+3 (binder + editor + LLM placeholder)
- **phase 1 (binder + editor):** scene CRUD: `n` inline prompt (bubbles/textinput) for new scene, `d` delete with confirm dialog, `F2` rename, `K`/`J` reorder
- **phase 1 (binder + editor):** status bar with scene name, word count, modified indicator, session word count, session timer, and focus indicator
- **phase 1 (binder + editor):** Tab to toggle focus between binder and editor; Esc to return focus to editor from binder
- **phase 2 (revision history):** `RevisionBackend` interface (Save, Log, Diff, Restore) abstracted for future jj backend
- **phase 2 (revision history):** go-git backend implementing `RevisionBackend` — auto-commit on every Ctrl+S with structured commit messages
- **phase 2 (revision history):** history browser (Ctrl+H) with revision list, side-by-side diff view (enter), and restore (r)
- cross-document consistency fixes: resolved Ctrl+S save vs. summarize conflict (summarize → Ctrl+Shift+S); aligned phase ordering across all four docs; removed shell-out git commands in favour of go-git; removed premature pillow dependency
- platform notes: Windows compatibility via `filepath` package, `os.Executable()` for locating `chisel.py`, reserved character stripping

## [0.0.0] — 2026-05-24

### added
- initial repo creation
- README with project vision
- DESIGN.md with architecture, data model, pane layouts, llm integration
- GOALS.md with short/medium/long-term roadmap and non-goals
- PLAN.md with phased implementation details
