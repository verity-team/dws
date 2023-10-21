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
	// generate new affiliate code
	// (POST /affiliate/code)
	GenerateCode(ctx echo.Context, params GenerateCodeParams) error
	// get general donation data
	// (GET /donation/data)
	DonationData(ctx echo.Context) error
	// is the service alive?
	// (GET /live)
	Alive(ctx echo.Context) error
	// is the service alive and ready to do work?
	// (GET /ready)
	Ready(ctx echo.Context) error
	// get the donation data for the wallet address in question
	// (GET /user/data/{address})
	UserData(ctx echo.Context, address string) error
	// associate a wallet address with an affiliate code
	// (POST /wallet/connection)
	ConnectWallet(ctx echo.Context) error
}

// ServerInterfaceWrapper converts echo contexts to parameters.
type ServerInterfaceWrapper struct {
	Handler ServerInterface
}

// GenerateCode converts echo context to params.
func (w *ServerInterfaceWrapper) GenerateCode(ctx echo.Context) error {
	var err error

	// Parameter object where we will unmarshal all parameters from the context
	var params GenerateCodeParams

	headers := ctx.Request().Header
	// ------------- Required header parameter "delphi-key" -------------
	if valueList, found := headers[http.CanonicalHeaderKey("delphi-key")]; found {
		var DelphiKey DelphiKey
		n := len(valueList)
		if n != 1 {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Expected one value for delphi-key, got %d", n))
		}

		err = runtime.BindStyledParameterWithLocation("simple", false, "delphi-key", runtime.ParamLocationHeader, valueList[0], &DelphiKey)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter delphi-key: %s", err))
		}

		params.DelphiKey = DelphiKey
	} else {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Header parameter delphi-key is required, but not found"))
	}
	// ------------- Required header parameter "delphi-ts" -------------
	if valueList, found := headers[http.CanonicalHeaderKey("delphi-ts")]; found {
		var DelphiTs DelphiTs
		n := len(valueList)
		if n != 1 {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Expected one value for delphi-ts, got %d", n))
		}

		err = runtime.BindStyledParameterWithLocation("simple", false, "delphi-ts", runtime.ParamLocationHeader, valueList[0], &DelphiTs)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter delphi-ts: %s", err))
		}

		params.DelphiTs = DelphiTs
	} else {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Header parameter delphi-ts is required, but not found"))
	}
	// ------------- Required header parameter "delphi-signature" -------------
	if valueList, found := headers[http.CanonicalHeaderKey("delphi-signature")]; found {
		var DelphiSignature DelphiSignature
		n := len(valueList)
		if n != 1 {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Expected one value for delphi-signature, got %d", n))
		}

		err = runtime.BindStyledParameterWithLocation("simple", false, "delphi-signature", runtime.ParamLocationHeader, valueList[0], &DelphiSignature)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter delphi-signature: %s", err))
		}

		params.DelphiSignature = DelphiSignature
	} else {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Header parameter delphi-signature is required, but not found"))
	}

	// Invoke the callback with all the unmarshaled arguments
	err = w.Handler.GenerateCode(ctx, params)
	return err
}

// DonationData converts echo context to params.
func (w *ServerInterfaceWrapper) DonationData(ctx echo.Context) error {
	var err error

	// Invoke the callback with all the unmarshaled arguments
	err = w.Handler.DonationData(ctx)
	return err
}

// Alive converts echo context to params.
func (w *ServerInterfaceWrapper) Alive(ctx echo.Context) error {
	var err error

	// Invoke the callback with all the unmarshaled arguments
	err = w.Handler.Alive(ctx)
	return err
}

// Ready converts echo context to params.
func (w *ServerInterfaceWrapper) Ready(ctx echo.Context) error {
	var err error

	// Invoke the callback with all the unmarshaled arguments
	err = w.Handler.Ready(ctx)
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

// ConnectWallet converts echo context to params.
func (w *ServerInterfaceWrapper) ConnectWallet(ctx echo.Context) error {
	var err error

	// Invoke the callback with all the unmarshaled arguments
	err = w.Handler.ConnectWallet(ctx)
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

	router.POST(baseURL+"/affiliate/code", wrapper.GenerateCode)
	router.GET(baseURL+"/donation/data", wrapper.DonationData)
	router.GET(baseURL+"/live", wrapper.Alive)
	router.GET(baseURL+"/ready", wrapper.Ready)
	router.GET(baseURL+"/user/data/:address", wrapper.UserData)
	router.POST(baseURL+"/wallet/connection", wrapper.ConnectWallet)

}

type GenerateCodeRequestObject struct {
	Params GenerateCodeParams
}

type GenerateCodeResponseObject interface {
	VisitGenerateCodeResponse(w http.ResponseWriter) error
}

type GenerateCode200JSONResponse AffiliateCode

func (response GenerateCode200JSONResponse) VisitGenerateCodeResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)

	return json.NewEncoder(w).Encode(response)
}

type GenerateCode400JSONResponse Error

func (response GenerateCode400JSONResponse) VisitGenerateCodeResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(400)

	return json.NewEncoder(w).Encode(response)
}

type GenerateCode401JSONResponse Error

func (response GenerateCode401JSONResponse) VisitGenerateCodeResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(401)

	return json.NewEncoder(w).Encode(response)
}

type GenerateCode5XXJSONResponse struct {
	Body       Error
	StatusCode int
}

func (response GenerateCode5XXJSONResponse) VisitGenerateCodeResponse(w http.ResponseWriter) error {
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

type AliveRequestObject struct {
}

type AliveResponseObject interface {
	VisitAliveResponse(w http.ResponseWriter) error
}

type Alive200Response struct {
}

func (response Alive200Response) VisitAliveResponse(w http.ResponseWriter) error {
	w.WriteHeader(200)
	return nil
}

type ReadyRequestObject struct {
}

type ReadyResponseObject interface {
	VisitReadyResponse(w http.ResponseWriter) error
}

type Ready200Response struct {
}

func (response Ready200Response) VisitReadyResponse(w http.ResponseWriter) error {
	w.WriteHeader(200)
	return nil
}

type Ready503Response struct {
}

func (response Ready503Response) VisitReadyResponse(w http.ResponseWriter) error {
	w.WriteHeader(503)
	return nil
}

type UserDataRequestObject struct {
	Address string `json:"address"`
}

type UserDataResponseObject interface {
	VisitUserDataResponse(w http.ResponseWriter) error
}

type UserData200JSONResponse UserDataResult

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

type UserData404Response struct {
}

func (response UserData404Response) VisitUserDataResponse(w http.ResponseWriter) error {
	w.WriteHeader(404)
	return nil
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

type ConnectWalletRequestObject struct {
	Body *ConnectWalletJSONRequestBody
}

type ConnectWalletResponseObject interface {
	VisitConnectWalletResponse(w http.ResponseWriter) error
}

type ConnectWallet200JSONResponse UserDataResult

func (response ConnectWallet200JSONResponse) VisitConnectWalletResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)

	return json.NewEncoder(w).Encode(response)
}

type ConnectWallet400JSONResponse Error

func (response ConnectWallet400JSONResponse) VisitConnectWalletResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(400)

	return json.NewEncoder(w).Encode(response)
}

type ConnectWallet401JSONResponse Error

func (response ConnectWallet401JSONResponse) VisitConnectWalletResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(401)

	return json.NewEncoder(w).Encode(response)
}

type ConnectWallet5XXJSONResponse struct {
	Body       Error
	StatusCode int
}

func (response ConnectWallet5XXJSONResponse) VisitConnectWalletResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(response.StatusCode)

	return json.NewEncoder(w).Encode(response.Body)
}

// StrictServerInterface represents all server handlers.
type StrictServerInterface interface {
	// generate new affiliate code
	// (POST /affiliate/code)
	GenerateCode(ctx context.Context, request GenerateCodeRequestObject) (GenerateCodeResponseObject, error)
	// get general donation data
	// (GET /donation/data)
	DonationData(ctx context.Context, request DonationDataRequestObject) (DonationDataResponseObject, error)
	// is the service alive?
	// (GET /live)
	Alive(ctx context.Context, request AliveRequestObject) (AliveResponseObject, error)
	// is the service alive and ready to do work?
	// (GET /ready)
	Ready(ctx context.Context, request ReadyRequestObject) (ReadyResponseObject, error)
	// get the donation data for the wallet address in question
	// (GET /user/data/{address})
	UserData(ctx context.Context, request UserDataRequestObject) (UserDataResponseObject, error)
	// associate a wallet address with an affiliate code
	// (POST /wallet/connection)
	ConnectWallet(ctx context.Context, request ConnectWalletRequestObject) (ConnectWalletResponseObject, error)
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

// GenerateCode operation middleware
func (sh *strictHandler) GenerateCode(ctx echo.Context, params GenerateCodeParams) error {
	var request GenerateCodeRequestObject

	request.Params = params

	handler := func(ctx echo.Context, request interface{}) (interface{}, error) {
		return sh.ssi.GenerateCode(ctx.Request().Context(), request.(GenerateCodeRequestObject))
	}
	for _, middleware := range sh.middlewares {
		handler = middleware(handler, "GenerateCode")
	}

	response, err := handler(ctx, request)

	if err != nil {
		return err
	} else if validResponse, ok := response.(GenerateCodeResponseObject); ok {
		return validResponse.VisitGenerateCodeResponse(ctx.Response())
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

// Alive operation middleware
func (sh *strictHandler) Alive(ctx echo.Context) error {
	var request AliveRequestObject

	handler := func(ctx echo.Context, request interface{}) (interface{}, error) {
		return sh.ssi.Alive(ctx.Request().Context(), request.(AliveRequestObject))
	}
	for _, middleware := range sh.middlewares {
		handler = middleware(handler, "Alive")
	}

	response, err := handler(ctx, request)

	if err != nil {
		return err
	} else if validResponse, ok := response.(AliveResponseObject); ok {
		return validResponse.VisitAliveResponse(ctx.Response())
	} else if response != nil {
		return fmt.Errorf("unexpected response type: %T", response)
	}
	return nil
}

// Ready operation middleware
func (sh *strictHandler) Ready(ctx echo.Context) error {
	var request ReadyRequestObject

	handler := func(ctx echo.Context, request interface{}) (interface{}, error) {
		return sh.ssi.Ready(ctx.Request().Context(), request.(ReadyRequestObject))
	}
	for _, middleware := range sh.middlewares {
		handler = middleware(handler, "Ready")
	}

	response, err := handler(ctx, request)

	if err != nil {
		return err
	} else if validResponse, ok := response.(ReadyResponseObject); ok {
		return validResponse.VisitReadyResponse(ctx.Response())
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

// ConnectWallet operation middleware
func (sh *strictHandler) ConnectWallet(ctx echo.Context) error {
	var request ConnectWalletRequestObject

	var body ConnectWalletJSONRequestBody
	if err := ctx.Bind(&body); err != nil {
		return err
	}
	request.Body = &body

	handler := func(ctx echo.Context, request interface{}) (interface{}, error) {
		return sh.ssi.ConnectWallet(ctx.Request().Context(), request.(ConnectWalletRequestObject))
	}
	for _, middleware := range sh.middlewares {
		handler = middleware(handler, "ConnectWallet")
	}

	response, err := handler(ctx, request)

	if err != nil {
		return err
	} else if validResponse, ok := response.(ConnectWalletResponseObject); ok {
		return validResponse.VisitConnectWalletResponse(ctx.Response())
	} else if response != nil {
		return fmt.Errorf("unexpected response type: %T", response)
	}
	return nil
}

// Base64 encoded, gzipped, json marshaled Swagger object
var swaggerSpec = []string{

	"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",
	"m/V7y8DbXG/+EwAA///IOifupSsAAA==",
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
