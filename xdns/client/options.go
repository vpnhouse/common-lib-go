package client

import (
	"time"

	"github.com/vpnhouse/common-lib-go/pkg/protect"
)

var Defaults = &options{
	systemTtl: systemDefaultTtl,

	cacheMaxSize:  cachedDefaultMaxSize,
	cacheMinTtl:   cachedDefaultMinTtl,
	cacheMaxTtl:   cachedDefaultMaxTtl,
	cacheKeepTime: cachedDefaultKeepTime,

	priorityFailureTimeout: 15 * time.Second,

	directTimeout:   directDefaultTimeout,
	directProtector: directDefaultProtector,
}

type options struct {
	systemTtl time.Duration

	cacheMaxSize  int
	cacheMinTtl   time.Duration
	cacheMaxTtl   time.Duration
	cacheKeepTime time.Duration

	priorityFailureTimeout time.Duration

	directTimeout   time.Duration
	directProtector protect.Protector
}

func (o *options) WithSystemTTL(value time.Duration) *options {
	o.systemTtl = value
	return o
}

func (o *options) WithCacheMaxSize(value int) *options {
	o.cacheMaxSize = value
	return o
}

func (o *options) WithCacheMinTTL(value time.Duration) *options {
	o.cacheMinTtl = value
	return o
}

func (o *options) WithCacheMaxTTL(value time.Duration) *options {
	o.cacheMaxTtl = value
	return o
}

func (o *options) WithCacheKeepTime(value time.Duration) *options {
	o.cacheKeepTime = value
	return o
}

func (o *options) WithPriorityFailureTimeout(value time.Duration) *options {
	o.priorityFailureTimeout = value
	return o
}

func (o *options) WithDirectTimeout(value time.Duration) *options {
	o.directTimeout = value
	return o
}

func (o *options) WithDirectProtector(value protect.Protector) *options {
	o.directProtector = value
	return o
}
