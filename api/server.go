// Package api provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/deepmap/oapi-codegen version v1.15.0 DO NOT EDIT.
package api

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/labstack/echo/v4"
	"github.com/oapi-codegen/runtime"
	strictecho "github.com/oapi-codegen/runtime/strictmiddleware/echo"
)

// ServerInterface represents all server handlers.
type ServerInterface interface {
	// set affiliate code for donation
	// (POST /donation/affiliate)
	SetAffiliateCode(ctx echo.Context) error
	// get general donation data
	// (GET /donation/data/)
	DonationData(ctx echo.Context) error
	// get the donation data for the wallet address in question
	// (GET /user/data/{address})
	UserData(ctx echo.Context, address string) error
}

// ServerInterfaceWrapper converts echo contexts to parameters.
type ServerInterfaceWrapper struct {
	Handler ServerInterface
}

// SetAffiliateCode converts echo context to params.
func (w *ServerInterfaceWrapper) SetAffiliateCode(ctx echo.Context) error {
	var err error

	// Invoke the callback with all the unmarshaled arguments
	err = w.Handler.SetAffiliateCode(ctx)
	return err
}

// DonationData converts echo context to params.
func (w *ServerInterfaceWrapper) DonationData(ctx echo.Context) error {
	var err error

	// Invoke the callback with all the unmarshaled arguments
	err = w.Handler.DonationData(ctx)
	return err
}

// UserData converts echo context to params.
func (w *ServerInterfaceWrapper) UserData(ctx echo.Context) error {
	var err error
	// ------------- Path parameter "address" -------------
	var address string

	err = runtime.BindStyledParameterWithLocation("simple", false, "address", runtime.ParamLocationPath, ctx.Param("address"), &address)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter address: %s", err))
	}

	// Invoke the callback with all the unmarshaled arguments
	err = w.Handler.UserData(ctx, address)
	return err
}

// This is a simple interface which specifies echo.Route addition functions which
// are present on both echo.Echo and echo.Group, since we want to allow using
// either of them for path registration
type EchoRouter interface {
	CONNECT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	DELETE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	GET(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	HEAD(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	OPTIONS(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	PATCH(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	POST(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	PUT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	TRACE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
}

// RegisterHandlers adds each server route to the EchoRouter.
func RegisterHandlers(router EchoRouter, si ServerInterface) {
	RegisterHandlersWithBaseURL(router, si, "")
}

// Registers handlers, and prepends BaseURL to the paths, so that the paths
// can be served under a prefix.
func RegisterHandlersWithBaseURL(router EchoRouter, si ServerInterface, baseURL string) {

	wrapper := ServerInterfaceWrapper{
		Handler: si,
	}

	router.POST(baseURL+"/donation/affiliate", wrapper.SetAffiliateCode)
	router.GET(baseURL+"/donation/data/", wrapper.DonationData)
	router.GET(baseURL+"/user/data/:address", wrapper.UserData)

}

type SetAffiliateCodeRequestObject struct {
	Body *SetAffiliateCodeJSONRequestBody
}

type SetAffiliateCodeResponseObject interface {
	VisitSetAffiliateCodeResponse(w http.ResponseWriter) error
}

type SetAffiliateCode200Response struct {
}

func (response SetAffiliateCode200Response) VisitSetAffiliateCodeResponse(w http.ResponseWriter) error {
	w.WriteHeader(200)
	return nil
}

type SetAffiliateCode400JSONResponse Error

func (response SetAffiliateCode400JSONResponse) VisitSetAffiliateCodeResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(400)

	return json.NewEncoder(w).Encode(response)
}

type SetAffiliateCode401JSONResponse Error

func (response SetAffiliateCode401JSONResponse) VisitSetAffiliateCodeResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(401)

	return json.NewEncoder(w).Encode(response)
}

type SetAffiliateCode5XXJSONResponse struct {
	Body       Error
	StatusCode int
}

func (response SetAffiliateCode5XXJSONResponse) VisitSetAffiliateCodeResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(response.StatusCode)

	return json.NewEncoder(w).Encode(response.Body)
}

type DonationDataRequestObject struct {
}

type DonationDataResponseObject interface {
	VisitDonationDataResponse(w http.ResponseWriter) error
}

type DonationData200JSONResponse DonationData

func (response DonationData200JSONResponse) VisitDonationDataResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)

	return json.NewEncoder(w).Encode(response)
}

type DonationData400JSONResponse Error

func (response DonationData400JSONResponse) VisitDonationDataResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(400)

	return json.NewEncoder(w).Encode(response)
}

type DonationData401JSONResponse Error

func (response DonationData401JSONResponse) VisitDonationDataResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(401)

	return json.NewEncoder(w).Encode(response)
}

type DonationData5XXJSONResponse struct {
	Body       Error
	StatusCode int
}

func (response DonationData5XXJSONResponse) VisitDonationDataResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(response.StatusCode)

	return json.NewEncoder(w).Encode(response.Body)
}

type UserDataRequestObject struct {
	Address string `json:"address"`
}

type UserDataResponseObject interface {
	VisitUserDataResponse(w http.ResponseWriter) error
}

type UserData200JSONResponse UserData

func (response UserData200JSONResponse) VisitUserDataResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)

	return json.NewEncoder(w).Encode(response)
}

type UserData400JSONResponse Error

func (response UserData400JSONResponse) VisitUserDataResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(400)

	return json.NewEncoder(w).Encode(response)
}

type UserData401JSONResponse Error

func (response UserData401JSONResponse) VisitUserDataResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(401)

	return json.NewEncoder(w).Encode(response)
}

type UserData5XXJSONResponse struct {
	Body       Error
	StatusCode int
}

func (response UserData5XXJSONResponse) VisitUserDataResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(response.StatusCode)

	return json.NewEncoder(w).Encode(response.Body)
}

// StrictServerInterface represents all server handlers.
type StrictServerInterface interface {
	// set affiliate code for donation
	// (POST /donation/affiliate)
	SetAffiliateCode(ctx context.Context, request SetAffiliateCodeRequestObject) (SetAffiliateCodeResponseObject, error)
	// get general donation data
	// (GET /donation/data/)
	DonationData(ctx context.Context, request DonationDataRequestObject) (DonationDataResponseObject, error)
	// get the donation data for the wallet address in question
	// (GET /user/data/{address})
	UserData(ctx context.Context, request UserDataRequestObject) (UserDataResponseObject, error)
}

type StrictHandlerFunc = strictecho.StrictEchoHandlerFunc
type StrictMiddlewareFunc = strictecho.StrictEchoMiddlewareFunc

func NewStrictHandler(ssi StrictServerInterface, middlewares []StrictMiddlewareFunc) ServerInterface {
	return &strictHandler{ssi: ssi, middlewares: middlewares}
}

type strictHandler struct {
	ssi         StrictServerInterface
	middlewares []StrictMiddlewareFunc
}

// SetAffiliateCode operation middleware
func (sh *strictHandler) SetAffiliateCode(ctx echo.Context) error {
	var request SetAffiliateCodeRequestObject

	var body SetAffiliateCodeJSONRequestBody
	if err := ctx.Bind(&body); err != nil {
		return err
	}
	request.Body = &body

	handler := func(ctx echo.Context, request interface{}) (interface{}, error) {
		return sh.ssi.SetAffiliateCode(ctx.Request().Context(), request.(SetAffiliateCodeRequestObject))
	}
	for _, middleware := range sh.middlewares {
		handler = middleware(handler, "SetAffiliateCode")
	}

	response, err := handler(ctx, request)

	if err != nil {
		return err
	} else if validResponse, ok := response.(SetAffiliateCodeResponseObject); ok {
		return validResponse.VisitSetAffiliateCodeResponse(ctx.Response())
	} else if response != nil {
		return fmt.Errorf("unexpected response type: %T", response)
	}
	return nil
}

// DonationData operation middleware
func (sh *strictHandler) DonationData(ctx echo.Context) error {
	var request DonationDataRequestObject

	handler := func(ctx echo.Context, request interface{}) (interface{}, error) {
		return sh.ssi.DonationData(ctx.Request().Context(), request.(DonationDataRequestObject))
	}
	for _, middleware := range sh.middlewares {
		handler = middleware(handler, "DonationData")
	}

	response, err := handler(ctx, request)

	if err != nil {
		return err
	} else if validResponse, ok := response.(DonationDataResponseObject); ok {
		return validResponse.VisitDonationDataResponse(ctx.Response())
	} else if response != nil {
		return fmt.Errorf("unexpected response type: %T", response)
	}
	return nil
}

// UserData operation middleware
func (sh *strictHandler) UserData(ctx echo.Context, address string) error {
	var request UserDataRequestObject

	request.Address = address

	handler := func(ctx echo.Context, request interface{}) (interface{}, error) {
		return sh.ssi.UserData(ctx.Request().Context(), request.(UserDataRequestObject))
	}
	for _, middleware := range sh.middlewares {
		handler = middleware(handler, "UserData")
	}

	response, err := handler(ctx, request)

	if err != nil {
		return err
	} else if validResponse, ok := response.(UserDataResponseObject); ok {
		return validResponse.VisitUserDataResponse(ctx.Response())
	} else if response != nil {
		return fmt.Errorf("unexpected response type: %T", response)
	}
	return nil
}

// Base64 encoded, gzipped, json marshaled Swagger object
var swaggerSpec = []string{

	"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",
	"dve/AAAA///7muvQqR4AAA==",
}

// GetSwagger returns the content of the embedded swagger specification file
// or error if failed to decode
func decodeSpec() ([]byte, error) {
	zipped, err := base64.StdEncoding.DecodeString(strings.Join(swaggerSpec, ""))
	if err != nil {
		return nil, fmt.Errorf("error base64 decoding spec: %w", err)
	}
	zr, err := gzip.NewReader(bytes.NewReader(zipped))
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %w", err)
	}
	var buf bytes.Buffer
	_, err = buf.ReadFrom(zr)
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %w", err)
	}

	return buf.Bytes(), nil
}

var rawSpec = decodeSpecCached()

// a naive cached of a decoded swagger spec
func decodeSpecCached() func() ([]byte, error) {
	data, err := decodeSpec()
	return func() ([]byte, error) {
		return data, err
	}
}

// Constructs a synthetic filesystem for resolving external references when loading openapi specifications.
func PathToRawSpec(pathToFile string) map[string]func() ([]byte, error) {
	res := make(map[string]func() ([]byte, error))
	if len(pathToFile) > 0 {
		res[pathToFile] = rawSpec
	}

	return res
}

// GetSwagger returns the Swagger specification corresponding to the generated code
// in this file. The external references of Swagger specification are resolved.
// The logic of resolving external references is tightly connected to "import-mapping" feature.
// Externally referenced files must be embedded in the corresponding golang packages.
// Urls can be supported but this task was out of the scope.
func GetSwagger() (swagger *openapi3.T, err error) {
	resolvePath := PathToRawSpec("")

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	loader.ReadFromURIFunc = func(loader *openapi3.Loader, url *url.URL) ([]byte, error) {
		pathToFile := url.String()
		pathToFile = path.Clean(pathToFile)
		getSpec, ok := resolvePath[pathToFile]
		if !ok {
			err1 := fmt.Errorf("path not found: %s", pathToFile)
			return nil, err1
		}
		return getSpec()
	}
	var specData []byte
	specData, err = rawSpec()
	if err != nil {
		return
	}
	swagger, err = loader.LoadFromData(specData)
	if err != nil {
		return
	}
	return
}
