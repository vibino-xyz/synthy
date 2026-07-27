package rpc

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"

	contracts "github.com/vibino-xyz/protos/contracts/build"
	"go.uber.org/fx"
	"google.golang.org/grpc"
)

const defaultGRPCPort = 9090

// NewGRPCServer builds a grpc.Server with every synthy RPC service registered.
// Add future services (e.g. additional retrieval surfaces) here.
func NewGRPCServer(retrievalServer *RetrievalServer) *grpc.Server {
	srv := grpc.NewServer()
	contracts.RegisterRetrievalServiceServer(srv, retrievalServer)
	return srv
}

// StartGRPCServer binds the gRPC server to the fx lifecycle. It listens on
// GRPC_PORT (default 9090) and stops gracefully. This mirrors the HTTP server
// wiring in commons/whttp.
func StartGRPCServer(lc fx.Lifecycle, srv *grpc.Server) {
	addr := fmt.Sprintf(":%d", grpcPort())
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			lis, err := net.Listen("tcp", addr)
			if err != nil {
				return fmt.Errorf("grpc listen on %s: %w", addr, err)
			}
			slog.Info("gRPC server listening", "addr", addr)
			go func() {
				if err := srv.Serve(lis); err != nil {
					slog.Error("gRPC server stopped", "error", err)
				}
			}()
			return nil
		},
		OnStop: func(context.Context) error {
			srv.GracefulStop()
			return nil
		},
	})
}

func grpcPort() int {
	if v := os.Getenv("GRPC_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			return p
		}
	}
	return defaultGRPCPort
}
