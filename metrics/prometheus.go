package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

type MiddlewareBuilder struct {
	Namespace string
	Subsystem string
	Name      string
	Help      string
}

func (m MiddlewareBuilder) Build() gin.HandlerFunc {
	vector := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name:      m.Name,
		Subsystem: m.Subsystem,
		Namespace: m.Namespace,
		Help:      m.Help,
		Objectives: map[float64]float64{
			0.5:  0.01,
			0.75: 0.01,
			0.90: 0.01,
			0.99: 0.001,
			//0.999: 0.0001,
		},
	}, []string{"pattern", "method", "status"})

	prometheus.MustRegister(vector)
	return func(ctx *gin.Context) {
		startTime := time.Now()
		defer func() {
			duration := time.Now().Sub(startTime).Milliseconds()
			pattern := ctx.FullPath()
			if pattern == "" {
				pattern = "unknown"
			}
			vector.WithLabelValues(pattern, ctx.Request.Method,
				strconv.Itoa(ctx.Writer.Status())).Observe(float64(duration))
		}()
		ctx.Next()
	}
}
