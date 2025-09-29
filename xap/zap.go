// Copyright 2021 The VPN House Authors. All rights reserved.
// Use of this source code is governed by a AGPL-style
// license that can be found in the LICENSE file.

package xap

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ZapType returns zap.String with a type name of v
func ZapType(v interface{}) zap.Field {
	return zap.String("type", fmt.Sprintf("%T", v))
}

func HumanReadableLogger(level string) *zap.Logger {
	var logLevel zap.AtomicLevel
	if err := logLevel.UnmarshalText([]byte(level)); err != nil {
		panic("failed to parse log level: + " + err.Error())
	}

	encoder := zap.NewDevelopmentEncoderConfig()
	encoder.EncodeLevel = zapcore.CapitalColorLevelEncoder

	loggerConfig := zap.Config{
		Development:       false,
		Level:             logLevel,
		OutputPaths:       []string{"stdout"},
		ErrorOutputPaths:  []string{"stderr"},
		Encoding:          "console",
		EncoderConfig:     encoder,
		DisableStacktrace: false,
	}

	z, err := loggerConfig.Build()
	if err != nil {
		panic(err)
	}

	return z
}

func JSONFormattedLogger(lvl zap.AtomicLevel) *zap.Logger {
	loggerConfig := zap.NewProductionConfig()
	loggerConfig.Level = lvl
	z, err := loggerConfig.Build()
	if err != nil {
		panic(err)
	}

	return z
}
