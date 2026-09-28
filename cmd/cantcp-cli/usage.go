package main

import (
	"fmt"
	"io"
)

// usageText is the top-level help.
const usageText = `cantcp-cli — client for cantcp servers (SocketCAN over TCP).

Usage:
  cantcp-cli <command> [flags]

Commands:
  listen    connect and print the CAN frames received from the server
  send      send frames built from flags or read from a candump-style file
  stats     fetch the server statistics over HTTP
  version   print the version
  help      print this help

Run "cantcp-cli <command> -h" for the command flags: connection address,
TLS options (--tls, --tls-ca, --tls-cert, --tls-key, --tls-server-name),
logging and the JSON configuration file (--config).

Developed by BURN-LAB: https://burn-lab.ru
`

// printUsage writes the top-level help to w.
func printUsage(w io.Writer) {
	fmt.Fprint(w, usageText)
}
