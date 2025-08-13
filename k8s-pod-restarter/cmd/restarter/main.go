package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/viniciushammett/k8s-pod-restarter/internal/api"
	"github.com/viniciushammett/k8s-pod-restarter/internal/config"
	"github.com/viniciushammett/k8s-pod-restarter/internal/k8s"
	"github.com/viniciushammett/k8s-pod-restarter/internal/logger"
	"github.com/viniciushammett/k8s-pod-restarter/internal/metrics"
	"github.com/viniciushammett/k8s-pod-restarter/internal/restarter"
	"github.com/viniciushammett/k8s-pod-restarter/internal/scheduler"
)

var (
	version = "0.1.0"
	commit  = "dev"
	date    = "unknown"
)

func main() {
	log := logger.New(os.Getenv("LOG_LEVEL"))

	root := &cobra.Command{
		Use:   "pod-restarter",
		Short: "Kubernetes Pod Restarter (CLI + API + Scheduler + Metrics)",
	}

	// Global flags
	var kubeconfig, contextName string
	root.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "path to kubeconfig (out-of-cluster)")
	root.PersistentFlags().StringVar(&contextName, "context", "", "kubeconfig context to use")

	// healthz/metrics HTTP server (shared)
	var httpAddr string
	root.PersistentFlags().StringVar(&httpAddr, "http-addr", ":8080", "HTTP listen address for /metrics and (API mode) routes")

	// --- CLI command: restart ---
	restartCmd := &cobra.Command{
		Use:   "restart",
		Short: "Restart pods by label selector (delete pods to trigger controller recreation)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ns, _ := cmd.Flags().GetString("namespace")
			sel, _ := cmd.Flags().GetString("selector")
			dry, _ := cmd.Flags().GetBool("dry-run")
			force, _ := cmd.Flags().GetBool("force")
			maxAge, _ := cmd.Flags().GetDuration("max-age")
			grace, _ := cmd.Flags().GetDuration("grace-period")

			if sel == "" {
				return fmt.Errorf("--selector is required (e.g. app=myapp)")
			}

			ctx := withSignals()
			cs, err := k8s.NewClient(kubeconfig, contextName)
			if err != nil {
				return err
			}
			metrics.MustRegister()
			go serveMetrics(httpAddr, log)

			r := restarter.New(log, cs)
			opts := restarter.Options{
				Namespace:   ns,
				Selector:    sel,
				DryRun:      dry,
				Force:       force,
				MaxAge:      maxAge,
				GracePeriod: grace,
			}
			return r.RestartPods(ctx, opts)
		},
	}
	restartCmd.Flags().String("namespace", "", "namespace to filter (empty = all)")
	restartCmd.Flags().String("selector", "", "label selector (e.g. app=myapp,component=api)")
	restartCmd.Flags().Bool("dry-run", false, "simulate actions (no delete)")
	restartCmd.Flags().Bool("force", false, "delete even if owned by DaemonSet/StaticPod (not recommended)")
	restartCmd.Flags().Duration("max-age", 0, "only restart pods older than this (e.g. 30m, 2h)")
	restartCmd.Flags().Duration("grace-period", 30*time.Second, "grace period before force kill")
	root.AddCommand(restartCmd)

	// --- API mode ---
	var apiAuthToken string
	apiCmd := &cobra.Command{
		Use:   "api",
		Short: "Run REST API server to trigger restarts on-demand",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := withSignals()
			cs, err := k8s.NewClient(kubeconfig, contextName)
			if err != nil {
				return err
			}
			metrics.MustRegister()

			s := api.NewServer(api.ServerDeps{
				Log:       log,
				Clientset: cs,
			}, api.Config{
				Addr:     httpAddr,
				AuthToken: apiAuthToken,
			})

			log.Info().Str("addr", httpAddr).Msg("starting API")
			return s.Run(ctx)
		},
	}
	apiCmd.Flags().StringVar(&apiAuthToken, "api-token", "", "optional bearer token to protect API")
	root.AddCommand(apiCmd)

	// --- Scheduler mode ---
	var cfgPath string
	schedCmd := &cobra.Command{
		Use:   "scheduler",
		Short: "Run cron scheduler from YAML config",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := withSignals()
			cs, err := k8s.NewClient(kubeconfig, contextName)
			if err != nil {
				return err
			}
			metrics.MustRegister()
			go serveMetrics(httpAddr, log)

			conf, err := config.Load(cfgPath)
			if err != nil {
				return err
			}
			r := restarter.New(log, cs)
			return scheduler.Run(ctx, log, conf, r)
		},
	}
	schedCmd.Flags().StringVar(&cfgPath, "config", "configs/config.yaml", "scheduler YAML config path")
	root.AddCommand(schedCmd)

	// --- version ---
	root.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("pod-restarter %s (%s) %s\n", version, commit, date)
		},
	})

	if err := root.Execute(); err != nil {
		log.Error().Err(err).Msg("command failed")
		os.Exit(1)
	}
}

func withSignals() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() { <-c; cancel() }()
	return ctx
}

func serveMetrics(addr string, log *logger.Logger) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", metrics.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	log.Info().Str("addr", addr).Msg("metrics server listening")
	if err := http.ListenAndServe(addr, log.HTTPLogger(mux)); err != nil && !strings.Contains(err.Error(), "Server closed") {
		log.Error().Err(err).Msg("metrics http server error")
	}
}