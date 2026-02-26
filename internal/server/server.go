// Package server implements the gRPC service for Sophia Who?.
package server

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/organic-programming/go-holons/pkg/transport"
	pb "github.com/organic-programming/sophia-who/gen/go/sophia_who/v1"
	"github.com/organic-programming/sophia-who/pkg/identity"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	grpcReflection "google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

// Server implements the SophiaWhoService gRPC interface.
type Server struct {
	pb.UnimplementedSophiaWhoServiceServer
}

// CreateIdentity creates a new holon identity from a gRPC request.
func (s *Server) CreateIdentity(ctx context.Context, req *pb.CreateIdentityRequest) (*pb.CreateIdentityResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	if strings.TrimSpace(req.GivenName) == "" {
		return nil, status.Error(codes.InvalidArgument, "given_name is required")
	}
	if strings.TrimSpace(req.FamilyName) == "" {
		return nil, status.Error(codes.InvalidArgument, "family_name is required")
	}
	if strings.TrimSpace(req.Motto) == "" {
		return nil, status.Error(codes.InvalidArgument, "motto is required")
	}
	if strings.TrimSpace(req.Composer) == "" {
		return nil, status.Error(codes.InvalidArgument, "composer is required")
	}

	id := identity.New()

	id.GivenName = req.GivenName
	id.FamilyName = req.FamilyName
	id.Motto = req.Motto
	id.Composer = req.Composer
	id.Clade = cladeToString(req.Clade)
	id.Reproduction = reproductionToString(req.Reproduction)

	if req.Lang != "" {
		id.Lang = req.Lang
	}
	if len(req.Aliases) > 0 {
		id.Aliases = req.Aliases
	}

	outputDir := req.OutputDir
	if outputDir == "" {
		dirName := strings.ToLower(id.GivenName + "-" + strings.TrimSuffix(id.FamilyName, "?"))
		dirName = strings.ReplaceAll(dirName, " ", "-")
		outputDir = filepath.Join(".holon", dirName)
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, status.Errorf(codes.Internal, "cannot create directory: %v", err)
	}

	outputPath := filepath.Join(outputDir, "HOLON.md")
	if err := identity.WriteHolonMD(id, outputPath); err != nil {
		return nil, status.Errorf(codes.Internal, "write HOLON.md: %v", err)
	}

	return &pb.CreateIdentityResponse{
		Identity: toProto(id),
		FilePath: outputPath,
	}, nil
}

// ShowIdentity retrieves a holon's identity by UUID.
func (s *Server) ShowIdentity(ctx context.Context, req *pb.ShowIdentityRequest) (*pb.ShowIdentityResponse, error) {
	if req == nil || strings.TrimSpace(req.Uuid) == "" {
		return nil, status.Error(codes.InvalidArgument, "uuid is required")
	}

	path, err := identity.FindByUUID(".", req.Uuid)
	if err != nil {
		if isIdentityNotFound(err) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "resolve holon by uuid: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot read %s: %v", path, err)
	}

	id, _, err := identity.ParseFrontmatter(data)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "parse HOLON.md: %v", err)
	}

	return &pb.ShowIdentityResponse{
		Identity:   toProto(id),
		FilePath:   path,
		RawContent: string(data),
	}, nil
}

// ListIdentities scans the project for all known holons.
func (s *Server) ListIdentities(ctx context.Context, req *pb.ListIdentitiesRequest) (*pb.ListIdentitiesResponse, error) {
	rootDir := "."
	if req != nil && strings.TrimSpace(req.RootDir) != "" {
		rootDir = req.RootDir
	}

	holons, err := identity.FindAllWithPaths(rootDir)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "scan identities: %v", err)
	}

	entries := make([]*pb.HolonEntry, 0, len(holons))
	for _, h := range holons {
		entries = append(entries, &pb.HolonEntry{
			Identity:     toProto(h.Identity),
			Origin:       "local",
			RelativePath: relativeHolonDir(rootDir, h.Path),
		})
	}

	return &pb.ListIdentitiesResponse{Entries: entries}, nil
}

// ListenAndServe starts the gRPC server on the given transport URI.
// Supported URIs: tcp://<host>:<port>, unix://<path>, stdio://
// When reflect is true, server reflection is enabled (mandatory per Constitution).
func ListenAndServe(listenURI string, reflect bool) error {
	lis, err := transport.Listen(listenURI)
	if err != nil {
		return fmt.Errorf("listen %s: %w", listenURI, err)
	}

	s := grpc.NewServer()
	pb.RegisterSophiaWhoServiceServer(s, &Server{})
	if reflect {
		grpcReflection.Register(s)
	}

	mode := "reflection ON"
	if !reflect {
		mode = "reflection OFF"
	}
	log.Printf("Sophia Who? gRPC server listening on %s (%s)", listenURI, mode)
	return s.Serve(lis)
}

func relativeHolonDir(rootDir, holonFilePath string) string {
	dir := filepath.Dir(holonFilePath)
	rel, err := filepath.Rel(rootDir, dir)
	if err != nil {
		return filepath.Clean(dir)
	}
	return filepath.Clean(rel)
}

func isIdentityNotFound(err error) bool {
	if err == nil {
		return false
	}
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(err.Error())), "holon not found:")
}

// --- Conversion helpers (private to server package) ---

func toProto(id identity.Identity) *pb.HolonIdentity {
	return &pb.HolonIdentity{
		Uuid:         id.UUID,
		GivenName:    id.GivenName,
		FamilyName:   id.FamilyName,
		Motto:        id.Motto,
		Composer:     id.Composer,
		Clade:        stringToClade(id.Clade),
		Status:       stringToStatus(id.Status),
		Born:         id.Born,
		Parents:      id.Parents,
		Reproduction: stringToReproduction(id.Reproduction),
		Aliases:      id.Aliases,
		GeneratedBy:  id.GeneratedBy,
		Lang:         id.Lang,
		ProtoStatus:  stringToStatus(id.ProtoStatus),
	}
}

func cladeToString(c pb.Clade) string {
	m := map[pb.Clade]string{
		pb.Clade_DETERMINISTIC_PURE:       "deterministic/pure",
		pb.Clade_DETERMINISTIC_STATEFUL:   "deterministic/stateful",
		pb.Clade_DETERMINISTIC_IO_BOUND:   "deterministic/io_bound",
		pb.Clade_PROBABILISTIC_GENERATIVE: "probabilistic/generative",
		pb.Clade_PROBABILISTIC_PERCEPTUAL: "probabilistic/perceptual",
		pb.Clade_PROBABILISTIC_ADAPTIVE:   "probabilistic/adaptive",
	}
	if s, ok := m[c]; ok {
		return s
	}
	return "deterministic/pure"
}

func stringToClade(s string) pb.Clade {
	m := map[string]pb.Clade{
		"deterministic/pure":       pb.Clade_DETERMINISTIC_PURE,
		"deterministic/stateful":   pb.Clade_DETERMINISTIC_STATEFUL,
		"deterministic/io_bound":   pb.Clade_DETERMINISTIC_IO_BOUND,
		"probabilistic/generative": pb.Clade_PROBABILISTIC_GENERATIVE,
		"probabilistic/perceptual": pb.Clade_PROBABILISTIC_PERCEPTUAL,
		"probabilistic/adaptive":   pb.Clade_PROBABILISTIC_ADAPTIVE,
	}
	if c, ok := m[s]; ok {
		return c
	}
	return pb.Clade_CLADE_UNSPECIFIED
}

func stringToStatus(s string) pb.Status {
	m := map[string]pb.Status{
		"draft":      pb.Status_DRAFT,
		"stable":     pb.Status_STABLE,
		"deprecated": pb.Status_DEPRECATED,
		"dead":       pb.Status_DEAD,
	}
	if st, ok := m[s]; ok {
		return st
	}
	return pb.Status_STATUS_UNSPECIFIED
}

func reproductionToString(r pb.ReproductionMode) string {
	m := map[pb.ReproductionMode]string{
		pb.ReproductionMode_MANUAL:      "manual",
		pb.ReproductionMode_ASSISTED:    "assisted",
		pb.ReproductionMode_AUTOMATIC:   "automatic",
		pb.ReproductionMode_AUTOPOIETIC: "autopoietic",
		pb.ReproductionMode_BRED:        "bred",
	}
	if s, ok := m[r]; ok {
		return s
	}
	return "manual"
}

func stringToReproduction(s string) pb.ReproductionMode {
	m := map[string]pb.ReproductionMode{
		"manual":      pb.ReproductionMode_MANUAL,
		"assisted":    pb.ReproductionMode_ASSISTED,
		"automatic":   pb.ReproductionMode_AUTOMATIC,
		"autopoietic": pb.ReproductionMode_AUTOPOIETIC,
		"bred":        pb.ReproductionMode_BRED,
	}
	if r, ok := m[s]; ok {
		return r
	}
	return pb.ReproductionMode_REPRODUCTION_UNSPECIFIED
}
