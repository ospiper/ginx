package ginx

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// 2016-09-27 09:38:21.541541811 +0200 CEST
// 127.0.0.1 - frank [10/Oct/2000:13:55:36 -0700]
// "GET /apache_pb.gif HTTP/1.0" 200 2326
// "http://www.example.com/start.html"
// "Mozilla/4.08 [en] (Win98; I ;Nav)"

var timeFormat = "2006-01-02 15:04:05.999999 -0700"

// LogHookConfig configures request logging exclusions.
type LogHookConfig struct {
	SkipPaths []string
	SkipFunc  gin.Skipper
}

// LogHook creates request logging middleware with exact paths excluded.
func LogHook(logger *logrus.Logger, notLogged ...string) gin.HandlerFunc {
	return LogHookWithConfig(logger, LogHookConfig{SkipPaths: notLogged})
}

// LogHookWithConfig creates request logging middleware with configurable
// exclusions. SkipFunc runs after the remaining handlers have completed.
func LogHookWithConfig(logger *logrus.Logger, config LogHookConfig) gin.HandlerFunc {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	var skip map[string]struct{}

	if length := len(config.SkipPaths); length > 0 {
		skip = make(map[string]struct{}, length)

		for _, p := range config.SkipPaths {
			skip[p] = struct{}{}
		}
	}

	return func(c *gin.Context) {
		log := logger.WithContext(c)
		path := c.Request.URL.Path
		start := time.Now()
		c.Next()
		if _, ok := skip[path]; ok || (config.SkipFunc != nil && config.SkipFunc(c)) {
			return
		}

		stop := time.Since(start)
		latency := float64(stop.Microseconds()) / 1000
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		clientUserAgent := c.Request.UserAgent()
		referer := c.Request.Referer()
		dataLength := c.Writer.Size()
		if dataLength < 0 {
			dataLength = 0
		}

		entry := log.WithFields(logrus.Fields{
			"start_at":    start.Unix(),
			"hostname":    hostname,
			"status_code": statusCode,
			"elapsed":     latency, // ms
			"client_ip":   clientIP,
			"method":      c.Request.Method,
			"path":        path,
			"referer":     referer,
			"data_length": dataLength, // bytes
			"user_agent":  clientUserAgent,
			"request_id":  requestid.Get(c),
		})

		if len(c.Errors) > 0 {
			entry.Error(c.Errors.ByType(gin.ErrorTypePrivate).String())
		} else {
			msg := fmt.Sprintf("%s - %s [%s] \"%s %s\" %d %d \"%s\" \"%s\" (%.3fms)", clientIP, hostname, time.Now().Format(timeFormat), c.Request.Method, path, statusCode, dataLength, referer, clientUserAgent, latency)
			if statusCode >= http.StatusInternalServerError {
				entry.Error(msg)
			} else if statusCode >= http.StatusBadRequest {
				entry.Warn(msg)
			} else {
				entry.Info(msg)
			}
		}
	}
}
