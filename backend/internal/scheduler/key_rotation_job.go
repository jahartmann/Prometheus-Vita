package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/antigravity/prometheus/internal/service/sshkeys"
)

type KeyRotationJob struct {
	sshkeySvc *sshkeys.Service
	interval  time.Duration
}

func NewKeyRotationJob(sshkeySvc *sshkeys.Service, interval time.Duration) *KeyRotationJob {
	return &KeyRotationJob{
		sshkeySvc: sshkeySvc,
		interval:  interval,
	}
}

func (j *KeyRotationJob) Name() string {
	return "key_rotation"
}

func (j *KeyRotationJob) Interval() time.Duration {
	return j.interval
}

func (j *KeyRotationJob) Run(ctx context.Context) error {
	// Check for due rotations
	dueSchedules, err := j.sshkeySvc.ListDueRotations(ctx)
	if err != nil {
		slog.Error("failed to list due rotations", slog.Any("error", err))
		return nil
	}

	rotated := 0
	for _, sched := range dueSchedules {
		newKey, err := j.sshkeySvc.RotateKey(ctx, sched.NodeID)
		if err != nil {
			slog.Warn("key rotation failed",
				slog.String("node_id", sched.NodeID.String()),
				slog.Any("error", err),
			)
			continue
		}

		// Update schedule
		now := time.Now().UTC()
		nextRotation := now.Add(time.Duration(sched.IntervalDays) * 24 * time.Hour)
		sched.LastRotatedAt = &now
		sched.NextRotationAt = &nextRotation
		_ = j.sshkeySvc.UpdateRotationSchedule(ctx, &sched)

		slog.Info("key rotated via schedule",
			slog.String("node_id", sched.NodeID.String()),
			slog.String("new_key_id", newKey.ID.String()),
		)
		rotated++
	}

	// Check for expiring keys (within 7 days)
	expiringKeys, err := j.sshkeySvc.GetExpiringSoon(ctx, time.Now().UTC().Add(7*24*time.Hour))
	if err != nil {
		slog.Error("failed to check expiring keys", slog.Any("error", err))
		return nil
	}

	if len(expiringKeys) > 0 {
		slog.Warn("SSH keys expiring soon",
			slog.Int("count", len(expiringKeys)),
		)
	}

	if rotated > 0 || len(expiringKeys) > 0 {
		slog.Info("key rotation job completed",
			slog.Int("rotated", rotated),
			slog.Int("expiring_soon", len(expiringKeys)),
		)
	}

	return nil
}
