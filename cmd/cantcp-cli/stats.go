package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	nethttp "net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/burn-lab-dev/cantcp/internal/app/domain"
	"github.com/burn-lab-dev/cantcp/internal/config"
	"github.com/burn-lab-dev/cantcp/internal/version"
)

// statsTimeout bounds the statistics HTTP request.
const statsTimeout = 10 * time.Second

// runStats implements "cantcp-cli stats": it fetches the daemon statistics
// over HTTP and prints them as a table or as raw JSON.
func runStats(args []string) error {
	flags := flag.NewFlagSet("cantcp-cli stats", flag.ContinueOnError)
	f := config.RegisterClientFlags(flags)
	asJSON := flags.Bool("json", false, "print the raw JSON response")
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: cantcp-cli stats [flags]\n\n"+
			"Fetches %s/api/v1/stats from the server.\n\nFlags:\n", "http://host:port")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return err
	}
	if f.Version {
		fmt.Println("cantcp-cli", version.Version)
		return nil
	}
	cfg, err := config.LoadClient(flags, f, os.LookupEnv, os.ReadFile)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), statsTimeout)
	defer cancel()
	url := strings.TrimSuffix(cfg.StatsServer, "/") + "/api/v1/stats"
	req, err := nethttp.NewRequestWithContext(ctx, nethttp.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := nethttp.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("statistics request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != nethttp.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("statistics request: HTTP %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	if *asJSON {
		_, err := io.Copy(os.Stdout, resp.Body)
		return err
	}
	var snap domain.Stats
	if err := json.NewDecoder(resp.Body).Decode(&snap); err != nil {
		return fmt.Errorf("statistics response: %w", err)
	}
	printStatsTable(os.Stdout, &snap)
	return nil
}

// printStatsTable renders the snapshot as a human-readable table.
func printStatsTable(w io.Writer, snap *domain.Stats) {
	fmt.Fprintf(w, "version:  %s\n", snap.Version)
	fmt.Fprintf(w, "uptime:   %s\n", (time.Duration(snap.UptimeSeconds) * time.Second).String())
	fmt.Fprintf(w, "CAN %s: frames read %d, written %d, read errors %d, write errors %d\n",
		snap.CAN.Interface, snap.CAN.FramesRead, snap.CAN.FramesWritten, snap.CAN.ReadErrors, snap.CAN.WriteErrors)
	fmt.Fprintf(w, "TCP:      clients %d, connections total %d, frames in %d, out %d, bytes in %d, out %d, dropped %d\n",
		snap.TCP.ConnectionsCurrent, snap.TCP.ConnectionsTotal,
		snap.TCP.FramesIn, snap.TCP.FramesOut, snap.TCP.BytesIn, snap.TCP.BytesOut, snap.TCP.Dropped)
	if len(snap.Clients) == 0 {
		return
	}
	fmt.Fprintln(w)
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "SOCKET\tSINCE\tFRAMES IN\tFRAMES OUT\tBYTES IN\tBYTES OUT\tDROPPED")
	for _, c := range snap.Clients {
		fmt.Fprintf(tw, "%s\t%s\t%d\t%d\t%d\t%d\t%d\n",
			c.Socket, c.Since.Format(time.RFC3339), c.FramesIn, c.FramesOut, c.BytesIn, c.BytesOut, c.Dropped)
	}
	_ = tw.Flush()
}
