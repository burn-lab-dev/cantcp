package tcp

import "net"

// countingConn wraps a connection and reports the number of bytes read and
// written to the callbacks. The callbacks are called on the connection
// goroutines only.
type countingConn struct {
	net.Conn
	onRead  func(int)
	onWrite func(int)
}

// Read reads from the connection and reports the byte count.
func (c *countingConn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	if n > 0 {
		c.onRead(n)
	}
	return n, err
}

// Write writes to the connection and reports the byte count.
func (c *countingConn) Write(p []byte) (int, error) {
	n, err := c.Conn.Write(p)
	if n > 0 {
		c.onWrite(n)
	}
	return n, err
}
