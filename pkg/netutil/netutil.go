package netutil

import "net"

// ParseSubnet parses a CIDR string (e.g. "192.168.1.0/24") or a single IP address string (e.g. "192.168.1.1")
// into a *net.IPNet subnet representation.
func ParseSubnet(s string) (*net.IPNet, error) {
	_, subnet, err := net.ParseCIDR(s)
	if err == nil {
		return subnet, nil
	}
	ip := net.ParseIP(s)
	if ip != nil {
		if ip.To4() != nil {
			_, subnet, err = net.ParseCIDR(s + "/32")
			return subnet, err
		}
		_, subnet, err = net.ParseCIDR(s + "/128")
		return subnet, err
	}
	return nil, err
}
