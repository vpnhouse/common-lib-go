package ipam

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

const (
	// Block ports from specified port range, allowing all others
	RestrictionModeBlockList = iota
	// Allow ports from specified port range, blocking all owhers
	RestrictionModeAllowList
)

var (
	protocolUDP protocolID = protocolID{name: "UDP", id: unix.IPPROTO_UDP}
	protocolTCP protocolID = protocolID{name: "TCP", id: unix.IPPROTO_TCP}
)

type ListMode struct {
	v int
}

type PortRange struct {
	low  uint16
	high uint16
}

type ProtocolPortConfig struct {
	Mode  ListMode    `yaml:"mode"`
	Ports []PortRange `yaml:"range,omitempty"`
}

type PortRestrictionConfig struct {
	UDP ProtocolPortConfig `yaml:"udp,omitempty"`
	TCP ProtocolPortConfig `yaml:"tcp,omitempty"`
}

type protocolID struct {
	name string
	id   uint8
}

func DefaultPortRestrictions() *PortRestrictionConfig {
	cfg := PortRestrictionConfig{
		UDP: ProtocolPortConfig{
			Mode: ListMode{
				v: RestrictionModeBlockList,
			},
			Ports: []PortRange{
				port(69),
				port(113),
				port(135),
				portRange(137, 139),
				port(445),
				port(514),
			},
		},
		TCP: ProtocolPortConfig{
			Mode: ListMode{
				v: RestrictionModeBlockList,
			},
			Ports: []PortRange{
				port(113),
				port(445),
			},
		},
	}

	return &cfg
}

func (mode ListMode) Int() int        { return mode.v }
func (mode ListMode) BlockList() bool { return mode.v == RestrictionModeBlockList }
func (mode ListMode) AllowList() bool { return mode.v == RestrictionModeAllowList }

func (mode *ListMode) UnmarshalText(raw []byte) error {
	s := string(raw)
	switch s {
	case "allow_list":
		mode.v = RestrictionModeAllowList
	case "block_list":
		mode.v = RestrictionModeBlockList
	default:
		return fmt.Errorf("unknown mode %s", s)
	}

	return nil
}

func (mode ListMode) String() string {
	switch mode.v {
	case RestrictionModeAllowList:
		return "allow_list"
	case RestrictionModeBlockList:
		return "block_list"
	default:
		return "unknown"
	}
}

func (mode ListMode) MarshalText() ([]byte, error) {
	return []byte(mode.String()), nil
}

func (rng *PortRange) UnmarshalText(raw []byte) error {
	s := string(raw)
	tuple_s := strings.Split(s, "-")
	if len(tuple_s) > 2 || len(tuple_s) < 1 {
		return fmt.Errorf("invalid range %s", s)
	}

	tuple := make([]uint16, len(tuple_s))
	for idx, p_s := range tuple_s {
		p, err := strconv.Atoi(strings.TrimSpace(p_s))
		if err != nil {
			return fmt.Errorf("invalid range %s", s)
		}
		if p < 0 || p > math.MaxUint16 {
			return fmt.Errorf("invalid range %s", s)
		}
		tuple[idx] = uint16(p)
	}

	rng.low = tuple[0]
	if len(tuple) == 1 {
		rng.high = tuple[0]
	} else {
		rng.high = tuple[1]
	}
	return nil
}

func (rng PortRange) MarshalText() ([]byte, error) {
	if rng.low == rng.high {
		return []byte(fmt.Sprintf("%d", rng.low)), nil
	} else {
		return []byte(fmt.Sprintf("%d-%d", rng.low, rng.high)), nil
	}
}

func portRange(low, high uint16) PortRange {
	return PortRange{
		low:  low,
		high: high,
	}
}

func port(port uint16) PortRange {
	return PortRange{
		low:  port,
		high: port,
	}
}
