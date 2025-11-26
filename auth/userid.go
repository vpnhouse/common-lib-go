// Copyright 2021 The VPN House Authors. All rights reserved.
// Use of this source code is governed by a AGPL-style
// license that can be found in the LICENSE file.

package auth

import (
	"errors"
	"regexp"
)

var (
	userIDRegexp = regexp.MustCompile("^([^/]*)/([^/]*)/(.*)$")
	nParts       = 3
)

func ParseUserID(v string) (project, auth, userID string, err error) {
	matches := userIDRegexp.FindStringSubmatch(v)
	if len(matches) != nParts+1 {
		err = errors.New("invalid user id format")
		return
	}

	project = matches[1]
	auth = matches[2]
	userID = matches[3]
	return
}
