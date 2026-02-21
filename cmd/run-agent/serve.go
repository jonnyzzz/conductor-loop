package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/api"
	"github.com/jonnyzzz/conductor-loop/internal/config"
	"github.com/spf13/cobra"
)

func newServeCmd() *cobra.Command {
	var (
		host       string
		port       int
		rootDir    string
		configPath string
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the HTTP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServe(host, port, cmd.Flags().Changed("port"), rootDir, configPath)
		},
	}

	cmd.Flags().StringVar(&host, "host", "127.0.0.1", "HTTP server host")
	cmd.Flags().IntVar(&port, "port", 14355, "HTTP server port")
	cmd.Flags().StringVar(&rootDir, "root", "", "run-agent root directory")
	cmd.Flags().StringVar(&configPath, "config", "", "config file path")

	return cmd
}

func runServe(host string, port int, explicitPort bool, rootDir, configPath string) error {
	logger := log.New(os.Stderr, "run-agent serve ", log.LstdFlags)

	server, err := api.NewServer(api.Options{
		RootDir:    rootDir,
		ConfigPath: configPath,
		APIConfig: config.APIConfig{
			Host: host,
			Port: port,
		},
		Version:          version,
		Logger:           logger,
		DisableTaskStart: true,
	})
	if err != nil {
		return err
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe(explicitPort)
	}()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-signalCh:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			return fmt.Errorf("shutdown server: %w", err)
		}
		return nil
	}
}
