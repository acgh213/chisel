# ✧ chisel ✧

a local-first, markdown-native writing tool with LLM augmentation. for essays, blog posts, and fiction — the kind of writing where you need to move scenes around at 2am without the tool getting in your way.

## what

chisel is a TUI writing environment that does what scrivener does for structure (binder, scene organization, corkboard vibes) but in a terminal, with plain markdown files, and with a local LLM that can help you think without writing for you.

it runs entirely locally. your writing lives in `.md` files on your filesystem. the LLM features are optional, configurable, and private — no cloud, no subscription, no uploading your half-finished novel to a server somewhere.

## core ideas

- **plain markdown files in folders.** if chisel disappears tomorrow, your writing is still there. no proprietary formats, no lock-in.
- **scene-level organization.** a binder tree that tracks scenes, chapters, drafts, word counts, status — the structural stuff scrivener gets right.
- **LLM as a tool, not a co-author.** rewrite suggestions, research gathering, stylistic analysis, summarization. the model helps you think. it doesn't replace your voice.
- **local-first, provider-flexible.** runs with LM Studio, llama.cpp, or any OpenAI/Anthropic-compatible API. you choose where the model lives.
- **pane configurations as modes.** two panes for focused writing (file tree + editor). three panes when you want the LLM in the room. toggle with a keystroke, not window management.
- **a research mood board.** a `/research` folder where the LLM can dump notes, topic explorations, and references tagged to specific scenes. searchable, scannable, out of your way when you're drafting.

## architecture (planned)

```
chisel/
├── chisel.py          # backend — LLM calls, research, analysis (Python)
├── tui/               # bubbletea TUI (Go)
│   ├── main.go
│   ├── model.go       # root model + state
│   ├── editor.go      # writing pane
│   ├── binder.go      # scene tree / navigation
│   ├── manifest.go    # scene metadata (JSONL, same pattern as screenshot cataloger)
│   ├── llm.go         # model interaction, streaming responses
│   ├── research.go    # research gathering and tagging
│   └── styles.go      # all colors and styles
├── scenes/            # your writing lives here — one .md file per scene
├── research/          # LLM-gathered research notes
└── manifest.jsonl     # scene metadata: title, status, word count, tags, draft order
```

## LLM features

- **rewrite** — suggest alternatives for a selected passage
- **generate** — continue from cursor with a prompt
- **summarize** — summarize a scene, chapter, or selection
- **ask** — ask questions about the current scene or research a topic
- **analyze** — surface stylistic patterns, overused tics, rhythm issues (using a custom fine-tuned model if available)
- **research** — gather and tag research notes to specific scenes

## configuration

```json
{
  "api_base": "http://localhost:1234/v1",
  "model": "gemma-4-e4b",
  "max_tokens": 2048,
  "temperature": 0.7,
  "editor": "internal",
  "theme": "peach"
}
```

## inspiration

- [scrivener](https://www.literatureandlatte.com/scrivener/overview) — the gold standard for scene-level writing organization
- [bubbletea](https://github.com/charmbracelet/bubbletea) — the TUI framework that makes terminal apps beautiful
- the screenshot cataloger — a sibling project that proved the architecture works (Go TUI → Python LLM backend → local models)

## status

**planning / pre-alpha.** this README is the design document. implementation starts soon.

---

*built for people who want their writing tools to feel like tools, not like software.*
