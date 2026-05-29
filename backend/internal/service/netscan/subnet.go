package netscan

import (
	"context"
	"fmt"
	"net"
	"strings"
)

// minSubnetPrefix bounds subnet discovery to networks of at most ~1024 hosts
// (/22) so we never launch an enormous sweep across a large/misconfigured range.
const minSubnetPrefix = 22

// parseSubnetFromIPAddr extracts the first usable IPv4 CIDR from `ip -o -f inet
// addr show scope global` output, skipping ranges larger than minSubnetPrefix.
func parseSubnetFromIPAddr(output string) (string, error) {
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		for i := 0; i+1 < len(fields); i++ {
			if fields[i] != "inet" {
				continue
			}
			cidr := fields[i+1]
			ip, ipnet, err := net.ParseCIDR(cidr)
			if err != nil || ip.To4() == nil {
				continue
			}
			if ones, _ := ipnet.Mask.Size(); ones >= minSubnetPrefix {
				return cidr, nil
			}
		}
	}
	return "", fmt.Errorf("netscan: no suitable IPv4 subnet found")
}

// deriveSubnet asks the node for its primary global IPv4 subnet so subnet
// discovery can sweep the LAN the node lives on without per-node configuration.
func deriveSubnet(ctx context.Context, runner SSHRunner) (string, error) {
	out, exitCode, err := runner.Run(ctx, "ip -o -f inet addr show scope global")
	if err != nil {
		return "", fmt.Errorf("netscan: derive subnet: %w", err)
	}
	if exitCode != 0 {
		return "", fmt.Errorf("netscan: derive subnet: ip exited %d", exitCode)
	}
	return parseSubnetFromIPAddr(out)
}
