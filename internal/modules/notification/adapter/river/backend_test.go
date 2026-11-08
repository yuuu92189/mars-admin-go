package river

import (
	"context"
	"errors"
	"testing"
	"time"

	messagingsqlc "mars-admin-go/internal/modules/messaging/adapter/postgres/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
)

type timeoutSMSProvider struct{}

func (timeoutSMSProvider) SendCode(context.Context, string, string) error {
	return errors.New("not used")
}

func (timeoutSMSProvider) SendNotice(ctx context.Context, _ string, _ string, _ map[string]string) error {
	<-ctx.Done()
	return ctx.Err()
}

func TestSMSCampaignPropagatesProviderTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	contacts := []messagingsqlc.ListNoticeRecipientContactsRow{
		{Phone: pgtype.Text{String: "13800000000", Valid: true}},
	}
	count, err := sendSMSCampaign(ctx, timeoutSMSProvider{}, contacts, "title", "content")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want deadline exceeded", err)
	}
	if count != 0 {
		t.Fatalf("success count = %d, want 0", count)
	}
}
