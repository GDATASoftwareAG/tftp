package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/gdatasoftwareag/tftp/v2/pkg/handler"
	"github.com/gdatasoftwareag/tftp/v2/pkg/secfs"
	"github.com/gdatasoftwareag/tftp/v2/pkg/tftp"
	"github.com/gdatasoftwareag/tftp/v2/pkg/udp"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	serveCmd = &cobra.Command{
		Use:   "serve",
		Short: "Run the tftp server",
		RunE:  serveTFTP,
	}
)

func serveTFTP(_ *cobra.Command, _ []string) (err error) {
	logger.Info("Creating TFTP Server",
		zap.Any("config", cfg),
	)

	responseHandling := tftp.NewResponseHandling()
	responseHandling.RegisterHandler(handler.NewHealthCheckHandler())
	responseHandling.RegisterHandler(handler.NewFSHandler(secfs.New(cfg.FSHandlerBaseDir), logger))

	var srv tftp.Server
	if srv, err = tftp.NewServer(logger, cfg.TFTP, udp.NewConnector(), responseHandling); err != nil {
		panic(err.Error())
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	err = srv.ListenAndServe(ctx)
	return
}
