package logx

import (
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type ContextLogHook struct{}

func (c *ContextLogHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (c *ContextLogHook) Fire(entry *logrus.Entry) error {
	if entry.Context == nil {
		return nil
	}
	ginCtx, ok := entry.Context.(*gin.Context)
	if !ok {
		return nil
	}
	entry.Data["client_ip"] = ginCtx.ClientIP()
	entry.Data["request_id"] = requestid.Get(ginCtx)
	return nil
}
