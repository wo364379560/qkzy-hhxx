package ratelimit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"github.com/ixayda/ix-Go/pkg/logger"
)

const DefaultMessage = "操作过于频繁，请稍后再试"

var ErrLimited = errors.New("rate limited")

const redisRateLimitTimeout = 100 * time.Millisecond

const atomicIncrementScript = `
local count = redis.call("INCRBY", KEYS[1], ARGV[1])
if redis.call("PTTL", KEYS[1]) < 0 then
  redis.call("PEXPIRE", KEYS[1], ARGV[2])
end
return count
`

var (
	redisWarningLogSampler = newKeyedLogSampler(time.Minute, maxSecurityLogSampleKeys)
	limitedLogSampler      = newKeyedLogSampler(10*time.Second, maxSecurityLogSampleKeys)
	overloadLogSampler     = newKeyedLogSampler(10*time.Second, maxSecurityLogSampleKeys)
)

type Rule struct {
	Scope  string
	Limit  int64
	Window time.Duration
}

type Decision struct {
	Allowed               bool
	Rule                  Rule
	Count                 int64
	Limit                 int64
	LocalSubjectUntracked bool
}

type SubjectRules struct {
	Subject string
	Rules   []Rule
}

func Check(ctx context.Context, subject string, rules []Rule) (Decision, error) {
	return CheckWithCost(ctx, subject, 1, rules)
}

func CheckWithCost(ctx context.Context, subject string, cost int64, rules []Rule) (Decision, error) {
	if cost <= 0 {
		cost = 1
	}
	subject = normalizeSubject(subject)
	if subject == "" || len(rules) == 0 {
		return Decision{Allowed: true}, nil
	}

	validRules := make([]Rule, 0, len(rules))
	for _, rule := range rules {
		if rule.Limit <= 0 || rule.Window <= 0 || strings.TrimSpace(rule.Scope) == "" {
			continue
		}
		decision := processLimiter.check(subject, cost, rule, time.Now())
		if !decision.Allowed {
			return decision, ErrLimited
		}
		if decision.LocalSubjectUntracked {
			logLocalCapacityWarning(rule.Scope)
		}
		validRules = append(validRules, rule)
	}

	redisDS := datasource.Datasource.Redis
	if redisDS == nil || !redisDS.Enabled() || len(validRules) == 0 {
		return Decision{Allowed: true}, nil
	}
	client := redisDS.GetClient()
	redisCtx, cancel := context.WithTimeout(ctx, redisRateLimitTimeout)
	defer cancel()
	for _, rule := range validRules {
		key := fmt.Sprintf("rate_limit:%s:%s", sanitize(rule.Scope), subject)
		ttlMillis := rule.Window.Milliseconds()
		if ttlMillis < 1 {
			ttlMillis = 1
		}
		count, err := client.Eval(redisCtx, atomicIncrementScript, []string{key}, cost, ttlMillis).Int64()
		if err != nil {
			logRedisWarning(rule.Scope, err)
			// The bounded process limiter above remains active. Redis outages must
			// not turn public authentication endpoints into an unprotected path.
			return Decision{Allowed: true}, nil
		}
		if count > rule.Limit {
			return Decision{Allowed: false, Rule: rule, Count: count, Limit: rule.Limit}, ErrLimited
		}
	}
	return Decision{Allowed: true}, nil
}

func CheckSubjects(ctx context.Context, subjects []SubjectRules) (Decision, error) {
	for _, item := range subjects {
		if strings.TrimSpace(item.Subject) == "" || len(item.Rules) == 0 {
			continue
		}
		decision, err := Check(ctx, item.Subject, item.Rules)
		if err != nil || !decision.Allowed {
			return decision, err
		}
	}
	return Decision{Allowed: true}, nil
}

func HashSubject(prefix, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(value))
	return sanitize(prefix) + ":" + hex.EncodeToString(sum[:])[:24]
}

func normalizeSubject(subject string) string {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return ""
	}
	subject = sanitize(subject)
	if len(subject) <= 192 {
		return subject
	}
	sum := sha256.Sum256([]byte(subject))
	return "oversized_" + hex.EncodeToString(sum[:])[:24]
}

func sanitize(value string) string {
	value = strings.TrimSpace(value)
	replacer := strings.NewReplacer(" ", "_", "\t", "_", "\n", "_", "\r", "_", ":", "_")
	return replacer.Replace(value)
}

func LogLimited(action, subject, ip string, decision Decision) {
	emit, suppressed := limitedLogSampler.sample(
		limitedLogSampleKey(action, decision.Rule.Scope),
		time.Now(),
	)
	if !emit {
		return
	}
	logger.Warnf("rate limit exceeded: action=%s subject=%s ip=%s scope=%s count=%d limit=%d window=%s suppressed=%d",
		action,
		subject,
		ip,
		decision.Rule.Scope,
		decision.Count,
		decision.Limit,
		decision.Rule.Window,
		suppressed,
	)
}

func LogOverload(action, reason, ip string) {
	emit, suppressed := overloadLogSampler.sample(
		overloadLogSampleKey(action, reason),
		time.Now(),
	)
	if !emit {
		return
	}
	logger.Warnf("request shed by protection gate: action=%s reason=%s ip=%s suppressed=%d",
		action, reason, ip, suppressed)
}

func logRedisWarning(scope string, err error) {
	emit, suppressed := redisWarningLogSampler.sample(redisWarningSampleKey(scope), time.Now())
	if !emit {
		return
	}
	logger.Warnf("redis rate limit unavailable; local fallback remains active: scope=%s err=%v suppressed=%d", scope, err, suppressed)
}

func logLocalCapacityWarning(scope string) {
	emit, suppressed := redisWarningLogSampler.sample("local_capacity|"+normalizeSubject(scope), time.Now())
	if !emit {
		return
	}
	logger.Warnf("local rate-limit subject capacity reached; new subjects rely on Redis plus endpoint/IP fuses: scope=%s suppressed=%d", scope, suppressed)
}
