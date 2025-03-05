package stats

import (
	"time"

	"github.com/google/uuid"
	"github.com/vpnhouse/common-lib-go/xcache"
	"github.com/vpnhouse/common-lib-go/xutils"
)

const maxBytes = 32 << 20 // 32 Mb

type Service struct {
	sessions *xcache.Cache // session_id -> rx, tx, timestamp, session {session_id, installation_id, user_id}
	onFlush  OnFlush
}

type Session struct {
	SessionID      string // uuid
	InstallationID string // uuid
	UserID         string // project(uuid)/auth_method(uuid)/ref(firebase|keycloak id)
}

type Report struct {
	Session
	DeltaRx uint64
	DeltaTx uint64
	DeltaT  uint64
}

type OnFlush func(report *Report)

var Now func() uint64 = func() uint64 {
	return uint64(time.Now().Unix())
}

func parse(k []byte, v []byte, nowT uint64) *Report {
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
	startT := ParseUint64(v[i : i+8])
	i += 8
	r.DeltaT = nowT - startT

	// Installation ID
	installationIDLen := int(ParseUint16(v[i : i+2]))
	i += 2
	r.InstallationID = xutils.BytesToString(v[i : i+installationIDLen])
	i += installationIDLen

	// User ID
	userIDLen := int(ParseUint16(v[i : i+2]))
	i += 2
	r.UserID = xutils.BytesToString(v[i : i+userIDLen])

	return r
}

func toValue(session *Session, drx, dtx uint64) []byte {
	// [8] Rx +
	// [8] Tx +
	// [8] StartT +
	// [2] len(InstallationID) +
	// [.] len(InstallationID) +
	// [2] len(UserID) +
	// [.] len(UserID)

	i := 0
	// Delta Rx, Tx
	d := make([]byte, 8+8+8+2+len(session.InstallationID)+2+len(session.UserID))
	RxTx(drx, dtx, d[i:i+16])
	i += 16

	// Start timestamp
	now := Now()
	SetUint64(now, d[i:i+8])

	// Installation ID
	installationIDLen := len(session.InstallationID)
	SetUint16(uint16(installationIDLen), d[i:i+2])
	i += 2
	copy(d[i:i+installationIDLen], xutils.StringToBytes(session.InstallationID))
	i += installationIDLen

	// User ID
	userIDLen := len(session.UserID)
	SetUint16(uint16(userIDLen), d[i:i+2])
	i += 2
	copy(d[i:i+userIDLen], xutils.StringToBytes(session.UserID))

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

func (s *Service) ReportStats(sessionID uuid.UUID, drx, dtx uint64, metadata func(sessionID uuid.UUID) *Session) {
	s.sessions.Update(sessionID[:], func(v []byte) ([]byte, bool, error) {
		if len(v) == 0 {
			session := metadata(sessionID)
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
	now := Now()
	for i := range evicted.Values {
		r := parse(evicted.Keys[i], evicted.Values[i], now)
		s.onFlush(r)
	}
}
