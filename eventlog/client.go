package eventlog

import (
	"fmt"
	"net"
	"sync"

	"github.com/vpnhouse/common-lib-go/human"
	"github.com/vpnhouse/tunnel/proto"
	"go.uber.org/zap"
)

const (
	federationAuthHeader = "X-VPNHOUSE-FEDERATION-KEY"
	tunnelAuthHeader     = "X-VPNHOUSE-TUNNEL-KEY"
)

type Client struct {
	opts         options
	client       proto.EventLogServiceClient
	out          chan *Event
	once         sync.Once
	stop         chan struct{}
	done         chan struct{}
	eventlogSync EventlogSync
	tunnelHost   string
	instanceID   string
}

func NewClient(instanceID string, tunnelHostPort string, eventlogSync EventlogSync, opt ...Option) (*Client, error) {
	tunnelHost, tunnelPort, err := net.SplitHostPort(tunnelHostPort)
	if err != nil || tunnelPort == "" {
		tunnelHost = tunnelHostPort
		tunnelPort = "8089" // Default port
	}
	opts := options{
		TunnelPort:             tunnelPort, // use port as default value in case no opts given
		TunnelID:               tunnelHost, // use host as default value in case no opts given
		LockTtl:                defaultLockTtl,
		LockProlongateTimeout:  defaultLockProlongateTimeout,
		ReportPositionInterval: defaultReportPositionInterval,
		WaitOutputWriteTimeout: defaultWaitOutputWriteTimeout,
	}
	for _, o := range opt {
		err := o(&opts)
		if err != nil {
			return nil, err
		}
	}
	if tunnelHost == "" {
		return nil, fmt.Errorf("tunnel host is not defined")
	}

	if instanceID == "" {
		return nil, fmt.Errorf("instance id is not defined")
	}

	return &Client{
		opts:         opts,
		out:          make(chan *Event),
		stop:         make(chan struct{}),
		done:         make(chan struct{}),
		tunnelHost:   tunnelHost,
		instanceID:   instanceID,
		eventlogSync: eventlogSync,
	}, nil
}

func (s *Client) Events() chan *Event {
	s.once.Do(func() {
		go func() {
			defer func() {
				close(s.out)
				close(s.done)
			}()
			lockTtl := s.getLockTtl()
			acquired, err := s.eventlogSync.Acquire(s.instanceID, s.tunnelHost, lockTtl)
			if !acquired {
				s.publishOrDrop(&Event{Error: fmt.Errorf("stop reading events as failed to acquire sync lock to process events: %w", ErrLockNotAcquired)})
				zap.L().Info("stop reading events as failed to acquire sync lock to process events",
					zap.String("instance_id", s.instanceID),
					zap.String("tunnel", s.tunnelHost),
					zap.Error(err),
				)
				return
			} else {
				zap.L().Debug("set sync lock",
					zap.String("instance_id", s.instanceID),
					zap.String("tunnel_id", s.opts.TunnelID),
					zap.Stringer("ttl", human.Interval(lockTtl)),
				)
			}
			err = s.connect()
			if err != nil {
				s.publishOrDrop(&Event{Error: err})
				return
			}
			s.readAndPublishEvents()
		}()
	})
	return s.out
}

func (s *Client) Close() {
	close(s.stop)
	<-s.done
}
