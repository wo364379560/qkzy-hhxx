package messagelimit

import (
	"context"
	"github.com/ixayda/ix-Go/internal/app/server/model"
	"testing"
)

// Real users must be able to pass the old sixth-message boundary without a
// database or Redis lookup. Existing quota rows must not participate in sending.
func TestRetiredQuotaAllowsRepeatedSendsWithoutStorage(t *testing.T) {
	for _, sender := range []*model.User{nil, {}, {ID: 1074, IMUserID: "unverified-user"}} {
		for n := 1; n <= 100; n++ {
			decision, err := CheckBeforeSend(context.Background(), sender)
			if err != nil || decision.Blocked || decision.Limit != 0 || decision.Count != 0 || decision.Reason != "" {
				t.Fatalf("message %d: decision=%+v err=%v", n, decision, err)
			}
			if err := RecordSuccessfulSend(context.Background(), sender); err != nil {
				t.Fatal(err)
			}
		}
	}
}
