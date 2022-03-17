// Copyright 2021 The VPN House Authors. All rights reserved.
// Use of this source code is governed by a AGPL-style
// license that can be found in the LICENSE file.

package xhttp

import (
	"net/http"
	"strings"

	"go.uber.org/zap"
)

func ExtractTokenFromRequest(r *http.Request) (string, bool) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		zap.L().Debug("no auth header was found")
		return "", false // No error, just no token
	}

	authHeaderParts := strings.Fields(authHeader)
	if len(authHeaderParts) != 2 || strings.ToLower(authHeaderParts[0]) != "bearer" {
		zap.L().Debug("bearer auth header was not found")
		return "", false
	}

	return authHeaderParts[1], true
}
