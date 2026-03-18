package netscan

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
)

var (
	cidrRegex        = regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}/\d{1,2}$`)
	ipRegex          = regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`)
	containerIDRegex = regexp.MustCompile(`^\d+$`)
)

// ValidateCIDR ensures the string is a valid CIDR notation (e.g. 192.168.1.0/24).
// It prevents command injection by rejecting anything not matching the strict regex
// before passing the value to net.ParseCIDR for semantic validation.
func ValidateCIDR(s string) error {
	if !cidrRegex.MatchString(s) {
		return fmt.Errorf("invalid CIDR: %s", s)
	}
	_, _, err := net.ParseCIDR(s)
	return err
}

// ValidateIP ensures the string is a valid dotted-quad IPv4 address.
func ValidateIP(s string) error {
	if !ipRegex.MatchString(s) {
		return fmt.Errorf("invalid IP: %s", s)
	}
	parts := strings.Split(s, ".")
	for _, p := range parts {
		n, _ := strconv.Atoi(p)
		if n < 0 || n > 255 {
			return fmt.Errorf("IP octet out of range: %s", p)
		}
	}
	return nil
}

// ValidateContainerID ensures the string is a numeric container/VM ID only.
// This prevents shell injection when interpolating into SSH commands.
func ValidateContainerID(s string) error {
	if !containerIDRegex.MatchString(s) {
		return fmt.Errorf("invalid container ID: %s", s)
	}
	return nil
}
