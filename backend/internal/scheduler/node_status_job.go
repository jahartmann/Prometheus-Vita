package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/antigravity/prometheus/internal/proxmox"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/monitor"
	"github.com/redis/go-redis/v9"
)

// NotificationSender is an optional interface for sending notifications on status changes.
type NotificationSender interface {
	Notify(ctx context.Context, eventType, subject, body string)
}

type NodeStatusJob struct {
	nodeRepo      repository.NodeRepository
	clientFactory proxmox.ClientFactory
	redis         *redis.Client
	wsHub         *monitor.WSHub
	interval      time.Duration
	notifSvc      NotificationSender
}

// SetNotificationService sets an optional notification service for node status events.
func (j *NodeStatusJob) SetNotificationService(svc NotificationSender) {
	j.notifSvc = svc
}

func NewNodeStatusJob(
	nodeRepo repository.NodeRepository,
	clientFactory proxmox.ClientFactory,
	redisClient *redis.Client,
	wsHub *monitor.WSHub,
	interval time.Duration,
) *NodeStatusJob {
	return &NodeStatusJob{
		nodeRepo:      nodeRepo,
		clientFactory: clientFactory,
		redis:         redisClient,
		wsHub:         wsHub,
		interval:      interval,
	}
}

func (j *NodeStatusJob) Name() string {
	return "node_status"
}

func (j *NodeStatusJob) Interval() time.Duration {
	return j.interval
}

func (j *NodeStatusJob) Run(ctx context.Context) error {
	nodes, err := j.nodeRepo.List(ctx)
	if err != nil {
		return fmt.Errorf("list nodes: %w", err)
	}

	for _, n := range nodes {
		node := n
		client, err := j.clientFactory.CreateClient(&node)
		if err != nil {
			slog.Warn("failed to create proxmox client",
				slog.String("node", node.Name),
				slog.Any("error", err),
			)
			_ = j.nodeRepo.UpdateStatus(ctx, node.ID, false)
			continue
		}

		pveNodes, err := client.GetNodes(ctx)
		if err != nil {
			slog.Warn("node unreachable",
				slog.String("node", node.Name),
				slog.Any("error", err),
			)
			wasOnline := node.IsOnline
			_ = j.nodeRepo.UpdateStatus(ctx, node.ID, false)
			j.cacheStatus(ctx, node.ID.String(), nil)
			j.wsHub.BroadcastMessage(monitor.WSMessage{
				Type: "node_status",
				Data: map[string]any{
					"node_id":   node.ID,
					"is_online": false,
					"status":    nil,
				},
			})
			if wasOnline && j.notifSvc != nil {
				j.notifSvc.Notify(ctx, "node_offline",
					fmt.Sprintf("Node offline: %s", node.Name),
					fmt.Sprintf("Node %s (%s) is no longer reachable.", node.Name, node.Hostname),
				)
			}
			continue
		}

		if !node.IsOnline && j.notifSvc != nil {
			j.notifSvc.Notify(ctx, "node_online",
				fmt.Sprintf("Node online: %s", node.Name),
				fmt.Sprintf("Node %s (%s) is back online.", node.Name, node.Hostname),
			)
		}
		_ = j.nodeRepo.UpdateStatus(ctx, node.ID, true)

		if len(pveNodes) > 0 {
			status, err := client.GetNodeStatus(ctx, pveNodes[0])
			if err != nil {
				slog.Warn("failed to get node status",
					slog.String("node", node.Name),
					slog.Any("error", err),
				)
				continue
			}
			j.cacheStatus(ctx, node.ID.String(), status)
			j.wsHub.BroadcastMessage(monitor.WSMessage{
				Type: "node_status",
				Data: map[string]any{
					"node_id":   node.ID,
					"is_online": true,
					"status":    status,
				},
			})
		}
	}

	return nil
}

func (j *NodeStatusJob) cacheStatus(ctx context.Context, nodeID string, status *proxmox.NodeStatus) {
	key := fmt.Sprintf("node:status:%s", nodeID)

	if status == nil {
		j.redis.Del(ctx, key)
		return
	}

	data, err := json.Marshal(status)
	if err != nil {
		slog.Warn("failed to marshal node status", slog.Any("error", err))
		return
	}

	j.redis.Set(ctx, key, data, 2*j.interval)
}
