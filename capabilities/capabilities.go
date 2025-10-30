package capabilities

import (
	"errors"
	"strings"
)

type Capability struct {
	id   string
	name string
}

type CapabilitySet struct {
	set []*Capability
}

var (
	ErrUnknown = errors.New("unknown capability")

	CapabilityGhost = &Capability{"gh", "Ghost"}
)

func ParseCapability(str string) (*Capability, error) {
	switch strings.ToLower(str) {
	case "gh":
		return CapabilityGhost, nil
	}

	return nil, ErrUnknown
}

func (s *Capability) Is(other *Capability) bool {
	return s.id == other.id
}

func (s *Capability) String() string {
	return s.name
}

func (s *Capability) Stringp() *string {
	return &s.name
}

func ParseCapabilitySet(str string, ignoreUnknown bool) (*CapabilitySet, error) {
	result := &CapabilitySet{}
	tokens := strings.Split(str, ",")
	for _, t := range tokens {
		c, err := ParseCapability(t)
		if err != nil {
			if ignoreUnknown {
				continue
			} else {
				return nil, ErrUnknown
			}
		}
		result.Set(c)
	}

	return result, nil
}

func (s *CapabilitySet) Set(c *Capability) {
	if !s.Contains(c) {
		s.set = append(s.set, c)
	}
}

func (s *CapabilitySet) Contains(c *Capability) bool {
	for _, sc := range s.set {
		if sc.Is(c) {
			return true
		}
	}

	return false
}

func (s *CapabilitySet) String() string {
	if len(s.set) == 0 {
		return ""
	}

	result := s.set[0].String()
	for idx := 1; idx < len(s.set); idx++ {
		result += "," + s.set[idx].id
	}

	return result
}

func (s *CapabilitySet) Stringp() *string {
	str := s.String()
	return &str
}
