package zap

import (
	"os"
	"testing"

	"github.com/lack-io/vine/service/logger"
	"go.uber.org/zap/zapcore"
)

func TestName(t *testing.T) {
	l, err := New()
	if err != nil {
		t.Fatal(err)
	}

	if l.String() != "zap" {
		t.Errorf("name is error %s", l.String())
	}

	t.Logf("test logger name: %s", l.String())
}

func TestLogf(t *testing.T) {
	l, err := New()
	if err != nil {
		t.Fatal(err)
	}

	logger.DefaultLogger = l
	logger.Logf(logger.InfoLevel, "test logf: %s", "name")
}

func TestSetLevel(t *testing.T) {
	l, err := New(WithJSONEncode())
	if err != nil {
		t.Fatal(err)
	}
	logger.DefaultLogger = l

	logger.Init(logger.WithLevel(logger.DebugLevel))
	l.Logf(logger.DebugLevel, "test show debug: %s", "debug msg")

	logger.Init(logger.WithLevel(logger.InfoLevel))
	l.Logf(logger.DebugLevel, "test non-show debug: %s", "debug msg")
}

func TestWithFileWriter(t *testing.T) {
	l, err := New(WithFileWriter(FileWriter{
		FileName:   "test.log",
		MaxSize:    1,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   false,
	}), WithWriter(zapcore.AddSync(os.Stdout)))
	if err != nil {
		t.Fatal(err)
	}
	defer l.Sync()
	logger.DefaultLogger = l

	l.Logf(logger.InfoLevel, "test")
}
