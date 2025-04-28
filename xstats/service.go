package xstats

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/vpnhouse/common-lib-go/xcache"
)

const maxBytes = 32 << 20 // 32 Mb

type Service struct {
	sessions *xcache.Cache // session_id -> rx, tx, timestamp, data {installation_id, user_id, country etc.}
	onFlush  OnFlush
}

// Use shorten'd json names for more compact in memory placments
type SessionData struct {
	InstallationID string `json:"i_id,omitempty"`
	UserID         string `json:"u_id,omitempty"`
	Country        string `json:"c,omitempty"`
}

type Session struct {
	SessionID string // uuid
	SessionData
}

type Report struct {
	Session
	CreatedNano uint64
	DeltaRx     uint64
	DeltaTx     uint64
	DeltaTNano  uint64
}

type (
	OnFlush func(report *Report)
	OnData  func(sessionID uuid.UUID, out *SessionData)
)

var NowNano func() uint64 = func() uint64 {
	return uint64(time.Now().UnixNano())
}

func parse(k []byte, v []byte, nowNano uint64) *Report {
	sessionID, _ := uuid.FromBytes(k)
	r := &Report{
		Session: Session{
			SessionID: sessionID.String(),
		},
	}
	i := 0

	// Delta Rx, Tx
	r.DeltaRx, r.DeltaTx = ParseRxTx(v[i : i+16])
	i += 16

	// Start collecting timestamp in seconds
	r.CreatedNano = ParseUint64(v[i : i+8])
	i += 8
	r.DeltaTNano = nowNano - r.CreatedNano

	// Data
	dataLen := int(ParseUint16(v[i : i+2]))
	i += 2
	// Must not be any error
	_ = json.Unmarshal(v[i:i+dataLen], &r.SessionData)

	return r
}

func toValue(session *Session, drx, dtx uint64) []byte {
	// [8] Rx +
	// [8] Tx +
	// [8] StartT +
	// [2] len(Data) +
	// [.] Data

	data, _ := json.Marshal(session.SessionData)
	d := make([]byte, 8+8+8+2+len(data))

	i := 0
	// Delta Rx, Tx
	RxTx(drx, dtx, d[i:i+16])
	i += 16

	// Start timestamp
	nowNano := NowNano()
	SetUint64(nowNano, d[i:i+8])
	i += 8

	// Data
	dataLen := len(data)
	SetUint16(uint16(dataLen), d[i:i+2])
	i += 2
	copy(d[i:i+dataLen], data)

	return d
}

func New(flushInterval time.Duration, onFlush OnFlush) (*Service, error) {
	s := &Service{
		onFlush: onFlush,
	}
	var err error
	s.sessions, err = xcache.New(maxBytes, s.onEvict)
	if err != nil {
		return nil, err
	}
	go s.run(flushInterval)
	return s, nil
}

func (s *Service) ReportStats(sessionID uuid.UUID, drx, dtx uint64, onData OnData) {
	s.sessions.Update(sessionID[:], func(v []byte) ([]byte, bool, error) {
		if len(v) == 0 {
			session := &Session{SessionID: sessionID.String()}
			onData(sessionID, &session.SessionData)
			return toValue(session, drx, dtx), true, nil
		}
		return AddRxTx(drx, dtx, v), false, nil
	})
}

func (s *Service) run(flushInterval time.Duration) {
	ticker := time.NewTicker(flushInterval)
	for range ticker.C {
		// It causes the onEvict been called if any
		s.sessions.Reset()
	}
}

func (s *Service) onEvict(evicted *xcache.Items) {
	nowNano := NowNano()
	for i := range evicted.Values {
		r := parse(evicted.Keys[i], evicted.Values[i], nowNano)
		s.onFlush(r)
	}
}
