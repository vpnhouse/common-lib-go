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
	Ads              bool                    `json:"ads" yaml:"ads"`
	RestrictLocation []RestrictLocationEntry `json:"restrict_location" yaml:"restrict_location"`
	Wireguard        bool                    `json:"wireguard" yaml:"wireguard"`
	IPRose           bool                    `json:"iprose" yaml:"iprose"`
	Proxy            bool                    `json:"proxy" yaml:"proxy"`
	ShapeUpstream    *int                    `json:"shape_upstream" yaml:"shape_upstream"`
	ShapeDownstream  *int                    `json:"shape_downstream" yaml:"shape_downstream"`
}

type EntitlementsMapAny map[string]any

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
	if i.ShapeUpstream == nil {
		return 0, false
	}

	return *i.ShapeUpstream, true
}

func (i *Entitlements) GetShapeDownstream() (int, bool) {
	if i.ShapeDownstream == nil {
		return 0, false
	}

	return *i.ShapeDownstream, true
}
