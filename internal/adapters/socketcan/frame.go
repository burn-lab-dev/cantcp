//go:build linux

package socketcan

// maxFrameLen is sizeof(struct canfd_frame): the largest frame a SocketCAN
// raw socket returns and accepts.
const maxFrameLen = 72
