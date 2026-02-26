package server

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	holonsgrpcclient "github.com/organic-programming/go-holons/pkg/grpcclient"
	"github.com/organic-programming/go-holons/pkg/transport"
	pb "github.com/organic-programming/sophia-who/gen/go/sophia_who/v1"
	"github.com/organic-programming/sophia-who/pkg/identity"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func startContractMemClient(t *testing.T, root string) (pb.SophiaWhoServiceClient, func()) {
	t.Helper()

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir %s: %v", root, err)
	}

	mem := transport.NewMemListener()
	srv := grpc.NewServer()
	pb.RegisterSophiaWhoServiceServer(srv, &Server{})
	go func() {
		_ = srv.Serve(mem)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	conn, err := holonsgrpcclient.DialMem(ctx, mem)
	if err != nil {
		cancel()
		srv.Stop()
		_ = os.Chdir(originalWD)
		t.Fatalf("DialMem: %v", err)
	}

	cleanup := func() {
		_ = conn.Close()
		srv.Stop()
		cancel()
		_ = os.Chdir(originalWD)
	}

	return pb.NewSophiaWhoServiceClient(conn), cleanup
}

func validCreateReq(outputDir string) *pb.CreateIdentityRequest {
	return &pb.CreateIdentityRequest{
		GivenName:    "sophia",
		FamilyName:   "Contract",
		Motto:        "Contract-driven testing.",
		Composer:     "test-suite",
		Clade:        pb.Clade_DETERMINISTIC_PURE,
		Reproduction: pb.ReproductionMode_MANUAL,
		Lang:         "go",
		OutputDir:    outputDir,
	}
}

func TestContractCreateIdentityNominal(t *testing.T) {
	root := t.TempDir()
	client, cleanup := startContractMemClient(t, root)
	defer cleanup()

	resp, err := client.CreateIdentity(context.Background(), validCreateReq(filepath.Join("holons", "sophia-contract")))
	if err != nil {
		t.Fatalf("CreateIdentity failed: %v", err)
	}

	if resp.GetIdentity().GetUuid() == "" {
		t.Fatal("expected generated UUID")
	}
	if _, err := uuid.Parse(resp.GetIdentity().GetUuid()); err != nil {
		t.Fatalf("uuid parse failed: %v", err)
	}
	if resp.GetIdentity().GetGivenName() != "sophia" {
		t.Fatalf("given_name = %q, want %q", resp.GetIdentity().GetGivenName(), "sophia")
	}
	if resp.GetIdentity().GetComposer() != "test-suite" {
		t.Fatalf("composer = %q, want %q", resp.GetIdentity().GetComposer(), "test-suite")
	}
	if resp.GetIdentity().GetLang() != "go" {
		t.Fatalf("lang = %q, want %q", resp.GetIdentity().GetLang(), "go")
	}
	if resp.GetFilePath() == "" {
		t.Fatal("expected file_path")
	}
	if _, err := os.Stat(resp.GetFilePath()); err != nil {
		t.Fatalf("expected created file at %s: %v", resp.GetFilePath(), err)
	}

	data, err := os.ReadFile(resp.GetFilePath())
	if err != nil {
		t.Fatalf("read created file: %v", err)
	}
	parsed, _, err := identity.ParseFrontmatter(data)
	if err != nil {
		t.Fatalf("parse created HOLON.md: %v", err)
	}
	if parsed.UUID != resp.GetIdentity().GetUuid() {
		t.Fatalf("parsed uuid = %q, want %q", parsed.UUID, resp.GetIdentity().GetUuid())
	}
}

func TestContractCreateIdentityMissingGivenName(t *testing.T) {
	root := t.TempDir()
	client, cleanup := startContractMemClient(t, root)
	defer cleanup()

	req := validCreateReq(filepath.Join("holons", "missing-given"))
	req.GivenName = ""

	_, err := client.CreateIdentity(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for missing given_name")
	}
	if got := status.Code(err); got != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v", got, codes.InvalidArgument)
	}
}

func TestContractCreateIdentityMissingComposer(t *testing.T) {
	root := t.TempDir()
	client, cleanup := startContractMemClient(t, root)
	defer cleanup()

	req := validCreateReq(filepath.Join("holons", "missing-composer"))
	req.Composer = ""

	_, err := client.CreateIdentity(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for missing composer")
	}
	if got := status.Code(err); got != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v", got, codes.InvalidArgument)
	}
}

func TestContractShowIdentityNominal(t *testing.T) {
	root := t.TempDir()
	client, cleanup := startContractMemClient(t, root)
	defer cleanup()

	created, err := client.CreateIdentity(context.Background(), validCreateReq(filepath.Join("holons", "show-nominal")))
	if err != nil {
		t.Fatalf("CreateIdentity failed: %v", err)
	}

	resp, err := client.ShowIdentity(context.Background(), &pb.ShowIdentityRequest{Uuid: created.GetIdentity().GetUuid()})
	if err != nil {
		t.Fatalf("ShowIdentity failed: %v", err)
	}
	if resp.GetIdentity().GetUuid() != created.GetIdentity().GetUuid() {
		t.Fatalf("uuid = %q, want %q", resp.GetIdentity().GetUuid(), created.GetIdentity().GetUuid())
	}
	if resp.GetIdentity().GetGivenName() != created.GetIdentity().GetGivenName() {
		t.Fatalf("given_name = %q, want %q", resp.GetIdentity().GetGivenName(), created.GetIdentity().GetGivenName())
	}
	if resp.GetIdentity().GetComposer() != created.GetIdentity().GetComposer() {
		t.Fatalf("composer = %q, want %q", resp.GetIdentity().GetComposer(), created.GetIdentity().GetComposer())
	}
}

func TestContractShowIdentityNotFound(t *testing.T) {
	root := t.TempDir()
	client, cleanup := startContractMemClient(t, root)
	defer cleanup()

	_, err := client.ShowIdentity(context.Background(), &pb.ShowIdentityRequest{Uuid: "does-not-exist"})
	if err == nil {
		t.Fatal("expected not found error")
	}
	if got := status.Code(err); got != codes.NotFound {
		t.Fatalf("status code = %v, want %v", got, codes.NotFound)
	}
}

func TestContractListIdentitiesNominal(t *testing.T) {
	root := t.TempDir()
	client, cleanup := startContractMemClient(t, root)
	defer cleanup()

	first := validCreateReq(filepath.Join("holons", "list-alpha"))
	first.GivenName = "alpha"
	first.FamilyName = "One"
	if _, err := client.CreateIdentity(context.Background(), first); err != nil {
		t.Fatalf("CreateIdentity(first): %v", err)
	}

	second := validCreateReq(filepath.Join("holons", "list-beta"))
	second.GivenName = "beta"
	second.FamilyName = "Two"
	if _, err := client.CreateIdentity(context.Background(), second); err != nil {
		t.Fatalf("CreateIdentity(second): %v", err)
	}

	resp, err := client.ListIdentities(context.Background(), &pb.ListIdentitiesRequest{RootDir: "holons"})
	if err != nil {
		t.Fatalf("ListIdentities failed: %v", err)
	}
	if len(resp.GetEntries()) != 2 {
		t.Fatalf("entries = %d, want %d", len(resp.GetEntries()), 2)
	}

	seen := map[string]bool{}
	for _, entry := range resp.GetEntries() {
		if entry.GetIdentity().GetGivenName() != "" {
			seen[entry.GetIdentity().GetGivenName()] = true
		}
		if entry.GetOrigin() != "local" {
			t.Fatalf("origin = %q, want %q", entry.GetOrigin(), "local")
		}
		if entry.GetRelativePath() == "" {
			t.Fatal("expected relative_path to be set")
		}
	}
	if !seen["alpha"] || !seen["beta"] {
		t.Fatalf("expected alpha and beta in list, got %+v", seen)
	}
}

func TestContractListIdentitiesEmptyDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "empty"), 0755); err != nil {
		t.Fatalf("mkdir empty dir: %v", err)
	}

	client, cleanup := startContractMemClient(t, root)
	defer cleanup()

	resp, err := client.ListIdentities(context.Background(), &pb.ListIdentitiesRequest{RootDir: "empty"})
	if err != nil {
		t.Fatalf("ListIdentities failed: %v", err)
	}
	if len(resp.GetEntries()) != 0 {
		t.Fatalf("entries = %d, want 0", len(resp.GetEntries()))
	}
}
