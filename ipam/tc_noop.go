/*
 * // Copyright 2021 The VPNHouse Authors. All rights reserved.
 * // Use of this source code is governed by a AGPL-style
 * // license that can be found in the LICENSE file.
 */

package ipam

import (
	"github.com/vpnhouse/tunnel/pkg/xnet"
	"go.uber.org/zap"
)

type nopTC struct{}

func newNopTrafficControl() trafficControl {
	zap.L().Debug("using noop tc handle")
	return nopTC{}
}

func (nopTC) init() error {
	zap.L().Debug("init")
	return nil
}
func (nopTC) setLimit(forAddr xnet.IP, rate Rate) error {
	zap.L().Debug("set limit", zap.String("addr", forAddr.String()), zap.Stringer("rate", rate))
	return nil
}
func (nopTC) removeLimit(forAddr xnet.IP) error {
	zap.L().Debug("remove limit", zap.String("addr", forAddr.String()))
	return nil
}
func (nopTC) cleanup() error {
	return nil
}
