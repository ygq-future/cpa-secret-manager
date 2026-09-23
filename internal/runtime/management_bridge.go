package runtime

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"cpa-secret-manager/internal/management"
)

// managementRequest is the decoded host envelope for management.handle.
type managementRequest struct {
	Method  string
	Path    string
	Headers http.Header
	Query   url.Values
	Body    []byte
}

// managementResponse is the host envelope returned from management.handle.
type managementResponse struct {
	StatusCode int         `json:"StatusCode"`
	Headers    http.Header `json:"Headers"`
	Body       []byte      `json:"Body"`
}

type managementRoute struct {
	Method string `json:"Method"`
	Path   string `json:"Path"`
}

type managementResource struct {
	Path        string `json:"Path"`
	Menu        string `json:"Menu"`
	Description string `json:"Description"`
}

type managementRegistration struct {
	Routes    []managementRoute    `json:"routes"`
	Resources []managementResource `json:"resources,omitempty"`
}

type managementRequestWire struct {
	Method      string          `json:"Method"`
	MethodLower string          `json:"method"`
	Path        string          `json:"Path"`
	PathLower   string          `json:"path"`
	Headers     http.Header     `json:"Headers"`
	QueryRaw    json.RawMessage `json:"Query"`
	QueryLower  string          `json:"query"`
	Body        string          `json:"Body"`
	BodyLower   string          `json:"body"`
}

func managementRoutes() []managementRoute {
	return []managementRoute{
		{Method: http.MethodGet, Path: management.RouteSettings},
		{Method: http.MethodPut, Path: management.RouteSettings},
	}
}

func managementResources() []managementResource {
	return nil
}

func (r *Runtime) handleManagementRegister() []byte {
	return okEnvelope(managementRegistration{
		Routes:    managementRoutes(),
		Resources: managementResources(),
	})
}

func (r *Runtime) handleManagement(ctx context.Context, raw []byte) []byte {
	request, err := decodeManagementRequest(raw)
	if err != nil {
		return failureEnvelope("invalid_request", err.Error())
	}

	recorder := management.NewRecorder()
	r.handler.ServeHTTP(recorder, request.toHTTPRequest(ctx))
	status, headers, body := recorder.Result()

	return okEnvelope(managementResponse{StatusCode: status, Headers: headers, Body: body})
}

func decodeManagementRequest(raw []byte) (managementRequest, error) {
	if len(raw) == 0 {
		return managementRequest{}, fmt.Errorf("%w: management request is required", ErrInvalidRequest)
	}
	var wire managementRequestWire
	if err := json.Unmarshal(raw, &wire); err != nil {
		return managementRequest{}, fmt.Errorf("%w: decode management request: %v", ErrInvalidRequest, err)
	}

	request := managementRequest{
		Method:  firstNonEmpty(wire.Method, wire.MethodLower),
		Path:    firstNonEmpty(wire.Path, wire.PathLower),
		Headers: wire.Headers,
		Query:   decodeManagementQuery(wire.QueryRaw, wire.QueryLower),
	}
	if request.Method == "" || request.Path == "" {
		return managementRequest{}, fmt.Errorf("%w: management method and path are required", ErrInvalidRequest)
	}

	body, err := decodeManagementBody(wire.Body, wire.BodyLower)
	if err != nil {
		return managementRequest{}, err
	}
	request.Body = body
	return request, nil
}

func decodeManagementQuery(rawQuery json.RawMessage, legacyQuery string) url.Values {
	if len(rawQuery) > 0 {
		var encoded string
		if err := json.Unmarshal(rawQuery, &encoded); err == nil {
			if values, err := url.ParseQuery(strings.TrimPrefix(encoded, "?")); err == nil {
				return values
			}
		}
		var mapped map[string][]string
		if err := json.Unmarshal(rawQuery, &mapped); err == nil {
			return mapped
		}
	}
	if strings.TrimSpace(legacyQuery) != "" {
		if values, err := url.ParseQuery(strings.TrimPrefix(legacyQuery, "?")); err == nil {
			return values
		}
	}
	return nil
}

func decodeManagementBody(encoded string, legacy string) ([]byte, error) {
	if encoded != "" {
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return nil, fmt.Errorf("%w: decode management body: %v", ErrInvalidRequest, err)
		}
		return decoded, nil
	}
	return []byte(legacy), nil
}

func (request managementRequest) toHTTPRequest(ctx context.Context) *http.Request {
	path := request.Path
	if len(request.Query) > 0 {
		if encoded := request.Query.Encode(); encoded != "" {
			path += "?" + encoded
		}
	}
	httpRequest, err := http.NewRequestWithContext(ctx, request.Method, path, bytes.NewReader(request.Body))
	if err != nil {
		// The host always supplies a valid method and path; fall back to a GET so
		// the handler can still answer with a structured error.
		httpRequest, _ = http.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
	}
	for name, values := range request.Headers {
		for _, value := range values {
			httpRequest.Header.Add(name, value)
		}
	}
	return httpRequest
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
