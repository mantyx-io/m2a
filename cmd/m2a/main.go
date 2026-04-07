// Command m2a is a terminal client for the Agent2Agent protocol.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2aclient"
	"github.com/a2aproject/a2a-go/v2/a2aclient/agentcard"
	"github.com/a2aproject/a2a-go/v2/a2acompat/a2av0"
	"github.com/mantyx-io/m2a/internal/httpclient"
	"github.com/mantyx-io/m2a/internal/tui"
)

func main() {
	if len(os.Args) >= 2 && os.Args[1] == "version" {
		printVersion()
		return
	}
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "m2a: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	os.Args = append([]string{os.Args[0]}, reorderFlagsBeforePositionals(os.Args[1:])...)

	var (
		base      = flag.String("base", "", "Agent base URL (fetches /.well-known/agent-card.json unless -card is set)")
		cardURL   = flag.String("card", "", "Full URL to the agent card JSON (overrides -base well-known path)")
		cardPath  = flag.String("card-path", "", "Path relative to -base for the card (default: /.well-known/agent-card.json)")
		transport = flag.String("transport", "auto", "Transport: auto, http (HTTP+JSON), jsonrpc (gRPC not in this binary)")
		raw       = flag.Bool("raw", false, "Show agent replies as plain text (skip terminal markdown rendering)")
		debug     = flag.Bool("debug", false, "Log SendMessage request/response JSON in the chat transcript")
		showVer   = flag.Bool("version", false, "Print version and exit")
	)
	var headerFlags stringSliceFlag
	flag.Var(&headerFlags, "H", "HTTP header in the form Name: Value (repeatable; sent on card fetch and A2A requests)")
	flag.Parse()

	if *showVer {
		printVersion()
		return nil
	}

	args := flag.Args()
	if *base == "" && *cardURL == "" && len(args) == 1 {
		if isReservedPositional(args[0]) {
			return fmt.Errorf("%q is not a base URL — use the flag %s (see m2a -h)", args[0], reservedFlagHint(args[0]))
		}
		*base = args[0]
	}
	if *base == "" && *cardURL == "" {
		return fmt.Errorf("provide an agent base URL, e.g. m2a https://my-agent.example.com, or use -card for a direct card URL")
	}
	if *base != "" && *cardURL != "" {
		return fmt.Errorf("use either -base or -card, not both")
	}

	headerMap := parseHeaders(headerFlags)
	hc := httpclient.WithHeaders(headerMap)
	ctx := context.Background()

	card, err := loadAgentCard(ctx, hc, *base, *cardURL, *cardPath)
	if err != nil {
		return err
	}

	if err := ensureInferredInterfacesIfEmpty(card, *base, *cardURL, *transport); err != nil {
		return err
	}

	// Default transports speak A2A 1.0 wire format. Cards with protocolVersion 0.3.x need
	// [a2av0] JSON-RPC (legacy result shapes); using the 1.0 client against 0.3 yields
	// "unknown event type" when the server returns an unwrapped task/message.
	factoryOpts := []a2aclient.FactoryOption{
		a2aclient.WithJSONRPCTransport(hc),
		a2aclient.WithRESTTransport(hc),
		a2aclient.WithCompatTransport(a2av0.Version, a2a.TransportProtocolJSONRPC, a2av0.NewJSONRPCTransportFactory(a2av0.JSONRPCTransportConfig{Client: hc})),
		a2aclient.WithCompatTransport(a2av0.Version, a2a.TransportProtocolHTTPJSON, a2aclient.TransportFactoryFn(func(ctx context.Context, card *a2a.AgentCard, iface *a2a.AgentInterface) (a2aclient.Transport, error) {
			u, err := url.Parse(iface.URL)
			if err != nil {
				return nil, fmt.Errorf("failed to parse endpoint URL: %w", err)
			}
			return a2aclient.NewRESTTransport(u, hc), nil
		})),
	}
	if pref, err := transportPreference(*transport); err != nil {
		return err
	} else if len(pref) > 0 {
		factoryOpts = append(factoryOpts, a2aclient.WithConfig(a2aclient.Config{
			PreferredTransports: pref,
		}))
	}
	factory := a2aclient.NewFactory(factoryOpts...)
	client, err := factory.CreateFromCard(ctx, card)
	if err != nil {
		return fmt.Errorf("connect to agent: %w", err)
	}
	defer func() { _ = client.Destroy() }()

	agentName := strings.TrimSpace(card.Name)
	if agentName == "" {
		agentName = "agent"
	}

	return tui.Run(ctx, client, agentName, *raw, *debug)
}

func loadAgentCard(ctx context.Context, hc *http.Client, base, directCardURL, relPath string) (*a2a.AgentCard, error) {
	if directCardURL != "" {
		return fetchCardFromURL(ctx, hc, directCardURL)
	}
	resolver := &agentcard.Resolver{Client: hc}
	var opts []agentcard.ResolveOption
	if relPath != "" {
		p := relPath
		if !strings.HasPrefix(p, "/") {
			p = "/" + p
		}
		opts = append(opts, agentcard.WithPath(p))
	}
	return resolver.Resolve(ctx, strings.TrimRight(base, "/"), opts...)
}

func fetchCardFromURL(ctx context.Context, hc *http.Client, rawURL string) (*a2a.AgentCard, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch agent card: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch agent card: %s", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var card a2a.AgentCard
	if err := json.Unmarshal(body, &card); err != nil {
		return nil, fmt.Errorf("parse agent card: %w", err)
	}
	return &card, nil
}

// ensureInferredInterfacesIfEmpty fills in supportedInterfaces when the card omits them.
// For -transport auto, HTTP+JSON is listed before JSON-RPC so REST-only agents work.
func ensureInferredInterfacesIfEmpty(card *a2a.AgentCard, base, cardURLFlag, transport string) error {
	if len(card.SupportedInterfaces) > 0 {
		return nil
	}
	endpoint, err := inferAgentEndpoint(base, cardURLFlag)
	if err != nil {
		return fmt.Errorf("agent card has no supportedInterfaces: %w", err)
	}
	protos, err := inferredProtocols(transport)
	if err != nil {
		return err
	}
	var names []string
	for _, p := range protos {
		names = append(names, string(p))
	}
	fmt.Fprintf(os.Stderr, "m2a: card has no supportedInterfaces; trying %s at %s\n", strings.Join(names, ", then "), endpoint)
	var ifaces []*a2a.AgentInterface
	for _, p := range protos {
		ifaces = append(ifaces, a2a.NewAgentInterface(endpoint, p))
	}
	card.SupportedInterfaces = ifaces
	return nil
}

// transportPreference returns PreferredTransports for the SDK, or nil when order should follow the card.
func transportPreference(transport string) ([]a2a.TransportProtocol, error) {
	switch strings.ToLower(strings.TrimSpace(transport)) {
	case "", "auto":
		return nil, nil
	case "http", "rest", "http+json":
		return []a2a.TransportProtocol{a2a.TransportProtocolHTTPJSON}, nil
	case "jsonrpc":
		return []a2a.TransportProtocol{a2a.TransportProtocolJSONRPC}, nil
	case "grpc":
		return nil, fmt.Errorf("-transport grpc is not enabled in m2a (build from a2a-go with a2agrpc); use an agent card that lists gRPC or a different client")
	default:
		return nil, fmt.Errorf("unknown -transport %q (use auto, http, or jsonrpc)", transport)
	}
}

func inferredProtocols(transport string) ([]a2a.TransportProtocol, error) {
	switch strings.ToLower(strings.TrimSpace(transport)) {
	case "", "auto":
		return []a2a.TransportProtocol{
			a2a.TransportProtocolHTTPJSON,
			a2a.TransportProtocolJSONRPC,
		}, nil
	case "http", "rest", "http+json":
		return []a2a.TransportProtocol{a2a.TransportProtocolHTTPJSON}, nil
	case "jsonrpc":
		return []a2a.TransportProtocol{a2a.TransportProtocolJSONRPC}, nil
	case "grpc":
		return nil, fmt.Errorf("-transport grpc is not enabled in m2a; use jsonrpc, http, or auto")
	default:
		return nil, fmt.Errorf("unknown -transport %q (use auto, http, or jsonrpc)", transport)
	}
}

func inferAgentEndpoint(base, cardURLFlag string) (string, error) {
	if b := strings.TrimSpace(base); b != "" {
		return strings.TrimRight(b, "/"), nil
	}
	u, err := url.Parse(strings.TrimSpace(cardURLFlag))
	if err != nil {
		return "", fmt.Errorf("infer endpoint: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("cannot infer A2A URL from -card alone; use -base with the agent host, or a full https card URL")
	}
	origin := (&url.URL{Scheme: u.Scheme, Host: u.Host}).String()
	return strings.TrimRight(origin, "/"), nil
}

type stringSliceFlag []string

func (s *stringSliceFlag) String() string { return strings.Join(*s, ", ") }

func (s *stringSliceFlag) Set(v string) error {
	*s = append(*s, v)
	return nil
}

func parseHeaders(flags []string) map[string]string {
	out := make(map[string]string)
	for _, h := range flags {
		name, val, ok := strings.Cut(h, ":")
		if !ok {
			fmt.Fprintf(os.Stderr, "m2a: ignoring malformed -H %q (want Name: Value)\n", h)
			continue
		}
		out[strings.TrimSpace(name)] = strings.TrimSpace(val)
	}
	return out
}
