package subpub

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	subpubv1 "github.com/golang-vk-backend-challenge/internal/gen/go"
	"github.com/golang-vk-backend-challenge/internal/services/subpub"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type subPubServer struct {
	subpubv1.UnimplementedSubPubServer
	sp  *subpub.SubPub
	log *slog.Logger
}

func Register(gRPC *grpc.Server, sp *subpub.SubPub, log *slog.Logger) {
	subpubv1.RegisterSubPubServer(gRPC, &subPubServer{
		sp:  sp,
		log: log,
	})
}

func (sps *subPubServer) Subscribe(request *subpubv1.SubscribeRequest, stream grpc.ServerStreamingServer[subpubv1.Event]) error {
	subject := request.Key

	done := make(chan struct{})
	defer close(done)

	cb := func(msg any) {
		select {
		case <-stream.Context().Done():
			return
		default:
			if err := stream.Send(&subpubv1.Event{
				Data: fmt.Sprintf("%v", msg),
			}); err != nil {
				sps.log.Warn("failed to send a message to a client",
					slog.String("error", err.Error()),
				)
			}
		}
	}

	subscription, err := sps.sp.Subscribe(subject, cb)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to subscribe: %v", err)
	}

	subscription.WG.Add(1)
	go func() {
		defer subscription.WG.Done()
		for {
			select {
			case msg, ok := <-subscription.Queue:
				if !ok {
					return
				}
				cb(msg)
			case <-stream.Context().Done():
				return
			}
		}
	}()

	sps.log.Info("client subscribed", slog.String("subject", subject))
	<-stream.Context().Done()

	subscription.Unsubscribe()
	sps.log.Info("client has unsubscribed", slog.String("subject", subject))
	return nil
}

func (sps *subPubServer) Publish(ctx context.Context, request *subpubv1.PublishRequest) (*emptypb.Empty, error) {
	subject := request.Key
	data := request.Data

	err := sps.sp.Publish(subject, data)
	if errors.Is(err, subpub.ErrUnknownSubject) || errors.Is(err, subpub.ErrEmptySubject) {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "faild to publish message: %v", err)
	}

	sps.log.Info("message was published in subject", slog.String("message", data), slog.String("subject", subject))
	return &emptypb.Empty{}, nil
}
