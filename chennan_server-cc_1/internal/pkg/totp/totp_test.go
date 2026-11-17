package totp

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp"
	otptotp "github.com/pquerna/otp/totp"
)

func TestEnrollmentURIKeepsSecretForUnicodeAccounts(t *testing.T) {
	for _, account := range []string{"helper123", "一路向南", "测试 管理员", "助理+中文@example.com"} {
		t.Run(account, func(t *testing.T) {
			secret, uri, err := GenerateTOTPSecret(account)
			if err != nil {
				t.Fatal(err)
			}
			parsed, err := otp.NewKeyFromURL(uri)
			if err != nil {
				t.Fatal(err)
			}
			if parsed.Secret() != secret || parsed.AccountName() != account || parsed.Issuer() != "TxIM" {
				t.Fatal("enrollment URI changed the account or secret")
			}
			for _, c := range uri {
				if c > 127 {
					t.Fatal("URI must percent-encode Unicode labels")
				}
			}
			u, _ := url.Parse(uri)
			if strings.Contains(u.RawQuery, account) {
				t.Fatal("account leaked into OTP parameters")
			}
			code, err := otptotp.GenerateCode(parsed.Secret(), time.Now())
			if err != nil || !ValidateTOTP(secret, code) {
				t.Fatal("code from enrollment URI did not validate")
			}
		})
	}
}
