package utils

import (
	"net"
	"strconv"
)

// JoinHostPort is a type-safe wrapper over net.JoinHostPort.
func JoinHostPort(host string, port int) string {
	return net.JoinHostPort(host, strconv.Itoa(port))
}
