# Gemini A2A example agent

Minimal [A2A](https://a2a-protocol.org/latest/) server that forwards each user message to **Google Gemini** using [`google.golang.org/genai`](https://googleapis.github.io/go-genai/). Use it to exercise the **m2a** CLI against a real model.

## Prerequisites

- A Gemini API key from [Google AI Studio](https://aistudio.google.com/apikey) — see [Get a Gemini API key](https://ai.google.dev/gemini-api/docs/api-key).
- Go **1.24.4+** (same as the parent repo).

## Run the agent

From **this directory**:

```bash
export GEMINI_API_KEY="your-key-here"
# or: export GOOGLE_API_KEY="your-key-here"

go run . -port 8080
```

Or:

```bash
chmod +x run.sh
GEMINI_API_KEY="your-key-here" ./run.sh -port 8080
```

Defaults: `-port 8080`, `-model gemini-2.0-flash`. Override the model if your key supports a different one (see [Gemini models](https://ai.google.dev/gemini-api/docs/models/gemini)).

The process listens on `127.0.0.1` and serves:

- **Agent card:** `http://127.0.0.1:8080/.well-known/agent-card.json`
- **JSON-RPC:** `http://127.0.0.1:8080/invoke`
- **HTTP+JSON (REST):** `http://127.0.0.1:8080/`

## Talk to it with m2a

From the **repository root** (`m2a/`), in another terminal:

```bash
export GEMINI_API_KEY="same-key"   # not required for m2a; only the agent needs it

go run ./cmd/m2a/ -base http://127.0.0.1:8080
```

Or after `go install` / `go build` of `m2a`:

```bash
m2a -base http://127.0.0.1:8080
```

Try `-transport http` or `-transport jsonrpc` if you want to force one binding.

## Notes

- This example is for **local testing** only: no auth on the HTTP server, API key in the environment.
- The agent card lists **both** `HTTP+JSON` and `JSONRPC` so you can test either path.
