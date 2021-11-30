//go:generate mockgen -source=$GOFILE -destination=./mock/mock_$GOFILE -package=mock
package udp

import (
	"net"
)

type Connection interface {
	net.Conn
	ReadFromUDP(b []byte) (n int, addr *net.UDPAddr, err error)
}

type Connector interface {
	ResolveUDPAddr(network string, address string) (*net.UDPAddr, error)
	DialUDP(network string, laddr *net.UDPAddr, raddr *net.UDPAddr) (Connection, error)
	ListenUDP(network string, laddr *net.UDPAddr) (Connection, error)
}

type connector struct {
}

func NewConnector() Connector {
	return &connector{}
}

func (connector) ResolveUDPAddr(network string, address string) (*net.UDPAddr, error) {
	return net.ResolveUDPAddr(network, address)
}

func (connector) DialUDP(network string, laddr *net.UDPAddr, raddr *net.UDPAddr) (Connection, error) {
	return net.DialUDP(network, laddr, raddr)
}

func (connector) ListenUDP(network string, laddr *net.UDPAddr) (Connection, error) {
	return net.ListenUDP(network, laddr)
}
