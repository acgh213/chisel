#!/usr/bin/env python3
"""
chisel.py — LLM backend for chisel TUI.

Communicates via NDJSON over stdin/stdout. Reads requests from stdin,
calls the configured OpenAI-compatible API, and writes responses to stdout.

Protocol:
  Request:  {"op": "<operation>", "text": "...", "context": "..."}
  Response: {"op": "<operation>", "result": "...", "tokens": N, "status": "ok"}
  Streaming: multiple lines with "status": "streaming", then final "status": "ok"
"""

import json
import os
import sys
from pathlib import Path

# ---------------------------------------------------------------------------
# config loading
# ---------------------------------------------------------------------------

def load_config():
    """Load config.json from the project root (two dirs above this script, or
    from the CHISEL_PROJECT environment variable)."""
    project_dir = os.environ.get("CHISEL_PROJECT")
    if not project_dir:
        # Guess: chisel.py lives in the project root.
        project_dir = Path(__file__).resolve().parent

    config_path = Path(project_dir) / "config.json"
    if config_path.exists():
        with open(config_path) as f:
            return json.load(f)

    # Fallback defaults.
    return {
        "llm": {
            "api_base": "http://localhost:1234/v1",
            "model": "",
            "max_tokens": 2048,
            "temperature": 0.7,
        },
        "mirror": {
            "api_base": "http://localhost:1234/v1",
            "model": "",
            "max_tokens": 1024,
            "temperature": 0.3,
        },
    }


# ---------------------------------------------------------------------------
# response helpers
# ---------------------------------------------------------------------------

def send(op, result, tokens=0, status="ok"):
    """Write a single NDJSON response line to stdout."""
    line = json.dumps({
        "op": op,
        "result": result,
        "tokens": tokens,
        "status": status,
    })
    sys.stdout.write(line + "\n")
    sys.stdout.flush()


def send_streaming(op, chunk):
    """Write a streaming chunk."""
    line = json.dumps({
        "op": op,
        "result": chunk,
        "tokens": 0,
        "status": "streaming",
    })
    sys.stdout.write(line + "\n")
    sys.stdout.flush()


# ---------------------------------------------------------------------------
# llm client
# ---------------------------------------------------------------------------

def make_client(config, slot="llm"):
    """Create an OpenAI client from the given config slot."""
    try:
        from openai import OpenAI
    except ImportError:
        send("error", "openai package not installed — run: pip install openai")
        return None

    slot_cfg = config.get(slot, {})
    api_base = slot_cfg.get("api_base", "http://localhost:1234/v1")

    return OpenAI(
        base_url=api_base,
        api_key="not-needed",  # local models don't require a key
    )


# ---------------------------------------------------------------------------
# operations
# ---------------------------------------------------------------------------

def handle_rewrite(client, config, req):
    """Suggest alternative phrasings for the given text."""
    text = req.get("text", "")
    context = req.get("context", "")

    if not text.strip():
        send("rewrite", "(no text selected)")
        return

    system = (
        "You are a writing assistant. Suggest alternative phrasings for the "
        "given text. Offer exactly three alternatives, each on its own line "
        "prefixed with '• '. Be concise — one sentence per alternative. "
        "Match the original tone and style. Do not explain your choices."
    )

    user = f"Context:\n{context}\n\nText to rewrite:\n{text}"

    try:
        response = _chat(client, config, system, user)
        send("rewrite", response["content"], response["tokens"])
    except Exception as e:
        send("rewrite", f"error: {e}", status="error")


def handle_generate(client, config, req):
    """Continue writing from the cursor position."""
    text = req.get("text", "")
    guidance = req.get("guidance", "")

    if not text.strip():
        send("generate", "(no context to continue from)")
        return

    guidance_line = ""
    if guidance.strip():
        guidance_line = f"\nWriter's guidance: {guidance}"

    system = (
        "You are a writing assistant. Continue the given text in the same "
        "voice, tone, and style. Do not summarise or recap — just continue "
        "the prose naturally. Write 2-4 sentences."
    )

    user = f"Continue this:{guidance_line}\n\n{text}"

    try:
        response = _chat(client, config, system, user)
        send("generate", response["content"], response["tokens"])
    except Exception as e:
        send("generate", f"error: {e}", status="error")


def handle_summarize(client, config, req):
    """Summarise the given text."""
    text = req.get("text", "")

    if not text.strip():
        send("summarize", "(no text to summarise)")
        return

    system = (
        "Summarise the following text concisely. Preserve key details, "
        "character names, and plot points. Use 2-4 sentences."
    )

    try:
        response = _chat(client, config, system, text)
        send("summarize", response["content"], response["tokens"])
    except Exception as e:
        send("summarize", f"error: {e}", status="error")


def handle_ask(client, config, req):
    """Answer a free-form question about the text."""
    question = req.get("question", "")
    context = req.get("text", "")

    if not question.strip():
        send("ask", "(no question)")
        return

    system = (
        "You are a knowledgeable writing assistant. Answer the user's "
        "question about their text. Be specific and reference details "
        "from the provided context. Keep answers under 200 words."
    )

    user = f"Context:\n{context}\n\nQuestion: {question}"

    try:
        client_obj = make_client(config)
        if client_obj is None:
            return

        slot_cfg = config.get("llm", {})
        model = slot_cfg.get("model", "")
        max_tokens = slot_cfg.get("max_tokens", 1024)
        temperature = slot_cfg.get("temperature", 0.7)

        kwargs = {
            "model": model,
            "messages": [
                {"role": "system", "content": system},
                {"role": "user", "content": user},
            ],
            "max_tokens": max_tokens,
            "temperature": temperature,
            "stream": True,
        }

        stream = client_obj.chat.completions.create(**kwargs)
        full = []
        tokens = 0
        for chunk in stream:
            delta = chunk.choices[0].delta
            if delta.content:
                full.append(delta.content)
                send_streaming("ask", delta.content)
                tokens += 1

        send("ask", "".join(full), tokens, status="ok")
    except Exception as e:
        send("ask", f"error: {e}", status="error")


def handle_analyze(client, config, req):
    """Stylistic analysis using the mirror model."""
    text = req.get("text", "")

    if not text.strip():
        send("analyze", "(no text to analyse)")
        return

    system = (
        "You are a stylistic editor. Analyse the following text for: "
        "1) overused words or phrases (list top 3), "
        "2) sentence rhythm issues (monotonous length, awkward cadence), "
        "3) writerly tics (filter words, hedging, passive voice overuse). "
        "Be concise. Use bullet points."
    )

    try:
        response = _chat(client, config, system, text, slot="mirror")
        send("analyze", response["content"], response["tokens"])
    except Exception as e:
        send("analyze", f"error: {e}", status="error")


# ---------------------------------------------------------------------------
# internal helpers
# ---------------------------------------------------------------------------

def _chat(client, config, system, user, slot="llm"):
    """Send a chat completion and return {"content": str, "tokens": int}."""
    slot_cfg = config.get(slot, {})
    model = slot_cfg.get("model", "")
    max_tokens = slot_cfg.get("max_tokens", 1024)
    temperature = slot_cfg.get("temperature", 0.7)

    kwargs = {
        "model": model,
        "messages": [
            {"role": "system", "content": system},
            {"role": "user", "content": user},
        ],
        "max_tokens": max_tokens,
        "temperature": temperature,
    }

    resp = client.chat.completions.create(**kwargs)
    content = resp.choices[0].message.content or ""
    tokens = resp.usage.total_tokens if resp.usage else 0
    return {"content": content, "tokens": tokens}


# ---------------------------------------------------------------------------
# main loop
# ---------------------------------------------------------------------------

def main():
    config = load_config()
    client = make_client(config)
    if client is None:
        return

    # Signal readiness.
    send("ready", "chisel.py ready")

    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue

        try:
            req = json.loads(line)
        except json.JSONDecodeError:
            send("error", f"invalid json: {line[:80]}")
            continue

        op = req.get("op", "")

        if op == "rewrite":
            handle_rewrite(client, config, req)
        elif op == "generate":
            handle_generate(client, config, req)
        elif op == "summarize":
            handle_summarize(client, config, req)
        elif op == "ask":
            handle_ask(client, config, req)
        elif op == "analyze":
            handle_analyze(client, config, req)
        elif op == "ping":
            send("pong", "alive")
        else:
            send("error", f"unknown operation: {op}")


if __name__ == "__main__":
    main()
