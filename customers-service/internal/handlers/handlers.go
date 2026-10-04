// Package handlers implements gen.StrictServerInterface. Every /me/…
// operation resolves strictly through the caller's verified identity — the
// gateway assertion's `sub`, read through auth.RequireCaller — and never
// through anything the client supplies. The gateway has already enforced each
// operation's scope before the request reaches here (api-management); the
// only authorization question left to this service is whether a row belongs
// to the caller, and that answer is 404, never 403.
package handlers

import (
	"context"
	"net/url"
	"strconv"
	"strings"

	"customers-service/internal/auth"
	"customers-service/internal/gen"
	"customers-service/internal/models"
	"customers-service/internal/store"
)

const (
	defaultLimit = 20
	maxLimit     = 100
)

// Server implements gen.StrictServerInterface over an in-memory Store.
type Server struct {
	store *store.Store
}

// New returns a Server backed by the given Store.
func New(s *store.Store) *Server {
	return &Server{store: s}
}

// GetHealth is public (security: []): it reads no identity at all.
func (s *Server) GetHealth(ctx context.Context, request gen.GetHealthRequestObject) (gen.GetHealthResponseObject, error) {
	return gen.GetHealth200Response{}, nil
}

// GetMyProfile returns the caller's own profile, auto-provisioning a blank one
// on first access — there is no separate signup step.
func (s *Server) GetMyProfile(ctx context.Context, request gen.GetMyProfileRequestObject) (gen.GetMyProfileResponseObject, error) {
	caller, err := auth.RequireCaller(ctx)
	if err != nil {
		return nil, err
	}
	c := s.store.GetOrCreateProfile(caller.UserID)
	return gen.GetMyProfile200JSONResponse(toGenCustomer(c)), nil
}

// UpdateMyProfile creates or updates the caller's own profile.
func (s *Server) UpdateMyProfile(ctx context.Context, request gen.UpdateMyProfileRequestObject) (gen.UpdateMyProfileResponseObject, error) {
	caller, err := auth.RequireCaller(ctx)
	if err != nil {
		return nil, err
	}
	if request.Body == nil {
		return gen.UpdateMyProfile400JSONResponse{Code: 400, Message: "request body is required"}, nil
	}
	name := strings.TrimSpace(request.Body.Name)
	email := strings.TrimSpace(request.Body.Email)
	if name == "" || email == "" {
		return gen.UpdateMyProfile400JSONResponse{Code: 400, Message: "name and email are required"}, nil
	}
	c := s.store.PutProfile(caller.UserID, name, email)
	return gen.UpdateMyProfile200JSONResponse(toGenCustomer(c)), nil
}

// ListMyAddresses returns the caller's own shipping addresses, paginated.
func (s *Server) ListMyAddresses(ctx context.Context, request gen.ListMyAddressesRequestObject) (gen.ListMyAddressesResponseObject, error) {
	caller, err := auth.RequireCaller(ctx)
	if err != nil {
		return nil, err
	}
	all := s.store.ListAddresses(caller.UserID)

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

	count := len(all)
	data := make([]gen.Address, 0)
	if offset < count {
		end := offset + limit
		if end > count {
			end = count
		}
		for _, a := range all[offset:end] {
			data = append(data, toGenAddress(a))
		}
	}

	resp := gen.ListMyAddresses200JSONResponse{
		Count: count,
		Data:  data,
	}
	if offset+limit < count {
		resp.Next = pageURI(limit, offset+limit)
	}
	if offset > 0 {
		prevOffset := offset - limit
		if prevOffset < 0 {
			prevOffset = 0
		}
		resp.Previous = pageURI(limit, prevOffset)
	}
	return resp, nil
}

// AddMyAddress adds a new shipping address for the caller.
func (s *Server) AddMyAddress(ctx context.Context, request gen.AddMyAddressRequestObject) (gen.AddMyAddressResponseObject, error) {
	caller, err := auth.RequireCaller(ctx)
	if err != nil {
		return nil, err
	}
	in, verr := validateAddressInput(request.Body)
	if verr != "" {
		return gen.AddMyAddress400JSONResponse{Code: 400, Message: verr}, nil
	}
	a := s.store.AddAddress(caller.UserID, in)
	return gen.AddMyAddress201JSONResponse(toGenAddress(a)), nil
}

// UpdateMyAddress updates one of the caller's own addresses. An id that is
// not in the caller's own collection — whether unknown or another caller's —
// is a 404: the row simply is not there.
func (s *Server) UpdateMyAddress(ctx context.Context, request gen.UpdateMyAddressRequestObject) (gen.UpdateMyAddressResponseObject, error) {
	caller, err := auth.RequireCaller(ctx)
	if err != nil {
		return nil, err
	}
	in, verr := validateAddressInput(request.Body)
	if verr != "" {
		return gen.UpdateMyAddress400JSONResponse{Code: 400, Message: verr}, nil
	}
	a, ok := s.store.UpdateAddress(caller.UserID, request.AddressID, in)
	if !ok {
		return gen.UpdateMyAddress404JSONResponse{Code: 404, Message: "no such address for the caller"}, nil
	}
	return gen.UpdateMyAddress200JSONResponse(toGenAddress(a)), nil
}

// RemoveMyAddress removes one of the caller's own addresses.
func (s *Server) RemoveMyAddress(ctx context.Context, request gen.RemoveMyAddressRequestObject) (gen.RemoveMyAddressResponseObject, error) {
	caller, err := auth.RequireCaller(ctx)
	if err != nil {
		return nil, err
	}
	if ok := s.store.RemoveAddress(caller.UserID, request.AddressID); !ok {
		return gen.RemoveMyAddress404JSONResponse{Code: 404, Message: "no such address for the caller"}, nil
	}
	return gen.RemoveMyAddress204Response{}, nil
}

func validateAddressInput(body *gen.AddressInput) (models.Address, string) {
	if body == nil {
		return models.Address{}, "request body is required"
	}
	line1 := strings.TrimSpace(body.Line1)
	city := strings.TrimSpace(body.City)
	region := strings.TrimSpace(body.Region)
	postalCode := strings.TrimSpace(body.PostalCode)
	country := strings.TrimSpace(body.Country)
	if line1 == "" || city == "" || region == "" || postalCode == "" || country == "" {
		return models.Address{}, "line1, city, region, postalCode and country are required"
	}
	return models.Address{
		Line1:      line1,
		Line2:      strings.TrimSpace(body.Line2),
		City:       city,
		Region:     region,
		PostalCode: postalCode,
		Country:    country,
		IsDefault:  body.IsDefault,
	}, ""
}

func toGenCustomer(c models.Customer) gen.Customer {
	return gen.Customer{ID: c.ID, Name: c.Name, Email: c.Email}
}

func toGenAddress(a models.Address) gen.Address {
	return gen.Address{
		ID:         a.ID,
		Line1:      a.Line1,
		Line2:      a.Line2,
		City:       a.City,
		Region:     a.Region,
		PostalCode: a.PostalCode,
		Country:    a.Country,
		IsDefault:  a.IsDefault,
	}
}

// pageURI builds the relative URI ListMyAddresses' next/previous point to.
func pageURI(limit, offset int) string {
	v := url.Values{}
	v.Set("limit", strconv.Itoa(limit))
	v.Set("offset", strconv.Itoa(offset))
	return "/me/addresses?" + v.Encode()
}
