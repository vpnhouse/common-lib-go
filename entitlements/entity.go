package entitlements

import (
	"encoding/json"

	discoveryAPI "github.com/vpnhouse/api/go/client/discovery"
)

type RestrictLocationEntry struct {
	discoveryAPI.Location
	Credentials []discoveryAPI.Node `json:"credentials"`
}
type entitlements struct {
	Ads              bool                    `json:"ads" yaml:"ads"`
	RestrictLocation []RestrictLocationEntry `json:"restrict_location" yaml:"restrict_location"`
	Wireguard        bool                    `json:"wireguard" yaml:"wireguard"`
	IPRose           bool                    `json:"iprose" yaml:"iprose"`
	Proxy            bool                    `json:"proxy" yaml:"proxy"`
	ShapeUpstream    *int                    `json:"shape_upstream" yaml:"shape_upstream"`
	ShapeDownstream  *int                    `json:"shape_downstream" yaml:"shape_downstream"`
}

type Entitlements *entitlements
type EntitlementsMapAny map[string]any

func FromJSON(v []byte) (Entitlements, error) {
	i := entitlements{}
	err := json.Unmarshal(v, &i)
	if err != nil {
		return nil, err
	}
	return &i, nil
}

func FromMapAny(m EntitlementsMapAny) (Entitlements, error) {
	intermediate, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}

	var result entitlements
	err = json.Unmarshal(intermediate, &result)
	return &result, err
}

func (i *entitlements) ToJSON() ([]byte, error) {
	return json.Marshal(i)
}

func (i *entitlements) ToMapAny() (EntitlementsMapAny, error) {
	intermediate, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}

	var result EntitlementsMapAny
	err = json.Unmarshal(intermediate, &result)
	return result, err
}

func (i *entitlements) HasWireguard() bool {
	return i.Wireguard
}

func (i *entitlements) HasIPRose() bool {
	return i.IPRose
}

func (i entitlements) HasProxy() bool {
	return i.Proxy
}

func (i *entitlements) HasAds() bool {
	return i.Ads
}

func (i *entitlements) IsPaid() bool {
	return !i.HasAds()
}

func (i *entitlements) IsFree() bool {
	return i.HasAds()
}

func (i *entitlements) GetShapeUpstream() (int, bool) {
	if i.ShapeUpstream == nil {
		return 0, false
	}

	return *i.ShapeUpstream, true
}

func (i *entitlements) GetShapeDownstream() (int, bool) {
	if i.ShapeDownstream == nil {
		return 0, false
	}

	return *i.ShapeDownstream, true
}
