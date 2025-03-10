package stats

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/vpnhouse/common-lib-go/xrand"
)

type testData struct {
	InstallationID string `json:"installation_id"`
	UserID         string `json:"user_id"`
}

func TestService(t *testing.T) {
	var reports []*Report[testData]
	s, err := New(
		time.Second,
		func(report *Report[testData], extra Extra) {
			reports = append(reports, report)
			assert.Nil(t, extra)
		},
		nil,
	)
	assert.NoError(t, err)

	sessionID := uuid.New()
	session := &Session[testData]{
		Data: &testData{
			InstallationID: "installation_id_123",
			UserID: strings.Join([]string{
				uuid.New().String(),
				uuid.New().String(),
				xrand.RandomString(128),
			}, "/"),
		},
	}

	numMetadataCalls := 0
	data := func(sessionID uuid.UUID) *testData {
		numMetadataCalls++
		return session.Data
	}

	for i := 0; i < 10; i++ {
		s.ReportStats(sessionID, 123, 456, data)
	}

	// Metadata must be called 1 time for the same SessionID
	assert.Equal(t, 1, numMetadataCalls)

	time.Sleep(time.Second * 2)

	assert.Equal(t, 1, len(reports))
	report := reports[0]
	assert.Equal(t, sessionID.String(), report.SessionID)
	assert.Equal(t, session.Data.InstallationID, report.Data.InstallationID)
	assert.Equal(t, session.Data.UserID, report.Data.UserID)
	assert.Equal(t, uint64(123)*10, report.DeltaRx)
	assert.Equal(t, uint64(456)*10, report.DeltaTx)
	assert.True(t, report.DeltaT > 0)
}

func TestServiceEmptyData(t *testing.T) {
	var reports []*Report[testData]
	s, err := New(time.Second, func(report *Report[testData], extra Extra) {
		reports = append(reports, report)
		assert.Equal(t, Extra{"a": "b"}, extra)
	}, Extra{"a": "b"})
	assert.NoError(t, err)

	sessionID := uuid.New()
	session := &Session[testData]{}

	numMetadataCalls := 0
	data := func(sessionID uuid.UUID) *testData {
		numMetadataCalls++
		return session.Data
	}

	for i := 0; i < 10; i++ {
		s.ReportStats(sessionID, 123, 456, data)
	}

	// Metadata must be called 1 time for the same SessionID
	assert.Equal(t, 1, numMetadataCalls)

	time.Sleep(time.Second * 2)

	assert.Equal(t, 1, len(reports))
	report := reports[0]
	assert.Equal(t, sessionID.String(), report.SessionID)
	assert.Empty(t, report.Data.InstallationID)
	assert.Empty(t, report.Data.UserID)
	assert.Equal(t, uint64(123)*10, report.DeltaRx)
	assert.Equal(t, uint64(456)*10, report.DeltaTx)
	assert.True(t, report.DeltaT > 0)
}
