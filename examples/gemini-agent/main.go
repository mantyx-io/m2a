// Command gemini-agent is a minimal A2A server that forwards chat to Google Gemini for local testing.
package main

import (
	"context"
	"flag"
	"fmt"
	"iter"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2asrv"
	"google.golang.org/genai"
)

func main() {
	port := flag.Int("port", 8080, "HTTP listen port")
	model := flag.String("model", "gemini-3-flash-preview", "Gemini model id (see https://ai.google.dev/gemini-api/docs/models)")
	flag.Parse()

	if strings.TrimSpace(os.Getenv("GEMINI_API_KEY")) == "" && strings.TrimSpace(os.Getenv("GOOGLE_API_KEY")) == "" {
		log.Fatal("set GEMINI_API_KEY or GOOGLE_API_KEY to your Gemini API key")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{Backend: genai.BackendGeminiAPI})
	if err != nil {
		log.Fatalf("genai client: %v", err)
	}

	base := fmt.Sprintf("http://127.0.0.1:%d", *port)
	jsonRPC := base + "/invoke"
	agentCard := &a2a.AgentCard{
		Name:        "Gemini example (local)",
		Description: "Forwards user text to Gemini via the Google Gen AI SDK",
		Version:     "0.1.0",
		SupportedInterfaces: []*a2a.AgentInterface{
			a2a.NewAgentInterface(base, a2a.TransportProtocolHTTPJSON),
			a2a.NewAgentInterface(jsonRPC, a2a.TransportProtocolJSONRPC),
		},
		DefaultInputModes:  []string{"text/plain"},
		DefaultOutputModes: []string{"text/plain"},
		Capabilities:       a2a.AgentCapabilities{Streaming: false},
		Skills: []a2a.AgentSkill{
			{
				ID:          "gemini_chat",
				Name:        "Gemini chat",
				Description: "General Gemini completion for the user message",
				Tags:        []string{"gemini", "chat"},
			},
		},
	}

	exec := &geminiExecutor{client: client, model: *model}
	handler := a2asrv.NewHandler(exec)

	mux := http.NewServeMux()
	mux.Handle("/invoke", a2asrv.NewJSONRPCHandler(handler))
	mux.Handle("/", a2asrv.NewRESTHandler(handler))
	mux.Handle(a2asrv.WellKnownAgentCardPath, a2asrv.NewStaticAgentCardHandler(agentCard))

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	log.Printf("A2A Gemini agent listening on %s", base)
	log.Printf("  Agent card: %s%s", base, a2asrv.WellKnownAgentCardPath)
	log.Printf("  JSON-RPC:   %s", jsonRPC)
	log.Printf("  REST root:  %s", base)
	log.Printf("Try (from repo root): go run ./cmd/m2a/ -base %s", base)

	if err := http.Serve(ln, mux); err != nil {
		log.Fatal(err)
	}
}

type geminiExecutor struct {
	client *genai.Client
	model  string
}

func (g *geminiExecutor) Execute(ctx context.Context, execCtx *a2asrv.ExecutorContext) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {
		if execCtx.Message == nil {
			yield(a2a.NewMessage(a2a.MessageRoleAgent, a2a.NewTextPart("(no user message)")), nil)
			return
		}
		prompt := joinMessageText(execCtx.Message)
		if strings.TrimSpace(prompt) == "" {
			yield(a2a.NewMessage(a2a.MessageRoleAgent, a2a.NewTextPart("(empty message)")), nil)
			return
		}

		resp, err := g.client.Models.GenerateContent(ctx, g.model, genai.Text(prompt), nil)
		if err != nil {
			yield(a2a.NewMessage(a2a.MessageRoleAgent, a2a.NewTextPart("Gemini error: "+err.Error())), nil)
			return
		}
		out := strings.TrimSpace(resp.Text())
		if out == "" {
			out = "(no text in model response)"
		}
		yield(a2a.NewMessage(a2a.MessageRoleAgent, a2a.NewTextPart(out)), nil)
	}
}

func (g *geminiExecutor) Cancel(ctx context.Context, execCtx *a2asrv.ExecutorContext) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {
		yield(a2a.NewStatusUpdateEvent(
			execCtx,
			a2a.TaskStateCanceled,
			a2a.NewMessage(a2a.MessageRoleAgent, a2a.NewTextPart("Cancelled")),
		), nil)
	}
}

func joinMessageText(m *a2a.Message) string {
	var b strings.Builder
	for _, p := range m.Parts {
		if p == nil {
			continue
		}
		if t := strings.TrimSpace(p.Text()); t != "" {
			if b.Len() > 0 {
				b.WriteByte('\n')
			}
			b.WriteString(t)
		}
	}
	return b.String()
}
