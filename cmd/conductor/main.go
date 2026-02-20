package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/api"
	"github.com/jonnyzzz/conductor-loop/internal/config"
	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	var (
		configPath       string
		rootDir          string
		disableTaskStart bool
	)

	cmd := &cobra.Command{
		Use:          "conductor",
		Short:        "Conductor Loop orchestration CLI",
		Version:      version,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServer(configPath, rootDir, disableTaskStart)
		},
	}
	cmd.SetVersionTemplate("{{.Version}}\n")

	cmd.Flags().StringVar(&configPath, "config", "", "config file path")
	cmd.Flags().StringVar(&rootDir, "root", "", "run-agent root directory")
	cmd.Flags().BoolVar(&disableTaskStart, "disable-task-start", false, "disable task execution")

	cmd.AddCommand(newTaskCmd())
	cmd.AddCommand(newJobCmd())

	return cmd
}

func newTaskCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "task",
		Short: "Manage tasks",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println("task command not implemented yet")
		},
	}
}

func newJobCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "job",
		Short: "Manage jobs",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println("job command not implemented yet")
		},
	}
}

func runServer(configPath, rootDir string, disableTaskStart bool) error {
	logger := log.New(os.Stdout, "conductor ", log.LstdFlags)

	configPath = strings.TrimSpace(configPath)
	if configPath == "" {
		configPath = strings.TrimSpace(os.Getenv("CONDUCTOR_CONFIG"))
	}
	rootDir = strings.TrimSpace(rootDir)
	if rootDir == "" {
		rootDir = strings.TrimSpace(os.Getenv("CONDUCTOR_ROOT"))
	}
	if envDisable := strings.TrimSpace(os.Getenv("CONDUCTOR_DISABLE_TASK_START")); envDisable != "" {
		disableTaskStart = parseBool(envDisable)
	}

	var (
		apiConfig config.APIConfig
		cfg       *config.Config
	)

	if configPath != "" {
		loaded, err := config.LoadConfigForServer(configPath)
		if err != nil {
			logger.Printf("config load failed: %v (continuing with defaults)", err)
			disableTaskStart = true
		} else {
			cfg = loaded
			apiConfig = loaded.API
		}
	}

	if rootDir == "" && cfg != nil {
		rootDir = strings.TrimSpace(cfg.Storage.RunsDir)
	}

	var extraRoots []string
	if cfg != nil {
		extraRoots = cfg.Storage.ExtraRoots
	}

	var agentNames []string
	if cfg != nil {
		for name := range cfg.Agents {
			agentNames = append(agentNames, name)
		}
		sort.Strings(agentNames)
	}

	server, err := api.NewServer(api.Options{
		RootDir:          rootDir,
		ExtraRoots:       extraRoots,
		ConfigPath:       configPath,
		APIConfig:        apiConfig,
		Version:          version,
		AgentNames:       agentNames,
		Logger:           logger,
		DisableTaskStart: disableTaskStart,
	})
	if err != nil {
		return err
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
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

func parseBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}
