// Copyright 2021 The VPN House Authors. All rights reserved.
// Use of this source code is governed by a AGPL-style
// license that can be found in the LICENSE file.

package auth

import (
	"crypto/rsa"
	"errors"
	"fmt"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"github.com/vpnhouse/common-lib-go/xcrypto"
)

type JWTMaster struct {
	keyID   *uuid.UUID
	private *rsa.PrivateKey
	method  jwt.SigningMethod
}

func NewJWTMaster(private *rsa.PrivateKey, privateId *uuid.UUID) (*JWTMaster, error) {
	// Generate new private key if it's not given by caller
	if private == nil {
		if privateId != nil {
			return nil, errors.New("privateId must be nil when private is nil")
		}

		vPrivateId, err := uuid.NewRandom()
		if err != nil {
			return nil, err
		}

		privateId = &vPrivateId

		private, err = xcrypto.GenerateKey()
		if err != nil {
			return nil, fmt.Errorf("can't generate JWT key pair: %w", err)
		}
	} else {
		if privateId == nil {
			return nil, errors.New("privateId must be set when private is set")
		}
	}

	method := jwt.GetSigningMethod(jwtSigningMethod)
	if method == nil {
		return nil, fmt.Errorf("signing method is not supported: %v", jwtSigningMethod)
	}

	return &JWTMaster{
		private: private,
		keyID:   privateId,
		method:  method,
	}, nil
}

func (instance *JWTMaster) Token(claims jwt.Claims) (*string, error) {
	// Create token
	token := jwt.NewWithClaims(instance.method, claims)
	token.Header["kid"] = instance.keyID

	// Sign token
	signedToken, err := token.SignedString(instance.private)
	if err != nil {
		return nil, fmt.Errorf("can't sign token: %w", err)
	}

	return &signedToken, nil
}

func (instance *JWTMaster) Parse(tokenString string, claims jwt.Claims) error {
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return instance.private.Public(), nil
	})

	if err != nil || token == nil {
		return fmt.Errorf("invalid token: %w", err)
	}

	if !token.Valid {
		return errors.New("token is not valid")
	}

	method := token.Method.Alg()
	if method != instance.method.Alg() {
		return fmt.Errorf("invalid signine method: %v", method)
	}

	return nil
}
