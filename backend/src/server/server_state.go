package server

import "net"

// ServerState holds the server state information
type ServerState struct {
Address  string
Listener net.Listener
ID       string
}
