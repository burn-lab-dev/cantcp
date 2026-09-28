// Command cantcpd is the cantcp gateway daemon: it bridges one Linux
// SocketCAN interface and TCP clients speaking the cantcp protocol. The
// connection is either plain (trusted perimeter) or TLS 1.3.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/burn-lab-dev/cantcp/internal/adapters/socketcan"
	"github.com/burn-lab-dev/cantcp/internal/adapters/tlsconfig"
	"github.com/burn-lab-dev/cantcp/internal/app/domain"
	"github.com/burn-lab-dev/cantcp/internal/app/services"
	"github.com/burn-lab-dev/cantcp/internal/config"
	statshttp "github.com/burn-lab-dev/cantcp/internal/interfaces/http"
	"github.com/burn-lab-dev/cantcp/internal/interfaces/tcp"
	"github.com/burn-lab-dev/cantcp/internal/repositories"
	"github.com/burn-lab-dev/cantcp/internal/version"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "cantcpd:", err)
		os.Exit(1)
	}
}

// run parses the command line, loads the configuration and runs the daemon
// until a signal or a fatal component error.
func run() error {
	flags := flag.NewFlagSet("cantcpd", flag.ContinueOnError)
	f := config.RegisterServerFlags(flags)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: cantcpd [flags]\n\n"+
			"Bridges one SocketCAN interface and TCP clients speaking the cantcp protocol.\n"+
			"Settings come from the defaults, the JSON file, the CANTCP_* environment\n"+
			"variables and the flags, in that order of precedence.\n\nFlags:\n")
		flags.PrintDefaults()
	}
	if err := flags.Parse(os.Args[1:]); err != nil {
		return err
	}
	if f.Version {
		fmt.Println("cantcpd", version.Version)
		return nil
	}
	cfg, err := config.LoadServer(flags, f, os.LookupEnv, os.ReadFile)
	if err != nil {
		return err
	}
	return serve(cfg, os.Args[1:])
}

// serve wires the adapters, the bridge and the listeners and runs them until
// the context is canceled.
func serve(cfg domain.ConfigServer, args []string) error {
	logger, levelVar := config.NewLogger(cfg.Log, os.Stderr)
	logger.Info("cantcpd starting",
		"version", version.Version, "can", cfg.CAN.Interface, "mode", mode(cfg))

	bus, err := socketcan.Open(cfg.CAN.Interface, cfg.CAN.ErrorFrames)
	if err != nil {
		return err
	}
	defer bus.Close()

	stats := repositories.NewStats(version.Version, cfg.CAN.Interface, time.Now())
	bridge := services.NewBridge(bus, stats, logger)

	var tlsReloader *tlsconfig.Reloader
	if cfg.TLS.Enabled() {
		tlsReloader, err = tlsconfig.NewServer(cfg.TLS)
		if err != nil {
			return err
		}
	}

	tcpServer := tcp.NewServer(cfg, bridge, stats, tlsReloader, logger)
	if err := tcpServer.Listen(); err != nil {
		return err
	}
	logger.Info("TCP listener started", "addr", tcpServer.Addr().String())

	var statsServer *statshttp.Server
	if cfg.Stats.Enable {
		statsServer = statshttp.NewServer(cfg.Stats, stats, logger)
		if err := statsServer.Listen(); err != nil {
			return err
		}
		logger.Info("statistics listener started", "addr", statsServer.Addr().String())
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		_ = bus.Close()
	}()
	go reloadLoop(ctx, logger, args, levelVar, tlsReloader, tcpServer)

	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		fatalErr error
	)
	runComponent := func(name string, fn func(context.Context) error) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := fn(ctx); err != nil {
				logger.Error("component stopped", "component", name, "error", err)
				mu.Lock()
				if fatalErr == nil {
					fatalErr = err
				}
				mu.Unlock()
				stop()
			}
		}()
	}
	runComponent("bridge", bridge.Run)
	runComponent("tcp", tcpServer.Serve)
	if statsServer != nil {
		runComponent("stats", statsServer.Run)
	}

	<-ctx.Done()
	wg.Wait()
	logger.Info("cantcpd stopped")
	if fatalErr != nil {
		return fatalErr
	}
	return nil
}

// reloadLoop re-reads the configuration on SIGHUP: the TLS certificate
// material, the log level and the limits are applied without a restart. The
// listen address, the CAN interface, the TLS switch and the statistics
// address are fixed for the lifetime of the process.
func reloadLoop(ctx context.Context, logger *slog.Logger, args []string, levelVar *slog.LevelVar, tlsReloader *tlsconfig.Reloader, tcpServer *tcp.Server) {
	sighup := make(chan os.Signal, 1)
	signal.Notify(sighup, syscall.SIGHUP)
	defer signal.Stop(sighup)
	for {
		select {
		case <-ctx.Done():
			return
		case <-sighup:
			newCfg, err := loadConfig(args)
			if err != nil {
				logger.Error("reload failed: the configuration is not applied", "error", err)
				continue
			}
			levelVar.Set(config.SlogLevel(newCfg.Log.Level))
			if tlsReloader != nil && newCfg.TLS.Enabled() {
				if err := tlsReloader.Reload(newCfg.TLS); err != nil {
					logger.Error("reload failed: the TLS material is not applied", "error", err)
					continue
				}
			}
			tcpServer.UpdateConfig(newCfg)
			logger.Info("configuration reloaded")
		}
	}
}

// loadConfig re-reads the configuration from the same command line arguments
// without writing to the flag set output.
func loadConfig(args []string) (domain.ConfigServer, error) {
	flags := flag.NewFlagSet("cantcpd", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	f := config.RegisterServerFlags(flags)
	if err := flags.Parse(args); err != nil {
		return domain.ConfigServer{}, err
	}
	return config.LoadServer(flags, f, os.LookupEnv, os.ReadFile)
}

// mode names the connection mode for the start log.
func mode(cfg domain.ConfigServer) string {
	if cfg.TLS.Enabled() {
		return "tls1.3"
	}
	return "plain"
}
