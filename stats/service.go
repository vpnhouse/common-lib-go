package stats

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/vpnhouse/common-lib-go/xcache"
)

const maxBytes = 32 << 20 // 32 Mb

type Service[T any] struct {
	sessions *xcache.Cache // session_id -> rx, tx, timestamp, data {installation_id, user_id, country etc.}
	onFlush  OnFlush[T]
}

type Session[T any] struct {
	SessionID string // uuid
	Data      *T
}

type Report[T any] struct {
	Session[T]
	Created uint64
	DeltaRx uint64
	DeltaTx uint64
	DeltaT  uint64
}

type Extra map[string]string

type OnFlush[T any] func(report *Report[T])

var Now func() uint64 = func() uint64 {
	return uint64(time.Now().Unix())
}

func parse[T any](k []byte, v []byte, nowT uint64) *Report[T] {
	sessionID, _ := uuid.FromBytes(k)
	r := &Report[T]{
		Session: Session[T]{
			SessionID: sessionID.String(),
			Data:      new(T),
		},
	}
	i := 0

	// Delta Rx, Tx
	r.DeltaRx, r.DeltaTx = ParseRxTx(v[i : i+16])
	i += 16

	// Start collecting timestamp in seconds
	r.Created = ParseUint64(v[i : i+8])
	i += 8
	r.DeltaT = nowT - r.Created

	// Data
	dataLen := int(ParseUint16(v[i : i+2]))
	i += 2
	// Must not be any error
	_ = json.Unmarshal(v[i:i+dataLen], r.Data)

	return r
}

func toValue[T any](session *Session[T], drx, dtx uint64) []byte {
	// [8] Rx +
	// [8] Tx +
	// [8] StartT +
	// [2] len(Data) +
	// [.] Data

	data, _ := json.Marshal(session.Data)
	d := make([]byte, 8+8+8+2+len(data))

	i := 0
	// Delta Rx, Tx
	RxTx(drx, dtx, d[i:i+16])
	i += 16

	// Start timestamp
	now := Now()
	SetUint64(now, d[i:i+8])
	i += 8

	// Data
	dataLen := len(data)
	SetUint16(uint16(dataLen), d[i:i+2])
	i += 2
	copy(d[i:i+dataLen], data)

	return d
}

func New[T any](flushInterval time.Duration, onFlush OnFlush[T]) (*Service[T], error) {
	s := &Service[T]{
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

func (s *Service[T]) ReportStats(sessionID uuid.UUID, drx, dtx uint64, data func(sessionID uuid.UUID) *T) {
	s.sessions.Update(sessionID[:], func(v []byte) ([]byte, bool, error) {
		if len(v) == 0 {
			data := data(sessionID)
			return toValue(&Session[T]{
				SessionID: sessionID.String(),
				Data:      data,
			}, drx, dtx), true, nil
		}
		return AddRxTx(drx, dtx, v), false, nil
	})
}

func (s *Service[T]) run(flushInterval time.Duration) {
	ticker := time.NewTicker(flushInterval)
	for range ticker.C {
		// It causes the onEvict been called if any
		s.sessions.Reset()
	}
}

func (s *Service[T]) onEvict(evicted *xcache.Items) {
	now := Now()
	for i := range evicted.Values {
		r := parse[T](evicted.Keys[i], evicted.Values[i], now)
		s.onFlush(r)
	}
}
