// Package server keeps the historical import path for Sophia's gRPC server.
package server

import (
	pb "github.com/organic-programming/sophia-who/gen/go/sophia_who/v1"
	"github.com/organic-programming/sophia-who/pkg/service"
)

// Server is the Sophia gRPC server implementation.
type Server struct {
	service.Server
}

// ListenAndServe starts the Sophia gRPC server on the given transport URI.
func ListenAndServe(listenURI string, reflect bool) error {
	return service.ListenAndServe(listenURI, reflect)
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
	if mode, ok := m[s]; ok {
		return mode
	}
	return pb.ReproductionMode_REPRODUCTION_UNSPECIFIED
}
