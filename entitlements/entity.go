package entitlements

import (
	"encoding/json"

	discoveryAPI "github.com/vpnhouse/api/go/client/discovery"
)

type RestrictLocationEntry struct {
	discoveryAPI.Location
	Credentials []discoveryAPI.Node `json:"credentials"`
}
type Entitlements struct {
	Ads              bool                    `json:"ads,omitempty" yaml:"ads,omitempty"`
	RestrictLocation []RestrictLocationEntry `json:"restrict_location,omitempty" yaml:"restrict_location,omitempty"`
	Wireguard        bool                    `json:"wireguard,omitempty" yaml:"wireguard,omitempty"`
	IPRose           bool                    `json:"iprose,omitempty" yaml:"iprose,omitempty"`
	Proxy            bool                    `json:"proxy,omitempty" yaml:"proxy,omitempty"`
	ShapeUpstream    int                     `json:"shape_upstream,omitempty" yaml:"shape_upstream,omitempty"`
	ShapeDownstream  int                     `json:"shape_downstream,omitempty" yaml:"shape_downstream,omitempty"`
}

type ReducedEntitlements struct {
	Ads             bool     `json:"ads,omitempty" yaml:"ads,omitempty"`
	AllowedNodes    []string `json:"allow,omitempty" yaml:"allow,omitempty"`
	Wireguard       bool     `json:"wg,omitempty" yaml:"wg,omitempty"`
	IPRose          bool     `json:"ipr,omitempty" yaml:"ipr,omitempty"`
	Proxy           bool     `json:"prx,omitempty" yaml:"prx,omitempty"`
	ShapeUpstream   int      `json:"upst,omitempty" yaml:"upst,omitempty"`
	ShapeDownstream int      `json:"dnst,omitempty" yaml:"dnst,omitempty"`
}

type EntitlementsMapAny map[string]any

func (i *Entitlements) Reduce() *ReducedEntitlements {
	result := &ReducedEntitlements{
		Ads:             i.Ads,
		Wireguard:       i.Wireguard,
		IPRose:          i.IPRose,
		Proxy:           i.Proxy,
		ShapeUpstream:   i.ShapeDownstream,
		ShapeDownstream: i.ShapeDownstream,
	}

	return result
}

func FromJSONResuced(v []byte) (*ReducedEntitlements, error) {
	r := ReducedEntitlements{}
	err := json.Unmarshal(v, &r)
	if err == nil {
		return &r, nil
	}

	e := Entitlements{}
	err = json.Unmarshal(v, &e)
	if err == nil {
		return e.Reduce(), nil
	}

	return nil, err
}

func FromJSON(v []byte) (*Entitlements, error) {
	i := Entitlements{}
	err := json.Unmarshal(v, &i)
	if err != nil {
		return nil, err
	}
	return &i, nil
}

func FromMapAny(m EntitlementsMapAny) (*Entitlements, error) {
	intermediate, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}

	var result Entitlements
	err = json.Unmarshal(intermediate, &result)
	return &result, err
}

func (i *Entitlements) ToJSON() ([]byte, error) {
	return json.Marshal(i)
}

func (i *Entitlements) ToMapAny() (EntitlementsMapAny, error) {
	intermediate, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}

	var result EntitlementsMapAny
	err = json.Unmarshal(intermediate, &result)
	return result, err
}

func (i *Entitlements) HasWireguard() bool {
	return i.Wireguard
}

func (i *Entitlements) HasIPRose() bool {
	return i.IPRose
}

func (i *Entitlements) HasProxy() bool {
	return i.Proxy
}

func (i *Entitlements) HasAds() bool {
	return i.Ads
}

func (i *Entitlements) IsPaid() bool {
	return !i.HasAds()
}

func (i *Entitlements) IsFree() bool {
	return i.HasAds()
}

func (i *Entitlements) GetShapeUpstream() (int, bool) {
	return i.ShapeUpstream, i.ShapeUpstream > 0
}

func (i *Entitlements) GetShapeDownstream() (int, bool) {
	return i.ShapeDownstream, i.ShapeDownstream > 0
}
