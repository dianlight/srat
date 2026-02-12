// go
package service

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/root"
	"github.com/stretchr/testify/suite"
)

type HaRootServiceSuite struct {
	suite.Suite
	ctx    context.Context
	cancel context.CancelFunc
	svc    *HaRootService
}

// Minimal fake response types matching fields used in the service
type fakeSimpleOKResp struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *root.SimpleOkResponse
}

type fakeSimpleResp[T any] struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *struct {
		Data   *T      `json:"data,omitempty"`
		Result *string `json:"result,omitempty"`
	}
}

func (r *fakeSimpleResp[T]) Status() string  { return r.HTTPResponse.Status }
func (r *fakeSimpleResp[T]) StatusCode() int { return r.HTTPResponse.StatusCode }

type fakeGetSystemResp fakeSimpleResp[root.SystemInfo]
type fakeAvailableUpdatesResp fakeSimpleResp[struct {
	AvailableUpdates *[]root.UpdateItem `json:"available_updates,omitempty"`
}]

// fakeClient implements the minimal methods used by HaRootService
type fakeClient struct {
	getSystemResp    *fakeGetSystemResp
	getSystemErr     error
	getAvailableResp *fakeAvailableUpdatesResp
	getAvailableErr  error
	refreshResp      *fakeSimpleOKResp
	refreshErr       error
	reloadResp       *fakeSimpleOKResp
	reloadErr        error
}

func (f *fakeClient) GetSystemInfoWithResponse(ctx context.Context, opts ...root.RequestEditorFn) (*root.GetSystemInfoResponse, error) {
	// return concrete type expected by caller (it will be type-asserted in our tests to fakeResp)
	if f.getSystemErr != nil {
		return nil, f.getSystemErr
	}
	return &root.GetSystemInfoResponse{HTTPResponse: f.getSystemResp.HTTPResponse, JSON200: f.getSystemResp.JSON200}, nil
}

func (f *fakeClient) GetAvailableUpdatesWithResponse(ctx context.Context, opts ...root.RequestEditorFn) (*root.GetAvailableUpdatesResponse, error) {
	if f.getAvailableErr != nil {
		return nil, f.getAvailableErr
	}
	return &root.GetAvailableUpdatesResponse{HTTPResponse: f.getAvailableResp.HTTPResponse, JSON200: f.getAvailableResp.JSON200}, nil
}

func (f *fakeClient) RefreshUpdatesWithResponse(ctx context.Context, opts ...root.RequestEditorFn) (*root.RefreshUpdatesResponse, error) {
	if f.refreshErr != nil {
		return nil, f.refreshErr
	}
	return &root.RefreshUpdatesResponse{
		HTTPResponse: f.refreshResp.HTTPResponse,
		JSON200:      f.refreshResp.JSON200,
	}, nil
}

func (f *fakeClient) ReloadUpdatesWithResponse(ctx context.Context, opts ...root.RequestEditorFn) (*root.ReloadUpdatesResponse, error) {
	if f.reloadErr != nil {
		return nil, f.reloadErr
	}
	return &root.ReloadUpdatesResponse{
		HTTPResponse: f.reloadResp.HTTPResponse,
		JSON200:      f.reloadResp.JSON200,
	}, nil
}

// To satisfy the real generated client interface, provide no-op implementations of any other methods.
// (If the real interface contains other methods, they are not called by HaRootService and can be omitted
// in this test file because we only assign fakeClient to the service via an interface value.)
// However, to avoid compile issues we provide type assertions in tests when calling service methods.

func TestHaRootServiceSuite(t *testing.T) {
	suite.Run(t, new(HaRootServiceSuite))
}

func (s *HaRootServiceSuite) SetupTest() {
	var wg sync.WaitGroup
	s.ctx, s.cancel = context.WithCancel(context.WithValue(context.Background(), "wg", &wg))
	// Ensure package cache is clean between tests
	haRootSystemInfoCache.Flush()
	// default service with nil client; individual tests will set svc.client as needed
	s.svc = &HaRootService{
		apiContext:       s.ctx,
		apiContextCancel: s.cancel,
		client:           nil,
		state: &dto.ContextState{
			HACoreReady: true,
		},
	}
}

func (s *HaRootServiceSuite) TearDownTest() {
	s.cancel()
	haRootSystemInfoCache.Flush()
}

// Helper to adapt fake responses into the concrete shapes used by the service methods.
// The real generated client returns concrete types; our HaRootService methods only access
// StatusCode() and JSON200 fields, so we create the expected concrete types here.

type concreteGetSystemResp struct {
	status  int
	JSON200 *struct{ Data *root.SystemInfo }
}

func (r *concreteGetSystemResp) StatusCode() int { return r.status }

type concreteGetAvailableResp struct {
	status  int
	JSON200 *struct {
		Data *struct {
			AvailableUpdates *[]root.UpdateItem
		}
	}
}

func (r *concreteGetAvailableResp) StatusCode() int { return r.status }

type concreteSimpleResp struct {
	status  int
	JSON200 *struct{}
}

func (r *concreteSimpleResp) StatusCode() int { return r.status }

// Test: if cache contains value, GetSystemInfo returns it without using client
func (s *HaRootServiceSuite) TestGetSystemInfo_CacheReturnedWhenPresent() {
	sys := &root.SystemInfo{Hostname: new("cached-host")}
	setCachedSystemInfo(sys)

	// Use nil client to ensure method does NOT require client when cache present
	s.svc.client = nil

	got, err := s.svc.GetSystemInfo()
	s.NoError(err)
	s.Equal(sys, got)
}

// Test: when cache empty and client nil, expect initialization error
func (s *HaRootServiceSuite) TestGetSystemInfo_ClientNilWhenNoCache() {
	haRootSystemInfoCache.Flush()
	s.svc.client = nil

	_, err := s.svc.GetSystemInfo()
	s.Error(err)
	s.Contains(err.Error(), "HA Root client is not initialized")
}

// Test: client returns error
func (s *HaRootServiceSuite) TestGetSystemInfo_ClientError() {
	haRootSystemInfoCache.Flush()
	fc := &fakeClient{
		getSystemErr: errors.New("network fail"),
	}
	// set concrete client that returns the fake client's error via adapter
	s.svc.client = fc

	_, err := s.svc.GetSystemInfo()
	s.Error(err)
	s.Contains(err.Error(), "network fail")
}

// Test: client returns non-200 or missing JSON -> error
func (s *HaRootServiceSuite) TestGetSystemInfo_Non200OrMissingJSON() {
	haRootSystemInfoCache.Flush()
	// non-200 status
	//concreteResp := &concreteGetSystemResp{status: 500, JSON200: nil}
	// wrap into a fake client that returns the concrete response
	fc := &fakeClient{}
	// assign by setting getSystemResp to a fakeResp and adapt in method via concrete type
	fc.getSystemResp = &fakeGetSystemResp{
		HTTPResponse: &http.Response{StatusCode: 500},
		JSON200:      nil,
	}
	s.svc.client = fc

	_, err := s.svc.GetSystemInfo()
	s.Error(err)
	s.Contains(err.Error(), "Error getting system info from ha_root")
	// also test JSON200 present but Data nil
	fc.getSystemResp = &fakeGetSystemResp{
		HTTPResponse: &http.Response{StatusCode: 200},
		JSON200: &struct {
			Data   *root.SystemInfo `json:"data,omitempty"`
			Result *string          `json:"result,omitempty"`
		}{Data: nil, Result: nil},
	}
	_, err2 := s.svc.GetSystemInfo()
	s.Error(err2)
	s.Contains(err2.Error(), "Error getting system info from ha_root")
}

// Test: successful fetch sets cache and subsequent reads come from cache
func (s *HaRootServiceSuite) TestGetSystemInfo_SuccessAndCacheSet() {
	haRootSystemInfoCache.Flush()
	expected := &root.SystemInfo{Hostname: new("real-host")}
	// compose a fake response with JSON200.Data set
	fc := &fakeClient{
		getSystemResp: &fakeGetSystemResp{
			HTTPResponse: &http.Response{StatusCode: 200},
			JSON200: &struct {
				Data   *root.SystemInfo `json:"data,omitempty"`
				Result *string          `json:"result,omitempty"`
			}{Data: expected, Result: new("success")},
		},
	}
	s.svc.client = fc

	got, err := s.svc.GetSystemInfo()
	s.NoError(err)
	s.Equal(expected, got)

	// Now remove client and ensure cache returns same value
	s.svc.client = nil
	got2, err2 := s.svc.GetSystemInfo()
	s.NoError(err2)
	s.Equal(expected, got2)
}

// Tests for GetAvailableUpdates
func (s *HaRootServiceSuite) TestGetAvailableUpdates_VariousCases() {
	// client nil -> error
	s.svc.client = nil
	_, err := s.svc.GetAvailableUpdates()
	s.Error(err)
	s.Contains(err.Error(), "HA Root client is not initialized")

	// client returns error
	fcErr := &fakeClient{getAvailableErr: errors.New("avail fail")}
	s.svc.client = fcErr
	_, err2 := s.svc.GetAvailableUpdates()
	s.Error(err2)
	s.Contains(err2.Error(), "avail fail")

	// client returns non-200
	fcNon200 := &fakeClient{
		getAvailableResp: &fakeAvailableUpdatesResp{
			HTTPResponse: &http.Response{StatusCode: 200},
			JSON200:      nil,
		},
	}
	s.svc.client = fcNon200
	_, err3 := s.svc.GetAvailableUpdates()
	s.Error(err3)
	s.Contains(err3.Error(), "Error getting available updates from ha_root")

	// success
	updates := []root.UpdateItem{{Name: new("u1")}}
	fcSuccess := &fakeClient{
		getAvailableResp: &fakeAvailableUpdatesResp{
			HTTPResponse: &http.Response{StatusCode: 200},
			JSON200: &struct {
				Data *struct {
					AvailableUpdates *[]root.UpdateItem `json:"available_updates,omitempty"`
				} `json:"data,omitempty"`
				Result *string `json:"result,omitempty"`
			}{
				Data: &struct {
					AvailableUpdates *[]root.UpdateItem `json:"available_updates,omitempty"`
				}{
					AvailableUpdates: &updates,
				},
				Result: new("success"),
			},
		},
	}
	s.svc.client = fcSuccess
	got, err4 := s.svc.GetAvailableUpdates()
	s.NoError(err4)
	s.Equal(updates, got)
}

// Tests for RefreshUpdates and ReloadUpdates
func (s *HaRootServiceSuite) TestRefreshAndReload_VariousCases() {
	// Refresh: client nil
	s.svc.client = nil
	err := s.svc.RefreshUpdates()
	s.Error(err)
	s.Contains(err.Error(), "HA Root client is not initialized")

	// client returns error for refresh
	fcRefErr := &fakeClient{refreshErr: errors.New("refresh fail")}
	s.svc.client = fcRefErr
	err = s.svc.RefreshUpdates()
	s.Error(err)
	s.Contains(err.Error(), "refresh fail")

	// client returns non-200 for refresh
	fcRefNon200 := &fakeClient{
		refreshResp: &fakeSimpleOKResp{
			HTTPResponse: &http.Response{StatusCode: 500},
			JSON200:      nil,
		},
	}
	s.svc.client = fcRefNon200
	err = s.svc.RefreshUpdates()
	s.Error(err)
	s.Contains(err.Error(), "Error refreshing updates from ha_root")

	// client returns 200 for refresh
	fcRefOK := &fakeClient{
		refreshResp: &fakeSimpleOKResp{
			HTTPResponse: &http.Response{StatusCode: 200},
			JSON200:      &root.SimpleOkResponse{},
		},
	}
	s.svc.client = fcRefOK
	err = s.svc.RefreshUpdates()
	s.NoError(err)

	// client returns error for reload
	fcRelErr := &fakeClient{reloadErr: errors.New("reload fail")}
	s.svc.client = fcRelErr
	err = s.svc.ReloadUpdates()
	s.Error(err)
	s.Contains(err.Error(), "reload fail")

	// client returns non-200 for reload
	fcRelNon200 := &fakeClient{
		reloadResp: &fakeSimpleOKResp{HTTPResponse: &http.Response{StatusCode: 500}, JSON200: nil},
	}
	s.svc.client = fcRelNon200
	err = s.svc.ReloadUpdates()
	s.Error(err)
	s.Contains(err.Error(), "Error reloading updates from ha_root")

	// client returns 200 for reload
	fcRelOK := &fakeClient{
		reloadResp: &fakeSimpleOKResp{HTTPResponse: &http.Response{StatusCode: 200}, JSON200: &root.SimpleOkResponse{}},
	}
	s.svc.client = fcRelOK
	err = s.svc.ReloadUpdates()
	s.NoError(err)
}
