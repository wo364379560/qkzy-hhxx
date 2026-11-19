package ratelimit

import "time"

var (
	CreateGroupRules = []Rule{
		{Scope: "create_group:minute", Limit: 20, Window: time.Minute},
		{Scope: "create_group:day", Limit: 200, Window: 24 * time.Hour},
	}
	InviteJoinGroupRules = []Rule{
		{Scope: "invite_join_group:minute", Limit: 60, Window: time.Minute},
		{Scope: "invite_join_group:day", Limit: 2000, Window: 24 * time.Hour},
	}
	UpdateProfileRules = []Rule{
		{Scope: "update_profile:hour", Limit: 30, Window: time.Hour},
	}
	RefreshTokenRules = []Rule{
		{Scope: "refresh_token:minute", Limit: 60, Window: time.Minute},
	}
	PasswordWorkRules = []Rule{
		{Scope: "password_work:user:minute", Limit: 3, Window: time.Minute},
		{Scope: "password_work:user:hour", Limit: 10, Window: time.Hour},
	}
	AdminPasswordWorkRules = []Rule{
		{Scope: "password_work:admin:minute", Limit: 3, Window: time.Minute},
		{Scope: "password_work:admin:hour", Limit: 10, Window: time.Hour},
	}

	// Public authentication rules are deliberately multi-dimensional. The
	// endpoint-global rule caps a distributed botnet, while IP/account/device
	// rules contain a single source without relying on a spoofable device ID.
	PublicLoginGlobalRules = []Rule{
		{Scope: "public_login:global:second", Limit: 15, Window: time.Second},
		{Scope: "public_login:global:minute", Limit: 600, Window: time.Minute},
	}
	PublicLoginSharedGlobalRules = []Rule{
		{Scope: "public_login:shared:second", Limit: 25, Window: time.Second},
		{Scope: "public_login:shared:minute", Limit: 900, Window: time.Minute},
	}
	PublicLoginAccountRules = []Rule{
		{Scope: "public_login:account:10minute", Limit: 10, Window: 10 * time.Minute},
	}
	PublicLoginDeviceRules = []Rule{
		{Scope: "public_login:device:10minute", Limit: 20, Window: 10 * time.Minute},
	}

	PublicRegisterGlobalRules = []Rule{
		{Scope: "public_register:global:second", Limit: 5, Window: time.Second},
		{Scope: "public_register:global:minute", Limit: 120, Window: time.Minute},
	}
	PublicRegisterIPRules = []Rule{
		{Scope: "public_register:ip:hour", Limit: 50, Window: time.Hour},
	}
	PublicRegisterPhoneRules = []Rule{
		{Scope: "public_register:phone:hour", Limit: 5, Window: time.Hour},
	}
	PublicRegisterDeviceRules = []Rule{
		{Scope: "public_register:device:day", Limit: 5, Window: 24 * time.Hour},
	}

	PublicSMSGlobalRules = []Rule{
		{Scope: "public_sms:global:second", Limit: 3, Window: time.Second},
		{Scope: "public_sms:global:minute", Limit: 30, Window: time.Minute},
		{Scope: "public_sms:global:hour", Limit: 600, Window: time.Hour},
	}
	PublicSMSIPRules = []Rule{
		{Scope: "public_sms:ip:10minute", Limit: 10, Window: 10 * time.Minute},
		{Scope: "public_sms:ip:hour", Limit: 30, Window: time.Hour},
	}
	PublicSMSPhoneRules = []Rule{
		{Scope: "public_sms:phone:minute", Limit: 1, Window: time.Minute},
		{Scope: "public_sms:phone:10minute", Limit: 3, Window: 10 * time.Minute},
		{Scope: "public_sms:phone:day", Limit: 10, Window: 24 * time.Hour},
	}
	PublicSMSDeviceRules = []Rule{
		{Scope: "public_sms:device:hour", Limit: 10, Window: time.Hour},
	}

	PublicPasswordResetGlobalRules = []Rule{
		{Scope: "public_password_reset:global:second", Limit: 3, Window: time.Second},
		{Scope: "public_password_reset:global:minute", Limit: 60, Window: time.Minute},
	}
	PublicPasswordResetIPRules = []Rule{
		{Scope: "public_password_reset:ip:10minute", Limit: 10, Window: 10 * time.Minute},
	}
	PublicPasswordResetPhoneRules = []Rule{
		{Scope: "public_password_reset:phone:hour", Limit: 5, Window: time.Hour},
	}
	PublicPasswordResetDeviceRules = []Rule{
		{Scope: "public_password_reset:device:hour", Limit: 10, Window: time.Hour},
	}

	PublicRefreshGlobalRules = []Rule{
		{Scope: "public_refresh:global:second", Limit: 100, Window: time.Second},
		{Scope: "public_refresh:global:minute", Limit: 3000, Window: time.Minute},
	}
	PublicRefreshSharedGlobalRules = []Rule{
		{Scope: "public_refresh:shared:second", Limit: 150, Window: time.Second},
		{Scope: "public_refresh:shared:minute", Limit: 4500, Window: time.Minute},
	}
	PublicRefreshIPRules = []Rule{
		{Scope: "public_refresh:ip:minute", Limit: 300, Window: time.Minute},
	}
	PublicRefreshTokenRules = []Rule{
		{Scope: "public_refresh:token:minute", Limit: 60, Window: time.Minute},
	}
	PublicRefreshDeviceRules = []Rule{
		{Scope: "public_refresh:device:minute", Limit: 180, Window: time.Minute},
	}

	PublicVerifyGlobalRules = []Rule{
		{Scope: "public_verify:global:second", Limit: 10, Window: time.Second},
		{Scope: "public_verify:global:minute", Limit: 300, Window: time.Minute},
	}
	PublicVerifyIPRules = []Rule{
		{Scope: "public_verify:ip:minute", Limit: 30, Window: time.Minute},
	}
	PublicChallengeGlobalRules = []Rule{
		{Scope: "public_challenge:global:second", Limit: 20, Window: time.Second},
		{Scope: "public_challenge:global:minute", Limit: 600, Window: time.Minute},
	}
	PublicChallengeIPRules = []Rule{
		{Scope: "public_challenge:ip:minute", Limit: 60, Window: time.Minute},
	}
	PublicChallengeStatusGlobalRules = []Rule{
		{Scope: "public_challenge_status:global:second", Limit: 100, Window: time.Second},
		{Scope: "public_challenge_status:global:minute", Limit: 3000, Window: time.Minute},
	}
	PublicChallengeStatusIPRules = []Rule{
		{Scope: "public_challenge_status:ip:minute", Limit: 180, Window: time.Minute},
	}
	PublicChallengeStatusSubjectRules = []Rule{
		{Scope: "public_challenge_status:subject:minute", Limit: 60, Window: time.Minute},
	}
	PublicChallengeSMSRules = []Rule{
		{Scope: "public_challenge_sms:subject:10minute", Limit: 5, Window: 10 * time.Minute},
	}
	PublicChallengeExchangeRules = []Rule{
		{Scope: "public_challenge_exchange:subject:minute", Limit: 20, Window: time.Minute},
	}
	PublicBootstrapGlobalRules = []Rule{
		{Scope: "public_bootstrap:global:second", Limit: 100, Window: time.Second},
		{Scope: "public_bootstrap:global:minute", Limit: 3000, Window: time.Minute},
	}
	PublicBootstrapIPRules = []Rule{
		{Scope: "public_bootstrap:ip:minute", Limit: 120, Window: time.Minute},
	}
	PublicWebhookGlobalRules = []Rule{
		{Scope: "public_webhook:global:second", Limit: 200, Window: time.Second},
		{Scope: "public_webhook:global:minute", Limit: 6000, Window: time.Minute},
	}
	PublicDownloadGlobalRules = []Rule{
		{Scope: "public_download:global:second", Limit: 100, Window: time.Second},
		{Scope: "public_download:global:minute", Limit: 3000, Window: time.Minute},
	}
	PublicDownloadIPRules = []Rule{
		{Scope: "public_download:ip:minute", Limit: 120, Window: time.Minute},
	}
	PublicDownloadFileRules = []Rule{
		{Scope: "public_download:file:minute", Limit: 300, Window: time.Minute},
	}
	PublicAdminLoginGlobalRules = []Rule{
		{Scope: "public_admin_auth:global:second", Limit: 2, Window: time.Second},
		{Scope: "public_admin_auth:global:minute", Limit: 30, Window: time.Minute},
	}
	PublicAdminAuthSharedGlobalRules = []Rule{
		{Scope: "public_admin_auth:shared:second", Limit: 4, Window: time.Second},
		{Scope: "public_admin_auth:shared:minute", Limit: 60, Window: time.Minute},
	}
	PublicAdminLoginIPRules = []Rule{
		{Scope: "public_admin_auth:ip:10minute", Limit: 10, Window: 10 * time.Minute},
	}
	PublicAdminLoginAccountRules = []Rule{
		{Scope: "public_admin_auth:account:10minute", Limit: 5, Window: 10 * time.Minute},
	}
	QueryBatchRules = []Rule{
		{Scope: "query_batch:minute", Limit: 120, Window: time.Minute},
	}
	LastOnlineRules = []Rule{
		{Scope: "last_online:minute", Limit: 120, Window: time.Minute},
	}
)
