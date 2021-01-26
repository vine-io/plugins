// Copyright 2021 lack
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package zap

import (
	"github.com/lack-io/vine/service/logger"
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type callerSkipKey struct{}

func WithCallerSkip(i int) logger.Option {
	return logger.SetOption(callerSkipKey{}, i)
}

type configKey struct{}

// WithConfig pass zap.Config to logger
func WithConfig(c zap.Config) logger.Option {
	return logger.SetOption(configKey{}, c)
}

type encoderConfigKey struct{}

// WithEncoderConfig pass zapcore.EncoderConfig to logger
func WithEncoderConfig(c zapcore.EncoderConfig) logger.Option {
	return logger.SetOption(encoderConfigKey{}, c)
}

type namespaceKey struct{}

func WithNamespace(namespace string) logger.Option {
	return logger.SetOption(namespaceKey{}, namespace)
}

type writerKey struct{}

func WithWriter(writer zapcore.WriteSyncer) logger.Option {
	return logger.SetOption(writerKey{}, writer)
}

type encoderJSONKey struct{}

func WithJSONEncode() logger.Option {
	return logger.SetOption(encoderJSONKey{}, struct{}{})
}

type FileWriter struct {
	FileName   string
	MaxSize    int
	MaxBackups int
	MaxAge     int
	Compress   bool
}

func WithFileWriter(fw FileWriter) logger.Option {
	return logger.SetOption(FileWriter{}, &lumberjack.Logger{
		Filename:   fw.FileName,
		MaxSize:    fw.MaxSize,
		MaxBackups: fw.MaxBackups,
		MaxAge:     fw.MaxAge,
		Compress:   fw.Compress,
	})
}
