package main

import (
	"reflect"
	"testing"

	"github.com/a2aproject/a2a-go/v2/a2a"
)

func TestParseHeaders(t *testing.T) {
	got := parseHeaders([]string{
		"Authorization: Bearer token",
		"X-Custom:  a:b ",
	})
	want := map[string]string{
		"Authorization": "Bearer token",
		"X-Custom":      "a:b",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseHeaders() = %#v, want %#v", got, want)
	}
}

func TestInferAgentEndpoint(t *testing.T) {
	t.Parallel()
	got, err := inferAgentEndpoint("https://agent.example/foo", "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://agent.example/foo" {
		t.Fatalf("base: got %q", got)
	}
	got, err = inferAgentEndpoint("", "https://agent.example/path/card.json")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://agent.example" {
		t.Fatalf("card origin: got %q", got)
	}
}

func TestInferAgentEndpoint_errors(t *testing.T) {
	t.Parallel()
	_, err := inferAgentEndpoint("", "")
	if err == nil {
		t.Fatal("want error")
	}
	_, err = inferAgentEndpoint("", "/relative-only")
	if err == nil {
		t.Fatal("want error for relative card URL")
	}
}

func TestTransportPreference(t *testing.T) {
	t.Parallel()
	got, err := transportPreference("auto")
	if err != nil || got != nil {
		t.Fatalf("auto: %v %v", got, err)
	}
	got, err = transportPreference("jsonrpc")
	if err != nil || len(got) != 1 || got[0] != a2a.TransportProtocolJSONRPC {
		t.Fatalf("jsonrpc: %v %v", got, err)
	}
	got, err = transportPreference("http")
	if err != nil || len(got) != 1 || got[0] != a2a.TransportProtocolHTTPJSON {
		t.Fatalf("http: %v %v", got, err)
	}
	_, err = transportPreference("grpc")
	if err == nil {
		t.Fatal("grpc should error")
	}
}

func TestInferredProtocols(t *testing.T) {
	t.Parallel()
	p, err := inferredProtocols("auto")
	if err != nil || len(p) != 2 {
		t.Fatalf("auto: %v %v", p, err)
	}
	p, err = inferredProtocols("jsonrpc")
	if err != nil || len(p) != 1 || p[0] != a2a.TransportProtocolJSONRPC {
		t.Fatalf("jsonrpc: %v %v", p, err)
	}
}
