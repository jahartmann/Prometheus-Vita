package scheduler

import (
	"context"

	"github.com/antigravity/prometheus/internal/proxmox"
)

// latestNetRates returns the most recent node network in/out rates (bytes/sec)
// from RRD data, skipping a trailing empty/incomplete bucket. The Proxmox node
// /status endpoint does not expose netin/netout, so RRD is the reliable source
// for node throughput.
func latestNetRates(points []proxmox.RRDDataPoint) (int64, int64) {
	for i := len(points) - 1; i >= 0; i-- {
		if points[i].NetIn > 0 || points[i].NetOut > 0 {
			return int64(points[i].NetIn), int64(points[i].NetOut)
		}
	}
	if len(points) > 0 {
		last := points[len(points)-1]
		return int64(last.NetIn), int64(last.NetOut)
	}
	return 0, 0
}

// nodeNetRatesFromRRD fetches the node's recent RRD data and returns its current
// network in/out rates. Returns 0,0 on error so metrics collection continues.
func nodeNetRatesFromRRD(ctx context.Context, client *proxmox.Client, pveNode string) (int64, int64) {
	points, err := client.GetNodeRRDData(ctx, pveNode, "hour")
	if err != nil {
		return 0, 0
	}
	return latestNetRates(points)
}
