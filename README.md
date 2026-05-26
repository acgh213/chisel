# ✧ chisel v0.1 ✧

A local-first markdown writing TUI. Binder tree on the left, editor on the right. Ctrl+S saves. That's it.

**Brutal MVP.** Filesystem IS the data model. No manifest, no config, no LLM, no git.

## quick start

```bash
go build -o chisel .
./chisel my-project/
```

Opens any directory as a writing project. Folders and `.md` files become the binder tree. Navigate with j/k, open with Enter, edit, save.

## keybindings

| key | action |
|-----|--------|
| j / k / ↑ / ↓ | Navigate binder |
| Enter | Open file / toggle folder |
| Space | Toggle folder |
| h / Left | Collapse folder |
| l / Right | Expand folder |
| Tab | Switch binder ↔ editor |
| Ctrl+N | New scene |
| Ctrl+S | Save |
| Ctrl+Q / Esc | Quit |

## what's not here (yet)

- No LLM integration
- No revision history (git/jj)
- No manifest or config files
- No export
- No themes (peach only)
- No corkboard, outline, character sheets, pomodoro

Just open a directory, write, save.

## architecture

```
Go binary
  ├── binder (filesystem tree — folders + .md files)
  └── editor (markdown textarea)
```

Single binary. No Python backend. No runtime dependencies beyond the terminal.

## building

```bash
go build -o chisel .
```

Requires Go 1.22+.

## philosophy

Plain markdown files in folders. If chisel disappears, your writing is still there. No lock-in, no proprietary formats.

## license

MIT — see [LICENSE](LICENSE).
