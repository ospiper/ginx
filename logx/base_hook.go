package logx

import (
	"github.com/sirupsen/logrus"
)

type BaseHook struct{}

func (b *BaseHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (b *BaseHook) Fire(entry *logrus.Entry) error {
	entry.Data["timestamp"] = entry.Time.Unix()
	return nil
}
