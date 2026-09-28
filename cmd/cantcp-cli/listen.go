package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	cantcp "github.com/burn-lab-dev/cantcp-lib-go"

	"github.com/burn-lab-dev/cantcp/internal/config"
	"github.com/burn-lab-dev/cantcp/internal/interfaces/tcp"
	"github.com/burn-lab-dev/cantcp/internal/version"
)

// runListen implements "cantcp-cli listen": it connects to the server and
// prints the CAN frames it receives.
func runListen(args []string) error {
	flags := flag.NewFlagSet("cantcp-cli listen", flag.ContinueOnError)
	f := config.RegisterClientFlags(flags)
	idStr := flags.String("id", "", "only print frames matching this identifier (hexadecimal); with --mask")
	maskStr := flags.String("mask", "", "identifier mask for the filter (hexadecimal, default 0: all frames)")
	asJSON := flags.Bool("json", false, "print one JSON object per frame")
	count := flags.Int("count", 0, "stop after this many frames (0: unlimited)")
	timeout := flags.Duration("timeout", 0, "stop after this time (0: unlimited)")
	canName := flags.String("can-name", "can0", "interface name used in the candump output")
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: cantcp-cli listen [flags]\n\n"+
			"Connects to the server and prints the CAN frames.\n\nFlags:\n")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return err
	}
	if f.Version {
		fmt.Println("cantcp-cli", version.Version)
		return nil
	}
	filterID, err := parseID(*idStr, "id")
	if err != nil {
		return err
	}
	filterMask, err := parseID(*maskStr, "mask")
	if err != nil {
		return err
	}
	cfg, err := config.LoadClient(flags, f, os.LookupEnv, os.ReadFile)
	if err != nil {
		return err
	}
	logger, _ := config.NewLogger(cfg.Log, os.Stderr)

	conn, err := tcp.Dial(context.Background(), cfg, logger)
	if err != nil {
		return err
	}
	defer conn.Close()
	if *timeout > 0 {
		_ = conn.SetReadDeadline(time.Now().Add(*timeout))
	}
	return listenFrames(conn, filterID, filterMask, *asJSON, *canName, *count, os.Stdout)
}

// listenFrames decodes the stream and prints the frames that pass the
// filter. The output is flushed per frame so that a pipe sees the frames
// immediately.
func listenFrames(conn net.Conn, filterID, filterMask uint32, asJSON bool, canName string, count int, w io.Writer) error {
	dec := cantcp.NewDecoder(conn)
	out := bufio.NewWriter(w)
	defer out.Flush()
	printed := 0
	for {
		var frame cantcp.Frame
		if err := dec.DecodeFrameInto(&frame); err != nil {
			return listenEnd(err)
		}
		if !matchFilter(&frame, filterID, filterMask) {
			continue
		}
		if asJSON {
			if err := writeFrameJSON(out, &frame, time.Now()); err != nil {
				return err
			}
		} else {
			formatCandump(out, canName, &frame, time.Now())
			if err := out.WriteByte('\n'); err != nil {
				return err
			}
		}
		if err := out.Flush(); err != nil {
			return err
		}
		printed++
		if count > 0 && printed >= count {
			return nil
		}
	}
}

// listenEnd classifies the end of the stream: a clean close, a timeout, or a
// real decode error.
func listenEnd(err error) error {
	var nerr net.Error
	switch {
	case errors.Is(err, io.EOF), errors.Is(err, net.ErrClosed):
		return nil
	case errors.As(err, &nerr) && nerr.Timeout():
		return nil
	default:
		return fmt.Errorf("decode: %w", err)
	}
}
