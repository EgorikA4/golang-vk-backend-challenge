package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	subpubgrpc "github.com/golang-vk-backend-challenge/internal/grpc/subpub"
	"github.com/golang-vk-backend-challenge/internal/services/subpub"
	"google.golang.org/grpc"
)

type App struct {
	log        *slog.Logger
	gRPCServer *grpc.Server
	port       int
	sp         *subpub.SubPub
}

func New(log *slog.Logger, port int, sp *subpub.SubPub) *App {
	gRPCServer := grpc.NewServer()
	subpubgrpc.Register(gRPCServer, sp, log)
	return &App{
		log:        log,
		gRPCServer: gRPCServer,
		port:       port,
		sp:         sp,
	}
}

func (a *App) MustRun() {
	if err := a.Run(); err != nil {
		panic(err)
	}
}

func (a *App) Run() error {
	const op = "grpcapp.Run"

	log := a.log.With(slog.String("op", op))

	listener, err := net.Listen("tcp4", fmt.Sprintf(":%d", a.port))
	if err != nil {
		return fmt.Errorf("%s: %v", op, err)
	}

	defer func() {
		if err := listener.Close(); err != nil {
			log.Warn("listener was closed with error", slog.String("error", err.Error()))
		}
	}()

	log.Info("grpc server is running", slog.String("addr", listener.Addr().String()))
	if err := a.gRPCServer.Serve(listener); err != nil {
		return fmt.Errorf("%s: %v", op, err)
	}
	return nil
}

func (a *App) Stop(ctx context.Context) {
	const (
		op          = "grpcapp.Stop"
		stopTimeout = 5 * time.Second
	)

	a.log.With(slog.String("op", op)).Info(
		"stopping grpc server", slog.Int("port", a.port),
	)

	done := make(chan struct{})
	go func() {
		a.gRPCServer.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(stopTimeout):
		a.log.Warn("GracefulStop timed out, forcing stop")
		a.gRPCServer.Stop()
	}

	if err := a.sp.Close(ctx); err != nil {
		a.log.Warn("failed to close SubPub", slog.String("error", err.Error()))
	}
}
