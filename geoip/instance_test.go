package geoip

import (
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func setupLogger(t *testing.T) {
	loggerConfig := zap.NewProductionConfig()
	loggerConfig.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	logger, err := loggerConfig.Build()
	if err != nil {
		t.Fatal(err)
		return
	}

	zap.ReplaceGlobals(logger)
}

func TestGeoipInstance(t *testing.T) {
	t.SkipNow()

	setupLogger(t)

	ip := net.ParseIP("103.90.72.135")

	dbPath := "./db.mmdb"
	geoip, err := NewGeoip(dbPath)
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()

		for range 10 {
			country, err := geoip.GetCountry(ip)
			errMsg := ""
			if err != nil {
				errMsg = err.Error()
			}
			zap.L().Info("get country",
				zap.Stringer("ip", ip), zap.String("country", country), zap.String("err", errMsg))

			time.Sleep(time.Millisecond * 300)
		}
	}()

	go func() {
		defer wg.Done()

		for range 10 {
			modTime := time.Now().UTC().Add(time.Hour)
			err := os.Chtimes(dbPath, time.Time{}, modTime)
			errMsg := ""
			if err != nil {
				errMsg = err.Error()
			}
			zap.L().Info("change mod time",
				zap.Time("mod_time", modTime), zap.String("err", errMsg))

			time.Sleep(time.Millisecond * 250)
		}
	}()

	go func() {
		defer wg.Done()

		oldName := dbPath + ".tmp"
		newName := dbPath
		for range 10 {
			oldName, newName = newName, oldName
			err := os.Rename(oldName, newName)
			errMsg := ""
			if err != nil {
				errMsg = err.Error()
			}
			zap.L().Info("rename db file",
				zap.String("old", oldName), zap.String("new", newName), zap.String("err", errMsg))

			time.Sleep(time.Millisecond * 400)
		}
	}()

	wg.Wait()
}
