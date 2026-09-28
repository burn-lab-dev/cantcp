package services

// maxFrameLen is sizeof(struct canfd_frame): the largest raw frame a
// SocketCAN bus produces and the cantcp protocol carries.
const maxFrameLen = 72
