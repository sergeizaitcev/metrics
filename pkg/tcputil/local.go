package tcputil

import (
	"net"
	"strconv"
)

// Local возвращает локальный IP-адрес.
func Local() net.IP {
	addrs, _ := net.InterfaceAddrs()
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP
			}
		}
	}
	return nil
}

// FreePort запрашивает у ядра свободный открытый порт, готовый
// к использованию.
func FreePort() (port string, err error) {
	a, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return "", err
	}

	l, err := net.ListenTCP("tcp", a)
	if err != nil {
		return "", err
	}

	p := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()

	return strconv.FormatInt(int64(p), 10), nil
}
