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
	"github.com/jonnyzzz/conductor-loop/internal/obslog"
	"github.com/spf13/cobra"
)

func newServeCmd() *cobra.Command {
	var (
		host                string
		port                int
		rootDir             string
		configPath          string
		disableTaskStart    bool
		apiKey              string
		watchdogInterval    time.Duration
		watchdogMaxFailures int
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the run-agent HTTP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			cliHost := ""
			cliPort := 0
			explicitPort := cmd.Flags().Changed("port")
			if cmd.Flags().Changed("host") {
				cliHost = host
			}
			if explicitPort {
				cliPort = port
			}
			return runServe(configPath, rootDir, disableTaskStart, cliHost, cliPort, explicitPort, apiKey, watchdogInterval, watchdogMaxFailures)
		},
	}

	cmd.Flags().StringVar(&host, "host", "0.0.0.0", "HTTP server host (overrides config)")
	cmd.Flags().IntVar(&port, "port", 14355, "HTTP server port (overrides config)")
	cmd.Flags().StringVar(&rootDir, "root", "", "run-agent root directory")
	cmd.Flags().StringVar(&configPath, "config", "", "config file path")
	cmd.Flags().BoolVar(&disableTaskStart, "disable-task-start", false, "disable task execution (monitoring-only mode)")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key for authentication (enables auth when set)")
	cmd.Flags().DurationVar(&watchdogInterval, "watchdog-interval", 30*time.Second, "interval between server health probe attempts")
	cmd.Flags().IntVar(&watchdogMaxFailures, "watchdog-max-failures", 3, "consecutive health probe failures before exiting")

	return cmd
}

func runServe(configPath, rootDir string, disableTaskStart bool, cliHost string, cliPort int, explicitPort bool, cliAPIKey string, watchdogInterval time.Duration, watchdogMaxFailures int) error {
	logger := log.New(os.Stdout, "run-agent serve ", log.LstdFlags)

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
			obslog.Log(logger, "ERROR", "startup", "config_discovery_failed",
				obslog.F("error", err),
			)
			return err
		}
		if found != "" {
			logger.Printf("Using config: %s", found)
			obslog.Log(logger, "INFO", "startup", "config_discovered",
				obslog.F("config_path", found),
			)
			configPath = found
		}
	}

	var (
		apiCfg config.APIConfig
		cfg    *config.Config
	)

	if configPath != "" {
		loaded, err := config.LoadConfigForServer(configPath)
		if err != nil {
			logger.Printf("config load failed: %v (continuing with defaults)", err)
			obslog.Log(logger, "ERROR", "startup", "config_load_failed",
				obslog.F("config_path", configPath),
				obslog.F("error", err),
			)
		} else {
			cfg = loaded
			apiCfg = loaded.API
			obslog.Log(logger, "INFO", "startup", "config_loaded",
				obslog.F("config_path", configPath),
				obslog.F("agent_count", len(loaded.Agents)),
			)
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
		apiCfg.Host = cliHost
	}
	if cliPort != 0 {
		apiCfg.Port = cliPort
	}
	if cliAPIKey != "" {
		apiCfg.AuthEnabled = true
		apiCfg.APIKey = cliAPIKey
	}
	if apiCfg.AuthEnabled && apiCfg.APIKey == "" {
		logger.Printf("WARNING: auth_enabled=true but no api_key set; authentication disabled")
		obslog.Log(logger, "WARN", "startup", "auth_disabled_missing_api_key")
		apiCfg.AuthEnabled = false
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
		APIConfig:        apiCfg,
		RootTaskLimit:    rootTaskLimit(cfg),
		Version:          version,
		AgentNames:       agentNames,
		Logger:           logger,
		DisableTaskStart: disableTaskStart,
	})
	if err != nil {
		obslog.Log(logger, "ERROR", "startup", "server_init_failed",
			obslog.F("error", err),
		)
		return err
	}
	fmt.Printf("run-agent %s\n", version)
	fmt.Println("By @jonnyzzz · https://linkedin.com/in/jonnyzzz · Support / Donate / Follow")
	obslog.Log(logger, "INFO", "startup", "server_starting",
		obslog.F("version", version),
		obslog.F("root_dir", rootDir),
		obslog.F("config_path", configPath),
		obslog.F("host", apiCfg.Host),
		obslog.F("port", apiCfg.Port),
		obslog.F("task_start_enabled", !disableTaskStart),
		obslog.F("auth_enabled", apiCfg.AuthEnabled),
		obslog.F("watchdog_interval", watchdogInterval),
		obslog.F("watchdog_max_failures", watchdogMaxFailures),
	)

	// Start watchdog health probe.
	watchdog := &api.Watchdog{
		Server:      server,
		Host:        loopbackHost(apiCfg.Host),
		Interval:    watchdogInterval,
		MaxFailures: watchdogMaxFailures,
		Logger:      log.New(os.Stderr, "run-agent watchdog ", log.LstdFlags),
	}
	go watchdog.Run()

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe(explicitPort)
	}()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			obslog.Log(logger, "INFO", "startup", "server_stopped")
			return nil
		}
		obslog.Log(logger, "ERROR", "startup", "server_stopped_with_error",
			obslog.F("error", err),
		)
		return err
	case sig := <-signalCh:
		obslog.Log(logger, "INFO", "startup", "shutdown_signal_received",
			obslog.F("signal", sig.String()),
		)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			obslog.Log(logger, "ERROR", "startup", "server_shutdown_failed",
				obslog.F("error", err),
			)
			return fmt.Errorf("shutdown server: %w", err)
		}
		obslog.Log(logger, "INFO", "startup", "server_shutdown_completed")
		return nil
	}
}

// loopbackHost returns "127.0.0.1" when host is an unspecified address (0.0.0.0 or ::),
// and the host value itself otherwise.
func loopbackHost(host string) string {
	host = strings.TrimSpace(host)
	if host == "" || host == "0.0.0.0" || host == "::" {
		return "127.0.0.1"
	}
	return host
}

func parseBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func rootTaskLimit(cfg *config.Config) int {
	if cfg == nil {
		return 0
	}
	if cfg.Defaults.MaxConcurrentRootTasks < 0 {
		return 0
	}
	return cfg.Defaults.MaxConcurrentRootTasks
}
