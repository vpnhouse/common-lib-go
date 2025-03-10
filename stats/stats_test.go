package stats

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/vpnhouse/common-lib-go/xrand"
)

func TestService(t *testing.T) {
	var reports []*Report
	s, err := New(
		time.Second,
		func(report *Report, extra Extra) {
			reports = append(reports, report)
			assert.Nil(t, extra)
		},
		nil,
	)
	assert.NoError(t, err)

	sessionID := uuid.New()
	session := &Session{
		InstallationID: "installation_id_123",
		UserID: strings.Join([]string{
			uuid.New().String(),
			uuid.New().String(),
			xrand.RandomString(128),
		}, "/"),
	}

	numMetadataCalls := 0
	metadata := func(sessionID uuid.UUID) *Session {
		numMetadataCalls++
		session.SessionID = sessionID.String()
		return session
	}

	for i := 0; i < 10; i++ {
		s.ReportStats(sessionID, 123, 456, metadata)
	}

	// Metadata must be called 1 time for the same SessionID
	assert.Equal(t, 1, numMetadataCalls)

	time.Sleep(time.Second * 2)

	assert.Equal(t, 1, len(reports))
	report := reports[0]
	assert.Equal(t, sessionID.String(), report.SessionID)
	assert.Equal(t, session.InstallationID, report.InstallationID)
	assert.Equal(t, session.UserID, report.UserID)
	assert.Equal(t, uint64(123)*10, report.DeltaRx)
	assert.Equal(t, uint64(456)*10, report.DeltaTx)
	assert.True(t, report.DeltaT > 0)
}

func TestServiceEmptyData(t *testing.T) {
	var reports []*Report
	s, err := New(time.Second, func(report *Report, extra Extra) {
		reports = append(reports, report)
		assert.Equal(t, Extra{"a": "b"}, extra)
	}, Extra{"a": "b"})
	assert.NoError(t, err)

	sessionID := uuid.New()
	session := &Session{}

	numMetadataCalls := 0
	metadata := func(sessionID uuid.UUID) *Session {
		numMetadataCalls++
		session.SessionID = sessionID.String()
		return session
	}

	for i := 0; i < 10; i++ {
		s.ReportStats(sessionID, 123, 456, metadata)
	}

	// Metadata must be called 1 time for the same SessionID
	assert.Equal(t, 1, numMetadataCalls)

	time.Sleep(time.Second * 2)

	assert.Equal(t, 1, len(reports))
	report := reports[0]
	assert.Equal(t, sessionID.String(), report.SessionID)
	assert.Empty(t, report.InstallationID)
	assert.Empty(t, report.UserID)
	assert.Equal(t, uint64(123)*10, report.DeltaRx)
	assert.Equal(t, uint64(456)*10, report.DeltaTx)
	assert.True(t, report.DeltaT > 0)
}
