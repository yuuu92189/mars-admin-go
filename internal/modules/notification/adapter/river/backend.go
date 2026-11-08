package river

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"mars-admin-go/internal/modules/integrations"
	messagingsqlc "mars-admin-go/internal/modules/messaging/adapter/postgres/sqlc"
	"mars-admin-go/internal/platform/apperror"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
)

const queueName = "notifications"

type DispatchArgs struct {
	NoticeID uuid.UUID `json:"noticeId" river:"unique"`
	Channel  string    `json:"channel" river:"unique"`
}

func (DispatchArgs) Kind() string { return "notification_dispatch" }

func (DispatchArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: queueName, MaxAttempts: 5}
}

type Backend struct {
	client *river.Client[pgx.Tx]
	logger *slog.Logger
}

type Options struct {
	SMS     integrations.SMSProvider
	Push    integrations.PushProvider
	Mail    integrations.MailProvider
	MaxJobs int
}

func NewEnqueuer(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) (*Backend, error) {
	if err := ensureSchema(ctx, pool); err != nil {
		return nil, err
	}
	return newClient(pool, logger, Options{})
}

func NewBackend(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger, options Options) (*Backend, error) {
	if err := ensureSchema(ctx, pool); err != nil {
		return nil, err
	}
	backend, err := newClient(pool, logger, options)
	if err != nil {
		return nil, err
	}
	if err := backend.client.Start(ctx); err != nil {
		return nil, fmt.Errorf("start notification worker: %w", err)
	}
	return backend, nil
}

func ensureSchema(ctx context.Context, pool *pgxpool.Pool) error {
	migrator, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		return fmt.Errorf("create River migrator: %w", err)
	}
	if _, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil); err != nil {
		return fmt.Errorf("migrate River schema: %w", err)
	}
	return nil
}

func newClient(pool *pgxpool.Pool, logger *slog.Logger, options Options) (*Backend, error) {
	workers := river.NewWorkers()
	river.AddWorker(workers, &dispatchWorker{
		pool: pool, sms: options.SMS, push: options.Push, mail: options.Mail, logger: logger,
	})
	maxJobs := options.MaxJobs
	if maxJobs <= 0 {
		maxJobs = 4
	}
	client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Logger: logger, Workers: workers,
		Queues:     map[string]river.QueueConfig{queueName: {MaxWorkers: maxJobs}},
		JobTimeout: 90 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("create notification River client: %w", err)
	}
	return &Backend{client: client, logger: logger}, nil
}

func (b *Backend) Start(ctx context.Context) error {
	return b.client.Start(ctx)
}

func (b *Backend) Stop(ctx context.Context) error {
	return b.client.Stop(ctx)
}

func (b *Backend) Enqueue(ctx context.Context, noticeID uuid.UUID, channel string) (int64, error) {
	result, err := b.client.Insert(ctx, DispatchArgs{NoticeID: noticeID, Channel: channel}, &river.InsertOpts{
		Queue: queueName, MaxAttempts: 5,
		UniqueOpts: river.UniqueOpts{ByArgs: true},
	})
	if err != nil {
		return 0, err
	}
	return result.Job.ID, nil
}

type dispatchWorker struct {
	river.WorkerDefaults[DispatchArgs]
	pool   *pgxpool.Pool
	sms    integrations.SMSProvider
	push   integrations.PushProvider
	mail   integrations.MailProvider
	logger *slog.Logger
}

func (w *dispatchWorker) Timeout(*river.Job[DispatchArgs]) time.Duration {
	return 75 * time.Second
}

func (w *dispatchWorker) Work(ctx context.Context, job *river.Job[DispatchArgs]) error {
	q := messagingsqlc.New(w.pool)
	notice, err := q.GetNotice(ctx, pgUUID(job.Args.NoticeID))
	if err != nil {
		return err
	}
	contacts, err := q.ListNoticeRecipientContacts(ctx, pgUUID(job.Args.NoticeID))
	if err != nil {
		return err
	}
	targetCount := int32(len(contacts)) // #nosec G115 -- bounded by notice recipient fan-out.
	successCount := int32(0)
	var sendErr error
	switch job.Args.Channel {
	case "sms":
		if w.sms == nil {
			sendErr = apperror.New(apperror.KindNotConfigured, "sms_not_configured", "SMS notification provider is not configured.")
			break
		}
		if targetCount == 0 {
			sendErr = errors.New("notice has no active recipients")
			break
		}
		successCount, sendErr = sendSMSCampaign(ctx, w.sms, contacts, notice.Title, notice.Content)
	case "push", "dingtalk", "feishu", "wechat_work":
		if w.push == nil {
			sendErr = apperror.New(apperror.KindNotConfigured, "push_not_configured", "Push notification provider is not configured.")
			break
		}
		if targetCount == 0 {
			sendErr = errors.New("notice has no active recipients")
			break
		}
		if err := w.push.Send(ctx, "*", notice.Title, notice.Content); err != nil {
			sendErr = err
		} else {
			successCount = 1
		}
	case "email":
		if w.mail == nil {
			sendErr = apperror.New(apperror.KindNotConfigured, "mail_not_configured", "Mail notification provider is not configured.")
			break
		}
		if targetCount == 0 {
			sendErr = errors.New("notice has no active recipients")
			break
		}
		for _, contact := range contacts {
			if !contact.Email.Valid || contact.Email.String == "" {
				continue
			}
			if err := w.mail.Send(ctx, contact.Email.String, notice.Title, notice.Content); err != nil {
				sendErr = err
				break
			}
			successCount++
		}
	default:
		sendErr = fmt.Errorf("unsupported notification channel %q", job.Args.Channel)
	}
	status := "success"
	errorMessage := ""
	if sendErr != nil {
		status = "failed"
		errorMessage = sendErr.Error()
		if len(errorMessage) > 500 {
			errorMessage = errorMessage[:500]
		}
	}
	if err := q.InsertNoticeSendLog(ctx, messagingsqlc.InsertNoticeSendLogParams{
		ID: pgUUID(mustID()), NoticeID: pgUUID(job.Args.NoticeID), Channel: job.Args.Channel,
		Status: status, TargetCount: targetCount, SuccessCount: successCount, ErrorMessage: errorMessage,
	}); err != nil {
		w.logger.ErrorContext(ctx, "persist notification delivery log", "error", err)
	}
	if sendErr != nil {
		w.logger.WarnContext(ctx, "notification delivery failed",
			"notice_id", job.Args.NoticeID, "channel", job.Args.Channel, "error", sendErr)
	}
	return sendErr
}

func sendSMSCampaign(
	ctx context.Context,
	provider integrations.SMSProvider,
	contacts []messagingsqlc.ListNoticeRecipientContactsRow,
	title, content string,
) (int32, error) {
	successCount := int32(0)
	for _, contact := range contacts {
		if !contact.Phone.Valid || contact.Phone.String == "" {
			continue
		}
		if err := provider.SendNotice(ctx, contact.Phone.String, "", map[string]string{
			"title": title, "content": content,
		}); err != nil {
			return successCount, err
		}
		successCount++
	}
	return successCount, nil
}

func pgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte(id), Valid: id != uuid.Nil}
}

func mustID() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.New()
	}
	return id
}
