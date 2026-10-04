// Package handlers implements catalog-service's strict OpenAPI server
// interface. Reach (public vs books:manage) is enforced entirely at the API
// gateway per openapi.yaml's security blocks — these handlers hold no
// operation-to-scope table of their own (go skill, api-management skill).
package handlers

import (
	"context"
	"errors"
	"strconv"

	"catalog-service/internal/gen"
	"catalog-service/internal/store"
)

const (
	defaultLimit = 20
	maxLimit     = 100
)

// Server implements gen.StrictServerInterface against an in-memory store.
type Server struct {
	store *store.Store
}

// New returns a Server backed by a freshly seeded in-memory store.
func New(s *store.Store) *Server {
	return &Server{store: s}
}

var _ gen.StrictServerInterface = (*Server)(nil)

// ListBooks is public (security: []): no caller identity is read.
func (s *Server) ListBooks(_ context.Context, request gen.ListBooksRequestObject) (gen.ListBooksResponseObject, error) {
	limit := request.Params.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	offset := request.Params.Offset
	if offset < 0 {
		offset = 0
	}

	books, count := s.store.List(request.Params.Q, request.Params.Tag, limit, offset)

	resp := gen.ListBooks200JSONResponse{
		Count: count,
		Data:  books,
	}
	if offset+limit < count {
		next := strconv.Itoa(offset + limit)
		resp.Next = "/books?offset=" + next + "&limit=" + strconv.Itoa(limit)
	}
	if offset > 0 {
		prevOffset := offset - limit
		if prevOffset < 0 {
			prevOffset = 0
		}
		resp.Previous = "/books?offset=" + strconv.Itoa(prevOffset) + "&limit=" + strconv.Itoa(limit)
	}
	return resp, nil
}

// GetBook is public (security: []): no caller identity is read.
func (s *Server) GetBook(_ context.Context, request gen.GetBookRequestObject) (gen.GetBookResponseObject, error) {
	b, err := s.store.Get(request.BookID)
	if err != nil {
		var nf store.ErrNotFound
		if errors.As(err, &nf) {
			return gen.GetBook404JSONResponse{Code: 404, Message: "no such book"}, nil
		}
		return nil, err
	}
	return gen.GetBook200JSONResponse(b), nil
}

// CreateBook is restricted to books:manage, enforced at the gateway — this
// handler performs no scope check of its own.
func (s *Server) CreateBook(_ context.Context, request gen.CreateBookRequestObject) (gen.CreateBookResponseObject, error) {
	if request.Body == nil {
		return gen.CreateBook400JSONResponse{Code: 400, Message: "request body is required"}, nil
	}
	if msg, ok := validateBookInput(*request.Body); !ok {
		return gen.CreateBook400JSONResponse{Code: 400, Message: msg}, nil
	}
	b := s.store.Create(*request.Body)
	return gen.CreateBook201JSONResponse(b), nil
}

// UpdateBook is restricted to books:manage, enforced at the gateway.
func (s *Server) UpdateBook(_ context.Context, request gen.UpdateBookRequestObject) (gen.UpdateBookResponseObject, error) {
	if request.Body == nil {
		return gen.UpdateBook400JSONResponse{Code: 400, Message: "request body is required"}, nil
	}
	if msg, ok := validateBookInput(*request.Body); !ok {
		return gen.UpdateBook400JSONResponse{Code: 400, Message: msg}, nil
	}
	b, err := s.store.Update(request.BookID, *request.Body)
	if err != nil {
		var nf store.ErrNotFound
		if errors.As(err, &nf) {
			return gen.UpdateBook404JSONResponse{Code: 404, Message: "no such book"}, nil
		}
		return nil, err
	}
	return gen.UpdateBook200JSONResponse(b), nil
}

// DeleteBook is restricted to books:manage, enforced at the gateway.
func (s *Server) DeleteBook(_ context.Context, request gen.DeleteBookRequestObject) (gen.DeleteBookResponseObject, error) {
	if err := s.store.Delete(request.BookID); err != nil {
		var nf store.ErrNotFound
		if errors.As(err, &nf) {
			return gen.DeleteBook404JSONResponse{Code: 404, Message: "no such book"}, nil
		}
		return nil, err
	}
	return gen.DeleteBook204Response{}, nil
}

// GetHealth is public (security: []): no caller identity is read.
func (s *Server) GetHealth(_ context.Context, _ gen.GetHealthRequestObject) (gen.GetHealthResponseObject, error) {
	return gen.GetHealth200Response{}, nil
}

func validateBookInput(in gen.BookInput) (msg string, ok bool) {
	if in.Title == "" {
		return "title is required", false
	}
	if in.Author == "" {
		return "author is required", false
	}
	if in.Price < 0 {
		return "price must not be negative", false
	}
	return "", true
}
