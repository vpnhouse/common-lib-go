package tunnel

import (
	"context"

	"github.com/vpnhouse/tunnel/proto"
	"google.golang.org/grpc/metadata"
)

type (
	AddRestriction    proto.AddRestriction
	DeleteRestriction proto.DeleteRestriction
)

func (s *Client) AddRestriction(ctx context.Context, addRestriction *AddRestriction) error {
	md := metadata.New(map[string]string{federationAuthHeader: s.authSecret})
	ctx = metadata.NewOutgoingContext(ctx, md)

	_, err := s.client.Event(ctx, &proto.EventRequest{
		Action: &proto.Action{
			AddRestriction: (*proto.AddRestriction)(addRestriction),
		},
	})
	return err
}

func (s *Client) DeleteRestriction(ctx context.Context, deleteRestriction *DeleteRestriction) error {
	md := metadata.New(map[string]string{federationAuthHeader: s.authSecret})
	ctx = metadata.NewOutgoingContext(ctx, md)

	_, err := s.client.Event(ctx, &proto.EventRequest{
		Action: &proto.Action{
			DeleteRestriction: (*proto.DeleteRestriction)(deleteRestriction),
		},
	})
	return err
}
