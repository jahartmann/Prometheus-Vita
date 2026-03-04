package scheduler

import (
	"context"
	"time"

	"github.com/antigravity/prometheus/internal/service/telegram"
)

type TelegramPollJob struct {
	botSvc   *telegram.BotService
	interval time.Duration
}

func NewTelegramPollJob(botSvc *telegram.BotService, interval time.Duration) *TelegramPollJob {
	return &TelegramPollJob{
		botSvc:   botSvc,
		interval: interval,
	}
}

func (j *TelegramPollJob) Name() string {
	return "telegram_poll"
}

func (j *TelegramPollJob) Interval() time.Duration {
	return j.interval
}

func (j *TelegramPollJob) Run(ctx context.Context) error {
	return j.botSvc.PollUpdates(ctx)
}
