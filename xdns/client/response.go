package client

import (
	"net/netip"
	"time"
)

// Attention! Do not forget to update Clone() call if add more pointers!
type Response struct {
	Exists             bool
	Addresses          []netip.Addr
	Expires            time.Time
	ProtectionRequired bool
}

func (r *Response) Clone() *Response {
	clone := *r
	clone.Addresses = make([]netip.Addr, len(r.Addresses))
	copy(clone.Addresses, r.Addresses)

	return &clone
}

func (r *Response) Successful() bool {
	return r != nil && r.Exists && len(r.Addresses) > 0
}

func (r *Response) Expired() bool {
	if r.Expires.IsZero() {
		return false
	}

	return time.Now().After(r.Expires)
}

func (r *Response) AddressesAsStrings() []string {
	result := make([]string, len(r.Addresses))
	for idx, addr := range r.Addresses {
		result[idx] = addr.String()
	}

	return result
}
