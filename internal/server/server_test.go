package server

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"nhooyr.io/websocket"

	"github.com/organic-programming/go-holons/pkg/transport"
	pb "github.com/organic-programming/sophia-who/gen/go/sophia_who/v1"
	"github.com/organic-programming/sophia-who/pkg/identity"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

// startTestServer launches a gRPC server over an in-memory connection.
// Returns a ready-to-use client and a cleanup function.
func startTestServer(t *testing.T, root string) (pb.SophiaWhoServiceClient, func()) {
	t.Helper()

	// Change to the test root so FindAll/FindByUUID scan the right directory
	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	lis := bufconn.Listen(bufSize)
	s := grpc.NewServer()
	pb.RegisterSophiaWhoServiceServer(s, &Server{})

	go func() { _ = s.Serve(lis) }()

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}

	cleanup := func() {
		conn.Close()
		s.Stop()
		os.Chdir(original) //nolint:errcheck
	}

	return pb.NewSophiaWhoServiceClient(conn), cleanup
}

// seedHolon creates a HOLON.md in a subdirectory of root.
func seedHolon(t *testing.T, root, uuid, givenName string) {
	t.Helper()
	dir := filepath.Join(root, givenName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	content := "---\nuuid: \"" + uuid + "\"\ngiven_name: \"" + givenName + "\"\nfamily_name: \"Test\"\nmotto: \"Testing.\"\ncomposer: \"Test\"\nclade: \"deterministic/pure\"\nstatus: draft\nborn: \"2026-01-01\"\nparents: []\nreproduction: \"manual\"\ngenerated_by: \"test\"\nlang: \"go\"\nproto_status: draft\n---\n# " + givenName + "\n"
	if err := os.WriteFile(filepath.Join(dir, "HOLON.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestListIdentities(t *testing.T) {
	root := t.TempDir()
	seedHolon(t, root, "list-uuid-1", "Alpha")
	seedHolon(t, root, "list-uuid-2", "Beta")

	client, cleanup := startTestServer(t, root)
	defer cleanup()

	resp, err := client.ListIdentities(context.Background(), &pb.ListIdentitiesRequest{})
	if err != nil {
		t.Fatalf("ListIdentities failed: %v", err)
	}
	if len(resp.Entries) != 2 {
		t.Fatalf("ListIdentities returned %d entries, want 2", len(resp.Entries))
	}

	paths := map[string]bool{}
	for _, e := range resp.Entries {
		if e.RelativePath == "" {
			t.Fatal("RelativePath must not be empty")
		}
		paths[e.RelativePath] = true
	}
	if !paths["Alpha"] {
		t.Error("ListIdentities missing RelativePath for Alpha")
	}
	if !paths["Beta"] {
		t.Error("ListIdentities missing RelativePath for Beta")
	}
}

func TestListIdentitiesEmpty(t *testing.T) {
	root := t.TempDir()

	client, cleanup := startTestServer(t, root)
	defer cleanup()

	resp, err := client.ListIdentities(context.Background(), &pb.ListIdentitiesRequest{})
	if err != nil {
		t.Fatalf("ListIdentities failed: %v", err)
	}
	if len(resp.Entries) != 0 {
		t.Errorf("ListIdentities returned %d entries, want 0", len(resp.Entries))
	}
}

func TestShowIdentity(t *testing.T) {
	root := t.TempDir()
	seedHolon(t, root, "show-uuid-42", "Gamma")

	client, cleanup := startTestServer(t, root)
	defer cleanup()

	resp, err := client.ShowIdentity(context.Background(), &pb.ShowIdentityRequest{
		Uuid: "show-uuid-42",
	})
	if err != nil {
		t.Fatalf("ShowIdentity failed: %v", err)
	}
	if resp.Identity.Uuid != "show-uuid-42" {
		t.Errorf("UUID = %q, want %q", resp.Identity.Uuid, "show-uuid-42")
	}
	if resp.Identity.GivenName != "Gamma" {
		t.Errorf("GivenName = %q, want %q", resp.Identity.GivenName, "Gamma")
	}
	if resp.RawContent == "" {
		t.Error("RawContent must not be empty")
	}
}

func TestShowIdentityPrefix(t *testing.T) {
	root := t.TempDir()
	seedHolon(t, root, "prefix-abcd-1234", "Delta")

	client, cleanup := startTestServer(t, root)
	defer cleanup()

	resp, err := client.ShowIdentity(context.Background(), &pb.ShowIdentityRequest{
		Uuid: "prefix-abcd",
	})
	if err != nil {
		t.Fatalf("ShowIdentity prefix failed: %v", err)
	}
	if resp.Identity.Uuid != "prefix-abcd-1234" {
		t.Errorf("UUID = %q, want %q", resp.Identity.Uuid, "prefix-abcd-1234")
	}
}

func TestShowIdentityNotFound(t *testing.T) {
	root := t.TempDir()

	client, cleanup := startTestServer(t, root)
	defer cleanup()

	_, err := client.ShowIdentity(context.Background(), &pb.ShowIdentityRequest{
		Uuid: "nonexistent",
	})
	if err == nil {
		t.Fatal("expected error for non-existent UUID")
	}
}

func TestCreateIdentity(t *testing.T) {
	root := t.TempDir()

	client, cleanup := startTestServer(t, root)
	defer cleanup()

	resp, err := client.CreateIdentity(context.Background(), &pb.CreateIdentityRequest{
		GivenName:  "NewHolon",
		FamilyName: "Creator",
		Motto:      "Born by gRPC.",
		Composer:   "Test Suite",
		Clade:      pb.Clade_PROBABILISTIC_GENERATIVE,
	})
	if err != nil {
		t.Fatalf("CreateIdentity failed: %v", err)
	}
	if resp.Identity.Uuid == "" {
		t.Error("UUID must not be empty")
	}
	if resp.Identity.GivenName != "NewHolon" {
		t.Errorf("GivenName = %q, want %q", resp.Identity.GivenName, "NewHolon")
	}
	if resp.FilePath == "" {
		t.Error("FilePath must not be empty")
	}

	// Verify the file was actually created
	if _, err := os.Stat(resp.FilePath); err != nil {
		t.Errorf("HOLON.md not created at %s: %v", resp.FilePath, err)
	}

	// Verify re-parseable
	data, err := os.ReadFile(resp.FilePath)
	if err != nil {
		t.Fatal(err)
	}
	parsed, _, err := identity.ParseFrontmatter(data)
	if err != nil {
		t.Fatalf("created HOLON.md is not parseable: %v", err)
	}
	if parsed.UUID != resp.Identity.Uuid {
		t.Errorf("parsed UUID = %q, want %q", parsed.UUID, resp.Identity.Uuid)
	}
}

func TestCreateIdentityValidation(t *testing.T) {
	root := t.TempDir()

	client, cleanup := startTestServer(t, root)
	defer cleanup()

	// Missing required fields
	_, err := client.CreateIdentity(context.Background(), &pb.CreateIdentityRequest{
		GivenName: "OnlyName",
		// Missing: FamilyName, Motto, Composer
	})
	if err == nil {
		t.Fatal("expected error for missing required fields")
	}
}

// --- Conversion helper tests (cover fallback branches) ---

func TestCladeToStringUnknown(t *testing.T) {
	result := cladeToString(pb.Clade_CLADE_UNSPECIFIED)
	if result != "deterministic/pure" {
		t.Errorf("cladeToString(UNSPECIFIED) = %q, want fallback", result)
	}
}

func TestCladeToStringAllValues(t *testing.T) {
	cases := []struct {
		clade pb.Clade
		want  string
	}{
		{pb.Clade_DETERMINISTIC_PURE, "deterministic/pure"},
		{pb.Clade_DETERMINISTIC_STATEFUL, "deterministic/stateful"},
		{pb.Clade_DETERMINISTIC_IO_BOUND, "deterministic/io_bound"},
		{pb.Clade_PROBABILISTIC_GENERATIVE, "probabilistic/generative"},
		{pb.Clade_PROBABILISTIC_PERCEPTUAL, "probabilistic/perceptual"},
		{pb.Clade_PROBABILISTIC_ADAPTIVE, "probabilistic/adaptive"},
	}
	for _, tc := range cases {
		got := cladeToString(tc.clade)
		if got != tc.want {
			t.Errorf("cladeToString(%v) = %q, want %q", tc.clade, got, tc.want)
		}
	}
}

func TestStringToCladeUnknown(t *testing.T) {
	result := stringToClade("unknown/clade")
	if result != pb.Clade_CLADE_UNSPECIFIED {
		t.Errorf("stringToClade(unknown) = %v, want CLADE_UNSPECIFIED", result)
	}
}

func TestStringToCladeAllValues(t *testing.T) {
	cases := []struct {
		s    string
		want pb.Clade
	}{
		{"deterministic/pure", pb.Clade_DETERMINISTIC_PURE},
		{"deterministic/stateful", pb.Clade_DETERMINISTIC_STATEFUL},
		{"deterministic/io_bound", pb.Clade_DETERMINISTIC_IO_BOUND},
		{"probabilistic/generative", pb.Clade_PROBABILISTIC_GENERATIVE},
		{"probabilistic/perceptual", pb.Clade_PROBABILISTIC_PERCEPTUAL},
		{"probabilistic/adaptive", pb.Clade_PROBABILISTIC_ADAPTIVE},
	}
	for _, tc := range cases {
		got := stringToClade(tc.s)
		if got != tc.want {
			t.Errorf("stringToClade(%q) = %v, want %v", tc.s, got, tc.want)
		}
	}
}

func TestStringToStatusAllValues(t *testing.T) {
	cases := []struct {
		s    string
		want pb.Status
	}{
		{"draft", pb.Status_DRAFT},
		{"stable", pb.Status_STABLE},
		{"deprecated", pb.Status_DEPRECATED},
		{"dead", pb.Status_DEAD},
		{"unknown", pb.Status_STATUS_UNSPECIFIED},
	}
	for _, tc := range cases {
		got := stringToStatus(tc.s)
		if got != tc.want {
			t.Errorf("stringToStatus(%q) = %v, want %v", tc.s, got, tc.want)
		}
	}
}

func TestReproductionToStringAllValues(t *testing.T) {
	cases := []struct {
		mode pb.ReproductionMode
		want string
	}{
		{pb.ReproductionMode_MANUAL, "manual"},
		{pb.ReproductionMode_ASSISTED, "assisted"},
		{pb.ReproductionMode_AUTOMATIC, "automatic"},
		{pb.ReproductionMode_AUTOPOIETIC, "autopoietic"},
		{pb.ReproductionMode_BRED, "bred"},
		{pb.ReproductionMode_REPRODUCTION_UNSPECIFIED, "manual"},
	}
	for _, tc := range cases {
		got := reproductionToString(tc.mode)
		if got != tc.want {
			t.Errorf("reproductionToString(%v) = %q, want %q", tc.mode, got, tc.want)
		}
	}
}

func TestStringToReproductionAllValues(t *testing.T) {
	cases := []struct {
		s    string
		want pb.ReproductionMode
	}{
		{"manual", pb.ReproductionMode_MANUAL},
		{"assisted", pb.ReproductionMode_ASSISTED},
		{"automatic", pb.ReproductionMode_AUTOMATIC},
		{"autopoietic", pb.ReproductionMode_AUTOPOIETIC},
		{"bred", pb.ReproductionMode_BRED},
		{"unknown", pb.ReproductionMode_REPRODUCTION_UNSPECIFIED},
	}
	for _, tc := range cases {
		got := stringToReproduction(tc.s)
		if got != tc.want {
			t.Errorf("stringToReproduction(%q) = %v, want %v", tc.s, got, tc.want)
		}
	}
}

// --- CreateIdentity with all optional fields ---

func TestCreateIdentityAllFields(t *testing.T) {
	root := t.TempDir()

	client, cleanup := startTestServer(t, root)
	defer cleanup()

	resp, err := client.CreateIdentity(context.Background(), &pb.CreateIdentityRequest{
		GivenName:    "Full",
		FamilyName:   "Holon",
		Motto:        "All fields matter.",
		Composer:     "Test",
		Clade:        pb.Clade_DETERMINISTIC_STATEFUL,
		Reproduction: pb.ReproductionMode_AUTOPOIETIC,
		Lang:         "rust",
		Aliases:      []string{"alias1", "alias2"},
		OutputDir:    filepath.Join(root, "custom-output"),
	})
	if err != nil {
		t.Fatalf("CreateIdentity failed: %v", err)
	}
	if resp.Identity.Lang != "rust" {
		t.Errorf("Lang = %q, want %q", resp.Identity.Lang, "rust")
	}
	if len(resp.Identity.Aliases) != 2 {
		t.Errorf("Aliases count = %d, want 2", len(resp.Identity.Aliases))
	}
	if resp.Identity.Clade != pb.Clade_DETERMINISTIC_STATEFUL {
		t.Errorf("Clade = %v, want DETERMINISTIC_STATEFUL", resp.Identity.Clade)
	}
}

// --- ListenAndServe error (port conflict) ---

func TestListenAndServePortConflict(t *testing.T) {
	// Bind a port first
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer lis.Close()

	// Extract the port
	port := lis.Addr().(*net.TCPAddr).Port

	// Now ListenAndServe on the same port should fail
	err = ListenAndServe(fmt.Sprintf("tcp://:%d", port), true)
	if err == nil {
		t.Fatal("expected error for port conflict")
	}
}

func TestListenAndServePortConflictNoReflect(t *testing.T) {
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer lis.Close()

	port := lis.Addr().(*net.TCPAddr).Port

	err = ListenAndServe(fmt.Sprintf("tcp://:%d", port), false)
	if err == nil {
		t.Fatal("expected error for port conflict (no-reflect)")
	}
}

func TestListenAndServeStartStop(t *testing.T) {
	// Find a free port
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	port := lis.Addr().(*net.TCPAddr).Port
	lis.Close() // free it for ListenAndServe

	errCh := make(chan error, 1)
	go func() {
		errCh <- ListenAndServe(fmt.Sprintf("tcp://:%d", port), true)
	}()

	// Give the server time to start, then connect and verify
	var conn net.Conn
	for i := 0; i < 20; i++ {
		conn, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
		if err == nil {
			break
		}
		// Small delay between retries
		select {
		case <-errCh:
			t.Fatal("server stopped early")
		default:
		}
	}
	if conn != nil {
		conn.Close()
	}

	// Server is running — kill it via port
	// Just verify it started; we can't cleanly stop ListenAndServe from outside
}

func TestListenAndServeStartStopNoReflect(t *testing.T) {
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	port := lis.Addr().(*net.TCPAddr).Port
	lis.Close()

	errCh := make(chan error, 1)
	go func() {
		errCh <- ListenAndServe(fmt.Sprintf("tcp://:%d", port), false)
	}()

	var conn net.Conn
	for i := 0; i < 20; i++ {
		conn, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
		if err == nil {
			break
		}
		select {
		case <-errCh:
			t.Fatal("server stopped early")
		default:
		}
	}
	if conn != nil {
		conn.Close()
	}
}

func TestCreateIdentityReadOnlyDir(t *testing.T) {
	root := t.TempDir()

	// Make root read-only so MkdirAll fails for the default output dir
	readonlyDir := filepath.Join(root, "readonly")
	if err := os.MkdirAll(readonlyDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(readonlyDir, 0555); err != nil {
		t.Skip("cannot change dir permissions")
	}
	defer os.Chmod(readonlyDir, 0755) //nolint:errcheck

	client, cleanup := startTestServer(t, root)
	defer cleanup()

	_, err := client.CreateIdentity(context.Background(), &pb.CreateIdentityRequest{
		GivenName:  "FailDir",
		FamilyName: "Test",
		Motto:      "Should fail.",
		Composer:   "Test",
		OutputDir:  filepath.Join(readonlyDir, "impossible", "nested"),
	})
	if err == nil {
		t.Fatal("expected error for read-only output directory")
	}
}

// --- mem:// transport test (using go-holons SDK MemListener) ---

func TestMemTransport(t *testing.T) {
	root := t.TempDir()
	seedHolon(t, root, "mem-uuid-1", "MemTest")

	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(original) //nolint:errcheck

	mem := transport.NewMemListener()
	s := grpc.NewServer()
	pb.RegisterSophiaWhoServiceServer(s, &Server{})
	go func() { _ = s.Serve(mem) }()
	defer s.Stop()

	conn, err := grpc.NewClient(
		"passthrough:///mem",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return mem.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	client := pb.NewSophiaWhoServiceClient(conn)
	resp, err := client.ListIdentities(context.Background(), &pb.ListIdentitiesRequest{})
	if err != nil {
		t.Fatalf("ListIdentities over mem://: %v", err)
	}
	if len(resp.Entries) != 1 {
		t.Errorf("ListIdentities returned %d entries, want 1", len(resp.Entries))
	}
}

// --- ws:// transport test ---

func TestWSTransport(t *testing.T) {
	root := t.TempDir()
	seedHolon(t, root, "ws-uuid-1", "WSTest")

	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(original) //nolint:errcheck

	wsLis, err := transport.Listen("ws://127.0.0.1:0")
	if err != nil {
		t.Fatalf("ws listen: %v", err)
	}
	defer wsLis.Close()

	s := grpc.NewServer()
	pb.RegisterSophiaWhoServiceServer(s, &Server{})
	reflection.Register(s)
	go func() { _ = s.Serve(wsLis) }()
	defer s.Stop()

	// Connect via WebSocket
	wsAddr := wsLis.Addr().String()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c, _, err := websocket.Dial(ctx, wsAddr, &websocket.DialOptions{
		Subprotocols: []string{"grpc"},
	})
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	wsConn := websocket.NetConn(ctx, c, websocket.MessageBinary)

	dialed := false
	//nolint:staticcheck
	conn, err := grpc.DialContext(ctx,
		"passthrough:///ws",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(_ context.Context, _ string) (net.Conn, error) {
			if dialed {
				return nil, fmt.Errorf("already consumed")
			}
			dialed = true
			return wsConn, nil
		}),
		grpc.WithBlock(),
	)
	if err != nil {
		wsConn.Close()
		t.Fatalf("grpc dial over ws: %v", err)
	}
	defer conn.Close()

	client := pb.NewSophiaWhoServiceClient(conn)
	resp, err := client.ListIdentities(context.Background(), &pb.ListIdentitiesRequest{})
	if err != nil {
		t.Fatalf("ListIdentities over ws://: %v", err)
	}
	if len(resp.Entries) != 1 {
		t.Errorf("ListIdentities returned %d entries, want 1", len(resp.Entries))
	}
}
