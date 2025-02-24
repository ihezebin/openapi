package openapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"gopkg.in/yaml.v2"
)

type APIOpts func(*API)

// WithApplyCustomSchemaToType enables customisation of types in the OpenAPI specification.
// Apply customisation to a specific type by checking the t parameter.
// Apply customisations to all types by ignoring the t parameter.
func WithApplyCustomSchemaToType(f func(t reflect.Type, s *openapi3.Schema)) APIOpts {
	return func(api *API) {
		api.ApplyCustomSchemaToType = f
	}
}

func WithInfo(info openapi3.Info) APIOpts {
	return func(api *API) {
		api.Info = info
		if info.Title != "" {
			api.Name = info.Title
		}
	}
}

func WithServer(servers ...openapi3.Server) APIOpts {
	return func(api *API) {
		api.Servers = servers
	}
}

// NewAPI creates a new API from the router.
func NewAPI(name string, opts ...APIOpts) *API {
	api := &API{
		Name:       name,
		KnownTypes: defaultKnownTypes,
		Routes:     make(map[Pattern]MethodToRoute),
		// map of model name to schema.
		models:   make(map[string]*openapi3.Schema),
		comments: make(map[string]map[string]string),
	}
	for _, o := range opts {
		o(api)
	}
	return api
}

var defaultKnownTypes = map[reflect.Type]openapi3.Schema{
	reflect.TypeOf(time.Time{}):  *openapi3.NewDateTimeSchema(),
	reflect.TypeOf(&time.Time{}): *openapi3.NewDateTimeSchema().WithNullable(),
}

// Route models a single API route.
type Route struct {
	// Method is the HTTP method of the route, e.g. http.MethodGet
	Method Method
	// Pattern of the route, e.g. /posts/list, or /users/{id}
	Pattern Pattern
	// Params of the route.
	Params Params
	// Models used in the route.
	Models Models
	// Tags used in the route.
	Tags []string
	// OperationID for the route.
	OperationID string
	// Description for the route.
	Description string
	// Summary for the route.
	Summary string
	// Deprecated sets whether the route is deprecated.
	Deprecated bool
}

// Params is a route parameter.
type Params struct {
	// Path parameters are used in the path of the URL, e.g. /users/{id} would
	// have a name of "id".
	Path map[string]PathParam
	// Query parameters are used in the querystring of the URL, e.g. /users/?sort={sortOrder} would
	// have a name of "sort".
	Query map[string]QueryParam
	// Header parameters are used in HTTP headers
	Header map[string]HeaderParam
}

// PathParam is a paramater that's used in the path of a URL.
type PathParam struct {
	// Description of the param.
	Description string
	// Regexp is a regular expression used to validate the param.
	// An empty string means that no validation is applied.
	Regexp string
	// Type of the param (string, number, integer, boolean).
	Type PrimitiveType
	// ApplyCustomSchema customises the OpenAPI schema for the path parameter.
	ApplyCustomSchema func(s *openapi3.Parameter)
}

// QueryParam is a paramater that's used in the querystring of a URL.
type QueryParam struct {
	// Description of the param.
	Description string
	// Regexp is a regular expression used to validate the param.
	// An empty string means that no validation is applied.
	Regexp string
	// Required sets whether the querystring parameter must be present in the URL.
	Required bool
	// AllowEmpty sets whether the querystring parameter can be empty.
	AllowEmpty bool
	// Type of the param (string, number, integer, boolean).
	Type PrimitiveType
	// ApplyCustomSchema customises the OpenAPI schema for the query parameter.
	ApplyCustomSchema func(s *openapi3.Parameter)
}

// HeaderParam is a parameter that's used in HTTP headers
type HeaderParam struct {
	// Description of the param
	Description string
	// Required sets whether the header must be present
	Required bool
	// Type of the param (string, number, integer, boolean)
	Type PrimitiveType
	// ApplyCustomSchema customises the OpenAPI schema for the header parameter
	ApplyCustomSchema func(s *openapi3.Parameter)
}

type PrimitiveType string

const (
	PrimitiveTypeString  PrimitiveType = "string"
	PrimitiveTypeBool    PrimitiveType = "boolean"
	PrimitiveTypeInteger PrimitiveType = "integer"
	PrimitiveTypeFloat64 PrimitiveType = "number"
)

// MethodToRoute maps from a HTTP method to a Route.
type MethodToRoute map[Method]*Route

// Method is the HTTP method of the route, e.g. http.MethodGet
type Method string

// Pattern of the route, e.g. /posts/list, or /users/{id}
type Pattern string

// API is a model of a REST API's routes, along with their
// request and response types.
type API struct {
	// Name of the API.
	Name string
	// Info of the API.
	Info openapi3.Info
	// Servers of the API.
	Servers []openapi3.Server
	// Routes of the API.
	// From patterns, to methods, to route.
	Routes map[Pattern]MethodToRoute
	// StripPkgPaths to strip from the type names in the OpenAPI output to avoid
	// leaking internal implementation details such as internal repo names.
	//
	// This increases the risk of type clashes in the OpenAPI output, i.e. two types
	// in different namespaces that are set to be stripped, and have the same type Name
	// could clash.
	//
	// Example values could be "github.com/ihezebin/openapi".
	StripPkgPaths []string

	// Models are the models that are in use in the API.
	// It's possible to customise the models prior to generation of the OpenAPI specification
	// by editing this value.
	models map[string]*openapi3.Schema

	// KnownTypes are added to the OpenAPI specification output.
	// The default implementation:
	//   Maps time.Time to a string.
	KnownTypes map[reflect.Type]openapi3.Schema

	// comments from the package. This can be cleared once the spec has been created.
	comments map[string]map[string]string

	// ApplyCustomSchemaToType callback to customise the OpenAPI specification for a given type.
	// Apply customisation to a specific type by checking the t parameter.
	// Apply customisations to all types by ignoring the t parameter.
	ApplyCustomSchemaToType func(t reflect.Type, s *openapi3.Schema)
}

// Merge route data into the existing configuration.
// This is typically used by adapters, such as the chiadapter
// to take information that the router already knows and add it
// to the specification.
func (api *API) Merge(r Route) {
	toUpdate := api.Route(string(r.Method), string(r.Pattern))
	mergeMap(toUpdate.Params.Path, r.Params.Path)
	mergeMap(toUpdate.Params.Query, r.Params.Query)
	mergeMap(toUpdate.Params.Header, r.Params.Header)
	if toUpdate.Models.Request.Type == nil {
		toUpdate.Models.Request = r.Models.Request
	}
	mergeMap(toUpdate.Models.Responses, r.Models.Responses)
}

func mergeMap[TKey comparable, TValue any](into, from map[TKey]TValue) {
	for kf, vf := range from {
		_, ok := into[kf]
		if !ok {
			into[kf] = vf
		}
	}
}

// Spec creates an OpenAPI 3.0 specification document for the API.
func (api *API) Spec() (spec *openapi3.T, err error) {
	spec, err = api.createOpenAPI()
	if err != nil {
		return
	}
	return
}

// Json creates a JSON representation of the OpenAPI specification.
func (api *API) Json() ([]byte, error) {
	spec, err := api.Spec()
	if err != nil {
		return nil, fmt.Errorf("create spec err: %w", err)
	}

	data, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("marshal spec err: %w", err)
	}

	return data, nil
}

// Yaml creates a YAML representation of the OpenAPI specification.
func (api *API) Yaml() ([]byte, error) {
	spec, err := api.Spec()
	if err != nil {
		return nil, fmt.Errorf("create spec err: %w", err)
	}

	data, err := yaml.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("marshal spec err: %w", err)
	}

	return data, nil
}

// Route upserts a route to the API definition.
func (api *API) Route(method, pattern string) (r *Route) {
	methodToRoute, ok := api.Routes[Pattern(pattern)]
	if !ok {
		methodToRoute = make(MethodToRoute)
		api.Routes[Pattern(pattern)] = methodToRoute
	}
	route, ok := methodToRoute[Method(method)]
	if !ok {
		route = &Route{
			Method:  Method(method),
			Pattern: Pattern(pattern),
			Models: Models{
				Responses: make(map[int]Model),
			},
			Params: Params{
				Path:   make(map[string]PathParam),
				Query:  make(map[string]QueryParam),
				Header: make(map[string]HeaderParam),
			},
		}
		methodToRoute[Method(method)] = route
	}
	return route
}

// Get defines a GET request route for the given pattern.
func (api *API) Get(pattern string) (r *Route) {
	return api.Route(http.MethodGet, pattern)
}

// Head defines a HEAD request route for the given pattern.
func (api *API) Head(pattern string) (r *Route) {
	return api.Route(http.MethodHead, pattern)
}

// Post defines a POST request route for the given pattern.
func (api *API) Post(pattern string) (r *Route) {
	return api.Route(http.MethodPost, pattern)
}

// Put defines a PUT request route for the given pattern.
func (api *API) Put(pattern string) (r *Route) {
	return api.Route(http.MethodPut, pattern)
}

// Patch defines a PATCH request route for the given pattern.
func (api *API) Patch(pattern string) (r *Route) {
	return api.Route(http.MethodPatch, pattern)
}

// Delete defines a DELETE request route for the given pattern.
func (api *API) Delete(pattern string) (r *Route) {
	return api.Route(http.MethodDelete, pattern)
}

// Connect defines a CONNECT request route for the given pattern.
func (api *API) Connect(pattern string) (r *Route) {
	return api.Route(http.MethodConnect, pattern)
}

// Options defines an OPTIONS request route for the given pattern.
func (api *API) Options(pattern string) (r *Route) {
	return api.Route(http.MethodOptions, pattern)
}

// Trace defines an TRACE request route for the given pattern.
func (api *API) Trace(pattern string) (r *Route) {
	return api.Route(http.MethodTrace, pattern)
}

// HasResponseModel configures a response for the route.
// Example:
//
//	api.Get("/user").HasResponseModel(http.StatusOK, openapi.ModelOf[User]())
func (rm *Route) HasResponseModel(status int, response Model) *Route {
	rm.Models.Responses[status] = response
	return rm
}

// HasResponseModel configures the request model of the route.
// Example:
//
//	api.Post("/user").HasRequestModel(http.StatusOK, openapi.ModelOf[User]())
func (rm *Route) HasRequestModel(request Model) *Route {
	rm.Models.Request = request
	return rm
}

// HasPathParameter configures a path parameter for the route.
func (rm *Route) HasPathParameter(name string, p PathParam) *Route {
	rm.Params.Path[name] = p
	return rm
}

// HasQueryParameter configures a query parameter for the route.
func (rm *Route) HasQueryParameter(name string, q QueryParam) *Route {
	rm.Params.Query[name] = q
	return rm
}

// HasHeaderParameter configures a header parameter for the route
func (rm *Route) HasHeaderParameter(name string, h HeaderParam) *Route {
	rm.Params.Header[name] = h
	return rm
}

// HasTags sets the tags for the route.
func (rm *Route) HasTags(tags []string) *Route {
	rm.Tags = append(rm.Tags, tags...)
	return rm
}

// HasOperationID sets the OperationID for the route.
func (rm *Route) HasOperationID(operationID string) *Route {
	rm.OperationID = operationID
	return rm
}

// HasDescription sets the description for the route.
func (rm *Route) HasDescription(description string) *Route {
	rm.Description = description
	return rm
}

// HasSummary sets the summary for the route.
func (rm *Route) HasSummary(summary string) *Route {
	rm.Summary = summary
	return rm
}

// HasDeprecated sets the deprecated for the route.
func (rm *Route) HasDeprecated(deprecated bool) *Route {
	rm.Deprecated = deprecated
	return rm
}

// HasResponseHeader 为指定状态码配置响应头
func (rm *Route) HasResponseHeader(status int, name string, h HeaderParam) *Route {
	if rm.Models.ResponseHeaders == nil {
		rm.Models.ResponseHeaders = make(map[int]map[string]HeaderParam)
	}
	if rm.Models.ResponseHeaders[status] == nil {
		rm.Models.ResponseHeaders[status] = make(map[string]HeaderParam)
	}
	rm.Models.ResponseHeaders[status][name] = h
	return rm
}

// Models defines the models used by a route.
type Models struct {
	Request         Model
	Responses       map[int]Model
	ResponseHeaders map[int]map[string]HeaderParam
}

// ModelOf creates a model of type T.
func ModelOf[T any]() Model {
	var t T
	m := Model{
		Type: reflect.TypeFor[T](),
	}
	if sm, ok := any(t).(CustomSchemaApplier); ok {
		m.s = sm.ApplyCustomSchema
	}
	return m
}

// ModelFromType creates a model of type t.
func ModelFromType(t reflect.Type) Model {
	return modelFromType(t)
}

func modelFromType(t reflect.Type) Model {
	m := Model{
		Type: t,
	}
	if sm, ok := reflect.New(t).Interface().(CustomSchemaApplier); ok {
		m.s = sm.ApplyCustomSchema
	}
	return m
}

// CustomSchemaApplier is a type that customises its OpenAPI schema.
type CustomSchemaApplier interface {
	ApplyCustomSchema(s *openapi3.Schema)
}

var _ CustomSchemaApplier = Model{}

// Model is a model used in one or more routes.
type Model struct {
	Type reflect.Type
	s    func(s *openapi3.Schema)
}

func (m Model) ApplyCustomSchema(s *openapi3.Schema) {
	if m.s == nil {
		return
	}
	m.s(s)
}
