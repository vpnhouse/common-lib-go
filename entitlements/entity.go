package entitlements

import (
	"encoding/json"
	"strconv"
)

type Entitlements map[string]any

const (
	Wireguard       = "wireguard"
	IPRose          = "iprose"
	Proxy           = "proxy"
	ShapeDownstream = "shape_downstream"
	ShapeUpstream   = "shape_upstream"
	Ads             = "ads"
	Restrictions    = "restrictions"
)

func ParseJSON(v []byte) (Entitlements, error) {
	s := Entitlements{}
	err := json.Unmarshal(v, &s)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s Entitlements) AsMap() map[string]any {
	return s
}

func (s Entitlements) AsMapPtr() *map[string]any {
	var m map[string]any = s
	return &m
}

func (s Entitlements) JSON() ([]byte, error) {
	return json.Marshal(s)
}

func (s Entitlements) SetWireguard(v bool) {
	s[Wireguard] = v
}

func (s Entitlements) HasWireguard() bool {
	v, _ := asBool(s[Wireguard])
	return v
}

func (s Entitlements) SetIPRose(v bool) {
	s[IPRose] = v
}

func (s Entitlements) HasIPRose() bool {
	v, _ := asBool(s[IPRose])
	return v
}

func (s Entitlements) SetProxy(v bool) {
	s[Proxy] = v
}

func (s Entitlements) HasProxy() bool {
	v, _ := asBool(s[Proxy])
	return v
}

func (s Entitlements) SetAds(v bool) {
	s[Ads] = v
}

func (s Entitlements) HasAds() bool {
	v, _ := asBool(s[Ads])
	return v
}

func (s Entitlements) IsPaid() bool {
	return !s.HasAds()
}

func (s Entitlements) IsFree() bool {
	return s.HasAds()
}

func (s Entitlements) SetShapeUpstream(v int) {
	s[ShapeUpstream] = v
}

func (s Entitlements) ShapeUpstream() (int, bool) {
	return asInt(s[ShapeUpstream])
}

func (s Entitlements) SetShapeDownstream(v int) {
	s[ShapeDownstream] = v
}

func (s Entitlements) ShapeDownstream() (int, bool) {
	return asInt(s[ShapeDownstream])
}

func (s Entitlements) Restrictions() (string, bool) {
	v, ok := s[Restrictions]
	if !ok {
		return "", false
	}

	switch restrictions := v.(type) {
	case string:
		return restrictions, true
	}

	return "", false
}

func (s Entitlements) SetRestrictions(v string) {
	s[Restrictions] = v
}

func asBool(value any) (bool, bool) {
	switch v := value.(type) {
	case bool:
		return v, true
	case string:
		if v == "true" {
			return true, true
		}
		if v == "false" {
			return false, true
		}
	}

	return false, false
}

func asInt(value any) (int, bool) {
	switch v := value.(type) {
	case int64:
		return int(v), true
	case int32:
		return int(v), true
	case uint64:
		return int(v), true
	case uint32:
		return int(v), true
	case uint:
		return int(v), true
	case int:
		return v, true
	case string:
		i, err := strconv.Atoi(v)
		if err != nil {
			return 0, false
		}
		return i, true
	}

	return 0, false
}
