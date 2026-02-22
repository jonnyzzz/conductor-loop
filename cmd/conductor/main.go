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
	"strconv"
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
		host             string
		port             int
		apiKey           string
	)

	cmd := &cobra.Command{
		Use:          "conductor",
		Short:        "Conductor Loop orchestration CLI",
		Version:      version,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Pass non-zero values only when the flag was explicitly set,
			// so that config file values take precedence over flag defaults.
			cliHost := ""
			cliPort := 0
			explicitPort := cmd.Flags().Changed("port")
			if cmd.Flags().Changed("host") {
				cliHost = host
			}
			if explicitPort {
				cliPort = port
			}
			return runServer(configPath, rootDir, disableTaskStart, cliHost, cliPort, explicitPort, apiKey)
		},
	}
	cmd.SetVersionTemplate("{{.Version}}\n")

	cmd.Flags().StringVar(&configPath, "config", "", "config file path")
	cmd.Flags().StringVar(&rootDir, "root", "", "run-agent root directory")
	cmd.Flags().BoolVar(&disableTaskStart, "disable-task-start", false, "disable task execution")
	cmd.Flags().StringVar(&host, "host", "0.0.0.0", "HTTP listen host (overrides config)")
	cmd.Flags().IntVar(&port, "port", 14355, "HTTP listen port (overrides config)")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key for authentication (enables auth when set)")

	cmd.AddCommand(newStatusCmd())
	cmd.AddCommand(newGoalCmd())
	cmd.AddCommand(newWorkflowCmd())
	cmd.AddCommand(newTaskCmd())
	cmd.AddCommand(newJobCmd())
	cmd.AddCommand(newProjectCmd())
	cmd.AddCommand(newWatchCmd())
	cmd.AddCommand(newBusCmd())

	return cmd
}

func runServer(configPath, rootDir string, disableTaskStart bool, cliHost string, cliPort int, explicitPort bool, cliAPIKey string) error {
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

	if configPath == "" {
		found, err := config.FindDefaultConfig()
		if err != nil {
			return err
		}
		if found == "" {
			return fmt.Errorf("no config file found; use --config to specify one")
		}
		logger.Printf("Using config: %s", found)
		configPath = found
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

	// Env vars override config file but are overridden by explicit CLI flags.
	if cliHost == "" {
		if h := strings.TrimSpace(os.Getenv("CONDUCTOR_HOST")); h != "" {
			cliHost = h
		}
	}
	if cliPort == 0 {
		if portStr := strings.TrimSpace(os.Getenv("CONDUCTOR_PORT")); portStr != "" {
			if p, err := strconv.Atoi(portStr); err == nil {
				cliPort = p
				explicitPort = true
			}
		}
	}

	// CLI flags override config file values when explicitly provided.
	if cliHost != "" {
		apiConfig.Host = cliHost
	}
	if cliPort != 0 {
		apiConfig.Port = cliPort
	}
	if cliAPIKey != "" {
		apiConfig.AuthEnabled = true
		apiConfig.APIKey = cliAPIKey
	}
	if apiConfig.AuthEnabled && apiConfig.APIKey == "" {
		logger.Printf("WARNING: auth_enabled=true but no api_key set; authentication disabled")
		apiConfig.AuthEnabled = false
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

func parseBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}
