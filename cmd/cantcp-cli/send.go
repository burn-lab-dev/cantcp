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

// runSend implements "cantcp-cli send": it sends one frame built from the
// flags, or all frames read from a candump-style file or stdin.
func runSend(args []string) error {
	flags := flag.NewFlagSet("cantcp-cli send", flag.ContinueOnError)
	f := config.RegisterClientFlags(flags)
	id := flags.String("id", "", "frame identifier in hexadecimal, for example 123 or 1ABCDE")
	data := flags.String("data", "", "payload bytes in hexadecimal, for example 11223344")
	ext := flags.Bool("ext", false, "force the extended (29-bit) identifier")
	fd := flags.Bool("fd", false, "send a CAN FD frame")
	brs := flags.Bool("brs", false, "set the CAN FD bit rate switch flag")
	count := flags.Int("count", 1, "send the frame this many times (only with --id)")
	interval := flags.Duration("interval", 0, "delay between the sends (only with --id)")
	input := flags.String("input", "", `file with candump-style frames, "-" reads stdin`)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: cantcp-cli send [flags] [file]\n\n"+
			"Sends frames to the server. With --id it builds one frame from --id and\n"+
			"--data; with --input or a file argument it reads candump-style lines\n"+
			"(one frame per line, see the documentation).\n\nFlags:\n")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return err
	}
	if f.Version {
		fmt.Println("cantcp-cli", version.Version)
		return nil
	}
	path := *input
	if path == "" && flags.NArg() > 0 {
		path = flags.Arg(0)
	}
	if *id == "" && path == "" {
		return errors.New("either --id or --input/positional file must be set")
	}
	if *id != "" && path != "" {
		return errors.New("--id and --input/positional file are mutually exclusive")
	}
	if path != "" && (*count != 1 || *interval != 0) {
		return errors.New("--count and --interval are only valid with --id")
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

	if *id != "" {
		frame, err := buildFrame(*id, *data, *ext, *fd, *brs)
		if err != nil {
			return err
		}
		return sendFrames(conn, []cantcp.Frame{frame}, *count, *interval, os.Stderr)
	}
	frames, err := readFrameFile(path)
	if err != nil {
		return err
	}
	if len(frames) == 0 {
		return errors.New("no frames to send")
	}
	return sendFrames(conn, frames, 1, 0, os.Stderr)
}

// readFrameFile reads candump-style frames from a file or stdin ("-").
func readFrameFile(path string) ([]cantcp.Frame, error) {
	var r io.Reader
	if path == "-" {
		r = os.Stdin
	} else {
		file, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		r = file
	}
	var frames []cantcp.Frame
	scanner := bufio.NewScanner(r)
	line := 0
	for scanner.Scan() {
		line++
		frame, err := parseFrameLine(scanner.Text())
		switch {
		case errors.Is(err, errCommentLine):
			continue
		case err != nil:
			return nil, fmt.Errorf("%s:%d: %w", path, line, err)
		default:
			frames = append(frames, frame)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return frames, nil
}

// sendFrames encodes the frames to the connection: count times in order with
// the given delay between the passes.
func sendFrames(conn net.Conn, frames []cantcp.Frame, count int, interval time.Duration, out io.Writer) error {
	enc := cantcp.NewEncoder(conn)
	sent := 0
	for pass := 0; pass < count; pass++ {
		for i := range frames {
			// MarshalBinary validates the frame and returns the raw
			// SocketCAN layout: frames built from flags may not carry it yet.
			raw, err := frames[i].MarshalBinary()
			if err != nil {
				return fmt.Errorf("send frame: %w", err)
			}
			if err := enc.Encode(raw); err != nil {
				return fmt.Errorf("send frame: %w", err)
			}
			sent++
		}
		if interval > 0 && pass < count-1 {
			time.Sleep(interval)
		}
	}
	fmt.Fprintf(out, "sent %d frame(s)\n", sent)
	return nil
}
