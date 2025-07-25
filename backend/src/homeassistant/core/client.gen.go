// Package core provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/oapi-codegen/oapi-codegen/v2 version v2.5.0 DO NOT EDIT.
package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	BearerAuthScopes = "bearerAuth.Scopes"
)

// CoreCheck defines model for CoreCheck.
type CoreCheck struct {
	// Errors Lista di errori.
	Errors *[]string `json:"errors,omitempty"`

	// Result Risultato del controllo della configurazione.
	Result *string `json:"result,omitempty"`
}

// CoreInfo defines model for CoreInfo.
type CoreInfo struct {
	// Arch L'architettura dell'host (armhf, aarch64, i386, amd64).
	Arch *string `json:"arch,omitempty"`

	// AudioInput La descrizione del dispositivo di input audio.
	AudioInput *string `json:"audio_input,omitempty"`

	// AudioOutput La descrizione del dispositivo di output audio.
	AudioOutput *string `json:"audio_output,omitempty"`

	// BackupsExcludeDatabase I backup escludono il file del database di Home Assistant per impostazione predefinita.
	BackupsExcludeDatabase *bool `json:"backups_exclude_database,omitempty"`

	// Boot True se deve avviarsi all'avvio.
	Boot *bool `json:"boot,omitempty"`

	// Image L'immagine del container che esegue il Core.
	Image *string `json:"image,omitempty"`

	// IpAddress L'indirizzo IP Docker interno del Supervisor.
	IpAddress *string `json:"ip_address,omitempty"`

	// Machine Il tipo di macchina che esegue l'host.
	Machine *string `json:"machine,omitempty"`

	// Port La porta su cui è in esecuzione Home Assistant.
	Port *int `json:"port,omitempty"`

	// Ssl True se Home Assistant utilizza SSL.
	Ssl *bool `json:"ssl,omitempty"`

	// UpdateAvailable True se è disponibile un aggiornamento.
	UpdateAvailable *bool `json:"update_available,omitempty"`

	// Version La versione installata del Core.
	Version *string `json:"version,omitempty"`

	// VersionLatest L'ultima versione pubblicata nel canale attivo.
	VersionLatest *string `json:"version_latest,omitempty"`

	// WaitBoot Tempo massimo di attesa durante l'avvio.
	WaitBoot *int `json:"wait_boot,omitempty"`

	// Watchdog True se il watchdog è abilitato.
	Watchdog *bool `json:"watchdog,omitempty"`
}

// CoreUpdate defines model for CoreUpdate.
type CoreUpdate struct {
	// Version La versione a cui aggiornare.
	Version *string `json:"version,omitempty"`
}

// UpdateCoreJSONRequestBody defines body for UpdateCore for application/json ContentType.
type UpdateCoreJSONRequestBody = CoreUpdate

// RequestEditorFn  is the function signature for the RequestEditor callback function
type RequestEditorFn func(ctx context.Context, req *http.Request) error

// Doer performs HTTP requests.
//
// The standard http.Client implements this interface.
type HttpRequestDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client which conforms to the OpenAPI3 specification for this service.
type Client struct {
	// The endpoint of the server conforming to this interface, with scheme,
	// https://api.deepmap.com for example. This can contain a path relative
	// to the server, such as https://api.deepmap.com/dev-test, and all the
	// paths in the swagger spec will be appended to the server.
	Server string

	// Doer for performing requests, typically a *http.Client with any
	// customized settings, such as certificate chains.
	Client HttpRequestDoer

	// A list of callbacks for modifying requests which are generated before sending over
	// the network.
	RequestEditors []RequestEditorFn
}

// ClientOption allows setting custom parameters during construction
type ClientOption func(*Client) error

// Creates a new Client, with reasonable defaults
func NewClient(server string, opts ...ClientOption) (*Client, error) {
	// create a client with sane default values
	client := Client{
		Server: server,
	}
	// mutate client and add all optional params
	for _, o := range opts {
		if err := o(&client); err != nil {
			return nil, err
		}
	}
	// ensure the server URL always has a trailing slash
	if !strings.HasSuffix(client.Server, "/") {
		client.Server += "/"
	}
	// create httpClient, if not already present
	if client.Client == nil {
		client.Client = &http.Client{}
	}
	return &client, nil
}

// WithHTTPClient allows overriding the default Doer, which is
// automatically created using http.Client. This is useful for tests.
func WithHTTPClient(doer HttpRequestDoer) ClientOption {
	return func(c *Client) error {
		c.Client = doer
		return nil
	}
}

// WithRequestEditorFn allows setting up a callback function, which will be
// called right before sending the request. This can be used to mutate the request.
func WithRequestEditorFn(fn RequestEditorFn) ClientOption {
	return func(c *Client) error {
		c.RequestEditors = append(c.RequestEditors, fn)
		return nil
	}
}

// The interface specification for the client above.
type ClientInterface interface {
	// CheckCoreConfig request
	CheckCoreConfig(ctx context.Context, reqEditors ...RequestEditorFn) (*http.Response, error)

	// GetCoreInfo request
	GetCoreInfo(ctx context.Context, reqEditors ...RequestEditorFn) (*http.Response, error)

	// GetCoreLogs request
	GetCoreLogs(ctx context.Context, reqEditors ...RequestEditorFn) (*http.Response, error)

	// RebootCore request
	RebootCore(ctx context.Context, reqEditors ...RequestEditorFn) (*http.Response, error)

	// RepairCore request
	RepairCore(ctx context.Context, reqEditors ...RequestEditorFn) (*http.Response, error)

	// RestartCore request
	RestartCore(ctx context.Context, reqEditors ...RequestEditorFn) (*http.Response, error)

	// UpdateCoreWithBody request with any body
	UpdateCoreWithBody(ctx context.Context, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*http.Response, error)

	UpdateCore(ctx context.Context, body UpdateCoreJSONRequestBody, reqEditors ...RequestEditorFn) (*http.Response, error)
}

func (c *Client) CheckCoreConfig(ctx context.Context, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewCheckCoreConfigRequest(c.Server)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) GetCoreInfo(ctx context.Context, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewGetCoreInfoRequest(c.Server)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) GetCoreLogs(ctx context.Context, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewGetCoreLogsRequest(c.Server)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) RebootCore(ctx context.Context, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewRebootCoreRequest(c.Server)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) RepairCore(ctx context.Context, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewRepairCoreRequest(c.Server)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) RestartCore(ctx context.Context, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewRestartCoreRequest(c.Server)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) UpdateCoreWithBody(ctx context.Context, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewUpdateCoreRequestWithBody(c.Server, contentType, body)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) UpdateCore(ctx context.Context, body UpdateCoreJSONRequestBody, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewUpdateCoreRequest(c.Server, body)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

// NewCheckCoreConfigRequest generates requests for CheckCoreConfig
func NewCheckCoreConfigRequest(server string) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/core/check")
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// NewGetCoreInfoRequest generates requests for GetCoreInfo
func NewGetCoreInfoRequest(server string) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/core/info")
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// NewGetCoreLogsRequest generates requests for GetCoreLogs
func NewGetCoreLogsRequest(server string) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/core/logs")
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// NewRebootCoreRequest generates requests for RebootCore
func NewRebootCoreRequest(server string) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/core/reboot")
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// NewRepairCoreRequest generates requests for RepairCore
func NewRepairCoreRequest(server string) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/core/repair")
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// NewRestartCoreRequest generates requests for RestartCore
func NewRestartCoreRequest(server string) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/core/restart")
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// NewUpdateCoreRequest calls the generic UpdateCore builder with application/json body
func NewUpdateCoreRequest(server string, body UpdateCoreJSONRequestBody) (*http.Request, error) {
	var bodyReader io.Reader
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	bodyReader = bytes.NewReader(buf)
	return NewUpdateCoreRequestWithBody(server, "application/json", bodyReader)
}

// NewUpdateCoreRequestWithBody generates requests for UpdateCore with any type of body
func NewUpdateCoreRequestWithBody(server string, contentType string, body io.Reader) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/core/update")
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", queryURL.String(), body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", contentType)

	return req, nil
}

func (c *Client) applyEditors(ctx context.Context, req *http.Request, additionalEditors []RequestEditorFn) error {
	for _, r := range c.RequestEditors {
		if err := r(ctx, req); err != nil {
			return err
		}
	}
	for _, r := range additionalEditors {
		if err := r(ctx, req); err != nil {
			return err
		}
	}
	return nil
}

// ClientWithResponses builds on ClientInterface to offer response payloads
type ClientWithResponses struct {
	ClientInterface
}

// NewClientWithResponses creates a new ClientWithResponses, which wraps
// Client with return type handling
func NewClientWithResponses(server string, opts ...ClientOption) (*ClientWithResponses, error) {
	client, err := NewClient(server, opts...)
	if err != nil {
		return nil, err
	}
	return &ClientWithResponses{client}, nil
}

// WithBaseURL overrides the baseURL.
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) error {
		newBaseURL, err := url.Parse(baseURL)
		if err != nil {
			return err
		}
		c.Server = newBaseURL.String()
		return nil
	}
}

// ClientWithResponsesInterface is the interface specification for the client with responses above.
type ClientWithResponsesInterface interface {
	// CheckCoreConfigWithResponse request
	CheckCoreConfigWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*CheckCoreConfigResponse, error)

	// GetCoreInfoWithResponse request
	GetCoreInfoWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*GetCoreInfoResponse, error)

	// GetCoreLogsWithResponse request
	GetCoreLogsWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*GetCoreLogsResponse, error)

	// RebootCoreWithResponse request
	RebootCoreWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*RebootCoreResponse, error)

	// RepairCoreWithResponse request
	RepairCoreWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*RepairCoreResponse, error)

	// RestartCoreWithResponse request
	RestartCoreWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*RestartCoreResponse, error)

	// UpdateCoreWithBodyWithResponse request with any body
	UpdateCoreWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*UpdateCoreResponse, error)

	UpdateCoreWithResponse(ctx context.Context, body UpdateCoreJSONRequestBody, reqEditors ...RequestEditorFn) (*UpdateCoreResponse, error)
}

type CheckCoreConfigResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *struct {
		Data   *CoreCheck                `json:"data,omitempty"`
		Result *CheckCoreConfig200Result `json:"result,omitempty"`
	}
}
type CheckCoreConfig200Result string

// Status returns HTTPResponse.Status
func (r CheckCoreConfigResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r CheckCoreConfigResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type GetCoreInfoResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *struct {
		Data   *CoreInfo             `json:"data,omitempty"`
		Result *GetCoreInfo200Result `json:"result,omitempty"`
	}
}
type GetCoreInfo200Result string

// Status returns HTTPResponse.Status
func (r GetCoreInfoResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r GetCoreInfoResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type GetCoreLogsResponse struct {
	Body         []byte
	HTTPResponse *http.Response
}

// Status returns HTTPResponse.Status
func (r GetCoreLogsResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r GetCoreLogsResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type RebootCoreResponse struct {
	Body         []byte
	HTTPResponse *http.Response
}

// Status returns HTTPResponse.Status
func (r RebootCoreResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r RebootCoreResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type RepairCoreResponse struct {
	Body         []byte
	HTTPResponse *http.Response
}

// Status returns HTTPResponse.Status
func (r RepairCoreResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r RepairCoreResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type RestartCoreResponse struct {
	Body         []byte
	HTTPResponse *http.Response
}

// Status returns HTTPResponse.Status
func (r RestartCoreResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r RestartCoreResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type UpdateCoreResponse struct {
	Body         []byte
	HTTPResponse *http.Response
}

// Status returns HTTPResponse.Status
func (r UpdateCoreResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r UpdateCoreResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

// CheckCoreConfigWithResponse request returning *CheckCoreConfigResponse
func (c *ClientWithResponses) CheckCoreConfigWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*CheckCoreConfigResponse, error) {
	rsp, err := c.CheckCoreConfig(ctx, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseCheckCoreConfigResponse(rsp)
}

// GetCoreInfoWithResponse request returning *GetCoreInfoResponse
func (c *ClientWithResponses) GetCoreInfoWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*GetCoreInfoResponse, error) {
	rsp, err := c.GetCoreInfo(ctx, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseGetCoreInfoResponse(rsp)
}

// GetCoreLogsWithResponse request returning *GetCoreLogsResponse
func (c *ClientWithResponses) GetCoreLogsWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*GetCoreLogsResponse, error) {
	rsp, err := c.GetCoreLogs(ctx, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseGetCoreLogsResponse(rsp)
}

// RebootCoreWithResponse request returning *RebootCoreResponse
func (c *ClientWithResponses) RebootCoreWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*RebootCoreResponse, error) {
	rsp, err := c.RebootCore(ctx, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseRebootCoreResponse(rsp)
}

// RepairCoreWithResponse request returning *RepairCoreResponse
func (c *ClientWithResponses) RepairCoreWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*RepairCoreResponse, error) {
	rsp, err := c.RepairCore(ctx, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseRepairCoreResponse(rsp)
}

// RestartCoreWithResponse request returning *RestartCoreResponse
func (c *ClientWithResponses) RestartCoreWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*RestartCoreResponse, error) {
	rsp, err := c.RestartCore(ctx, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseRestartCoreResponse(rsp)
}

// UpdateCoreWithBodyWithResponse request with arbitrary body returning *UpdateCoreResponse
func (c *ClientWithResponses) UpdateCoreWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*UpdateCoreResponse, error) {
	rsp, err := c.UpdateCoreWithBody(ctx, contentType, body, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseUpdateCoreResponse(rsp)
}

func (c *ClientWithResponses) UpdateCoreWithResponse(ctx context.Context, body UpdateCoreJSONRequestBody, reqEditors ...RequestEditorFn) (*UpdateCoreResponse, error) {
	rsp, err := c.UpdateCore(ctx, body, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseUpdateCoreResponse(rsp)
}

// ParseCheckCoreConfigResponse parses an HTTP response from a CheckCoreConfigWithResponse call
func ParseCheckCoreConfigResponse(rsp *http.Response) (*CheckCoreConfigResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &CheckCoreConfigResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 200:
		var dest struct {
			Data   *CoreCheck                `json:"data,omitempty"`
			Result *CheckCoreConfig200Result `json:"result,omitempty"`
		}
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest

	}

	return response, nil
}

// ParseGetCoreInfoResponse parses an HTTP response from a GetCoreInfoWithResponse call
func ParseGetCoreInfoResponse(rsp *http.Response) (*GetCoreInfoResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &GetCoreInfoResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 200:
		var dest struct {
			Data   *CoreInfo             `json:"data,omitempty"`
			Result *GetCoreInfo200Result `json:"result,omitempty"`
		}
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest

	}

	return response, nil
}

// ParseGetCoreLogsResponse parses an HTTP response from a GetCoreLogsWithResponse call
func ParseGetCoreLogsResponse(rsp *http.Response) (*GetCoreLogsResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &GetCoreLogsResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	return response, nil
}

// ParseRebootCoreResponse parses an HTTP response from a RebootCoreWithResponse call
func ParseRebootCoreResponse(rsp *http.Response) (*RebootCoreResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &RebootCoreResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	return response, nil
}

// ParseRepairCoreResponse parses an HTTP response from a RepairCoreWithResponse call
func ParseRepairCoreResponse(rsp *http.Response) (*RepairCoreResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &RepairCoreResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	return response, nil
}

// ParseRestartCoreResponse parses an HTTP response from a RestartCoreWithResponse call
func ParseRestartCoreResponse(rsp *http.Response) (*RestartCoreResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &RestartCoreResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	return response, nil
}

// ParseUpdateCoreResponse parses an HTTP response from a UpdateCoreWithResponse call
func ParseUpdateCoreResponse(rsp *http.Response) (*UpdateCoreResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &UpdateCoreResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	return response, nil
}
