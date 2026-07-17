package ginx

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func TestLogHookWithConfigSkipsAfterResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var logs bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&logs)
	logger.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true})

	router := gin.New()
	router.Use(LogHookWithConfig(logger, LogHookConfig{
		SkipPaths: []string{"/polling"},
		SkipFunc: func(c *gin.Context) bool {
			return c.Request.URL.Path == "/static" && c.Writer.Status() == http.StatusNoContent
		},
	}))
	router.GET("/polling", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.GET("/static", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	router.GET("/static-error", func(c *gin.Context) { c.Status(http.StatusNotFound) })
	router.GET("/api", func(c *gin.Context) { c.Status(http.StatusOK) })

	tests := []struct {
		path       string
		wantLogged bool
	}{
		{path: "/polling", wantLogged: false},
		{path: "/static", wantLogged: false},
		{path: "/static-error", wantLogged: true},
		{path: "/api", wantLogged: true},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			logs.Reset()
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if got := logs.Len() > 0; got != tt.wantLogged {
				t.Fatalf("request logged = %t, want %t; output:\n%s", got, tt.wantLogged, logs.String())
			}
		})
	}
}

func TestLogHookKeepsSkipPathsCompatibility(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var logs bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&logs)
	logger.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true})

	router := gin.New()
	router.Use(LogHook(logger, "/skip"))
	router.GET("/skip", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.GET("/keep", func(c *gin.Context) { c.Status(http.StatusOK) })

	for _, tt := range []struct {
		path       string
		wantLogged bool
	}{
		{path: "/skip", wantLogged: false},
		{path: "/keep", wantLogged: true},
	} {
		t.Run(tt.path, func(t *testing.T) {
			logs.Reset()
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if got := logs.Len() > 0; got != tt.wantLogged {
				t.Fatalf("request logged = %t, want %t; output:\n%s", got, tt.wantLogged, logs.String())
			}
		})
	}
}
