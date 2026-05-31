package rest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/ospiper/ginx/dbx"
)

type controllerTestModel struct {
	dbx.Model
}

func (m controllerTestModel) NewWithID(id int64) controllerTestModel {
	m.Model = dbx.Model{ID: id}
	return m
}

type controllerTestProvider struct {
	deleteCalls []int64
	customCalls []int64
	deleteErr   error
	customErr   error
}

func (p *controllerTestProvider) GetDB() *gorm.DB { return nil }

func (p *controllerTestProvider) Model(context.Context) *gorm.DB { return nil }

func (p *controllerTestProvider) Migrate() error { return nil }

func (p *controllerTestProvider) FindOne(context.Context, int64) (*controllerTestModel, error) {
	return &controllerTestModel{}, nil
}

func (p *controllerTestProvider) Find(context.Context, *FindConditions) ([]*controllerTestModel, error) {
	return nil, nil
}

func (p *controllerTestProvider) FindFirst(context.Context, *FindConditions) (*controllerTestModel, error) {
	return nil, nil
}

func (p *controllerTestProvider) FindAssoc(context.Context, any, string, *FindConditions) ([]*controllerTestModel, error) {
	return nil, nil
}

func (p *controllerTestProvider) Count(context.Context, []FilterFunc) (int64, error) {
	return 0, nil
}

func (p *controllerTestProvider) CountAssoc(context.Context, any, string, []FilterFunc) (int64, error) {
	return 0, nil
}

func (p *controllerTestProvider) Insert(context.Context, *controllerTestModel) error { return nil }

func (p *controllerTestProvider) InsertMany(context.Context, []*controllerTestModel) error {
	return nil
}

func (p *controllerTestProvider) InsertBatch(context.Context, []*controllerTestModel, int) error {
	return nil
}

func (p *controllerTestProvider) Update(context.Context, int64, *controllerTestModel) error {
	return nil
}

func (p *controllerTestProvider) UpdateFields(context.Context, int64, map[string]any) (*controllerTestModel, error) {
	return nil, nil
}

func (p *controllerTestProvider) Delete(_ context.Context, id int64) error {
	p.deleteCalls = append(p.deleteCalls, id)
	return p.deleteErr
}

func (p *controllerTestProvider) DeleteMany(context.Context, []int64) error { return nil }

func (p *controllerTestProvider) DeleteByID(_ context.Context, id int64) error {
	p.customCalls = append(p.customCalls, id)
	return p.customErr
}

type providerWithoutCustomDelete struct {
	deleteCalls []int64
	deleteErr   error
}

func (p *providerWithoutCustomDelete) GetDB() *gorm.DB { return nil }

func (p *providerWithoutCustomDelete) Model(context.Context) *gorm.DB { return nil }

func (p *providerWithoutCustomDelete) Migrate() error { return nil }

func (p *providerWithoutCustomDelete) FindOne(context.Context, int64) (*controllerTestModel, error) {
	return &controllerTestModel{}, nil
}

func (p *providerWithoutCustomDelete) Find(context.Context, *FindConditions) ([]*controllerTestModel, error) {
	return nil, nil
}

func (p *providerWithoutCustomDelete) FindFirst(context.Context, *FindConditions) (*controllerTestModel, error) {
	return nil, nil
}

func (p *providerWithoutCustomDelete) FindAssoc(context.Context, any, string, *FindConditions) ([]*controllerTestModel, error) {
	return nil, nil
}

func (p *providerWithoutCustomDelete) Count(context.Context, []FilterFunc) (int64, error) {
	return 0, nil
}

func (p *providerWithoutCustomDelete) CountAssoc(context.Context, any, string, []FilterFunc) (int64, error) {
	return 0, nil
}

func (p *providerWithoutCustomDelete) Insert(context.Context, *controllerTestModel) error { return nil }

func (p *providerWithoutCustomDelete) InsertMany(context.Context, []*controllerTestModel) error {
	return nil
}

func (p *providerWithoutCustomDelete) InsertBatch(context.Context, []*controllerTestModel, int) error {
	return nil
}

func (p *providerWithoutCustomDelete) Update(context.Context, int64, *controllerTestModel) error {
	return nil
}

func (p *providerWithoutCustomDelete) UpdateFields(context.Context, int64, map[string]any) (*controllerTestModel, error) {
	return nil, nil
}

func (p *providerWithoutCustomDelete) Delete(_ context.Context, id int64) error {
	p.deleteCalls = append(p.deleteCalls, id)
	return p.deleteErr
}

func (p *providerWithoutCustomDelete) DeleteMany(context.Context, []int64) error { return nil }

func TestRegisterResourceControllerDeleteFallsBackToProviderDelete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	group := r.Group("/items")

	provider := &providerWithoutCustomDelete{}
	RegisterResourceController[controllerTestModel](group, provider)

	req := httptest.NewRequest(http.MethodDelete, "/items/42", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if len(provider.deleteCalls) != 1 || provider.deleteCalls[0] != 42 {
		t.Fatalf("delete calls = %v, want [42]", provider.deleteCalls)
	}
}

func TestRegisterResourceControllerDeleteUsesCustomDeleterWhenAvailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	group := r.Group("/items")

	provider := &controllerTestProvider{}
	RegisterResourceController[controllerTestModel](group, provider)

	req := httptest.NewRequest(http.MethodDelete, "/items/7", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if len(provider.customCalls) != 1 || provider.customCalls[0] != 7 {
		t.Fatalf("custom delete calls = %v, want [7]", provider.customCalls)
	}
	if len(provider.deleteCalls) != 0 {
		t.Fatalf("default delete calls = %v, want none", provider.deleteCalls)
	}
}

func TestRegisterResourceControllerRegistersFullCRUDRoutesByDefault(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	group := r.Group("/items")

	provider := &controllerTestProvider{}
	RegisterResourceController[controllerTestModel](group, provider)

	assertRouteMethods(t, r.Routes(), "/items", []string{http.MethodGet, http.MethodPost})
	assertRouteMethods(t, r.Routes(), "/items/:id", []string{http.MethodDelete, http.MethodGet, http.MethodPut})
}

func TestRegisterResourceControllerReadOnlyOnlyRegistersGetRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	group := r.Group("/items")

	provider := &controllerTestProvider{}
	RegisterResourceController[controllerTestModel](group, provider, ReadOnly())

	assertRouteMethods(t, r.Routes(), "/items", []string{http.MethodGet})
	assertRouteMethods(t, r.Routes(), "/items/:id", []string{http.MethodGet})
}

func assertRouteMethods(t *testing.T, routes gin.RoutesInfo, path string, want []string) {
	t.Helper()

	got := make([]string, 0, len(want))
	for _, route := range routes {
		if route.Path == path {
			got = append(got, route.Method)
		}
	}
	sort.Strings(got)
	sort.Strings(want)
	if len(got) != len(want) {
		t.Fatalf("%s methods = %v, want %v", path, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s methods = %v, want %v", path, got, want)
		}
	}
}
