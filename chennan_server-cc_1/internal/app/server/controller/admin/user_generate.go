package controller_admin

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"sync"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/app/server/tencentim"
	"github.com/ixayda/ix-Go/internal/pkg/authwork"
	"github.com/ixayda/ix-Go/internal/pkg/idgen"
	"github.com/ixayda/ix-Go/pkg/idgs"
)

const (
	maxGeneratedUsersPerRequest = 200
	phoneSegmentWidth           = 4
	passwordCharset             = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789"
	passwordModeFixed           = "fixed"
	passwordModePrefixRandom    = "prefix_random"
	passwordModeRandomStrong    = "random_strong"
	phoneMiddleModeSequence     = "sequence"
	phoneMiddleModeRandom       = "random"
	accountImportConcurrency    = 5
)

var (
	phonePrefixPattern = regexpMustCompile(`^1[3-9]\d$`)
	digits4Pattern     = regexpMustCompile(`^\d{4}$`)
)

// GeneratePhoneTemplateReq 表示手机号模板。
type GeneratePhoneTemplateReq struct {
	Prefix      string `json:"prefix"`
	MiddleMode  string `json:"middle_mode"`
	MiddleStart int    `json:"middle_start"`
	Suffix      string `json:"suffix"`
}

// GenerateNicknameTemplateReq 表示昵称模板。
type GenerateNicknameTemplateReq struct {
	Prefix  string `json:"prefix"`
	Start   int    `json:"start"`
	Padding int    `json:"padding"`
}

// GeneratePasswordTemplateReq 表示密码模板。
type GeneratePasswordTemplateReq struct {
	Mode         string `json:"mode"`
	FixedValue   string `json:"fixed_value"`
	Prefix       string `json:"prefix"`
	RandomLength int    `json:"random_length"`
}

// GenerateUsersReq 表示批量模板生成用户请求。
type GenerateUsersReq struct {
	Count            int                         `json:"count"`
	OwnerID          uint                        `json:"owner_id"`
	RealNameVerified *bool                       `json:"real_name_verified"` // nil/true=已实名（默认），false=未实名
	CanCreateGroup   *bool                       `json:"can_create_group"`   // nil/false=不允许建群（默认），true=允许
	PhoneTemplate    GeneratePhoneTemplateReq    `json:"phone_template"`
	NicknameTemplate GenerateNicknameTemplateReq `json:"nickname_template"`
	PasswordTemplate GeneratePasswordTemplateReq `json:"password_template"`
}

// GeneratedUserItem 表示单条模板生成结果。
type GeneratedUserItem struct {
	Index    int    `json:"index"`
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	Phone    string `json:"phone"`
	Nickname string `json:"nickname"`
	Password string `json:"password"`
	IMUserID string `json:"im_user_id"`
}

// GenerateUsersResult 表示模板生成汇总结果。
type GenerateUsersResult struct {
	Requested int                 `json:"requested"`
	Success   int                 `json:"success"`
	Failed    int                 `json:"failed"`
	Items     []GeneratedUserItem `json:"items"`
}

func regexpMustCompile(pattern string) *regexp.Regexp {
	return regexp.MustCompile(pattern)
}

// Validate 校验并归一化模板配置。
func (req *GenerateUsersReq) Validate() error {
	req.PhoneTemplate.Prefix = strings.TrimSpace(req.PhoneTemplate.Prefix)
	req.PhoneTemplate.MiddleMode = strings.TrimSpace(req.PhoneTemplate.MiddleMode)
	req.PhoneTemplate.Suffix = strings.TrimSpace(req.PhoneTemplate.Suffix)
	req.NicknameTemplate.Prefix = strings.TrimSpace(req.NicknameTemplate.Prefix)
	req.PasswordTemplate.Mode = strings.TrimSpace(req.PasswordTemplate.Mode)
	req.PasswordTemplate.FixedValue = strings.TrimSpace(req.PasswordTemplate.FixedValue)
	req.PasswordTemplate.Prefix = strings.TrimSpace(req.PasswordTemplate.Prefix)

	if req.Count <= 0 || req.Count > maxGeneratedUsersPerRequest {
		return fmt.Errorf("生成数量需在 1~%d 之间", maxGeneratedUsersPerRequest)
	}

	if !phonePrefixPattern.MatchString(req.PhoneTemplate.Prefix) {
		return errors.New("手机号前三位格式无效")
	}
	if !digits4Pattern.MatchString(req.PhoneTemplate.Suffix) {
		return errors.New("手机号后四位格式无效")
	}

	switch req.PhoneTemplate.MiddleMode {
	case phoneMiddleModeSequence:
		if req.PhoneTemplate.MiddleStart < 0 || req.PhoneTemplate.MiddleStart > 9999 {
			return errors.New("手机号中间四位起始值需在 0000~9999 之间")
		}
		if req.PhoneTemplate.MiddleStart+req.Count-1 > 9999 {
			return errors.New("手机号模板剩余可用空间不足，请调整中间四位起始值")
		}
	case phoneMiddleModeRandom:
	default:
		return errors.New("手机号中间四位模式无效")
	}

	if req.NicknameTemplate.Prefix == "" {
		return errors.New("昵称前缀不能为空")
	}
	if req.NicknameTemplate.Start < 0 {
		return errors.New("昵称起始编号不能小于 0")
	}
	if req.NicknameTemplate.Padding < 0 || req.NicknameTemplate.Padding > 8 {
		return errors.New("昵称补零位数需在 0~8 之间")
	}

	switch req.PasswordTemplate.Mode {
	case passwordModeFixed:
		if len(req.PasswordTemplate.FixedValue) < 8 {
			return errors.New("固定密码至少需要 8 位")
		}
	case passwordModePrefixRandom:
		if req.PasswordTemplate.Prefix == "" {
			return errors.New("密码固定前缀不能为空")
		}
		if req.PasswordTemplate.RandomLength < 1 || req.PasswordTemplate.RandomLength > 24 {
			return errors.New("密码随机后缀长度需在 1~24 之间")
		}
		if len(req.PasswordTemplate.Prefix)+req.PasswordTemplate.RandomLength < 8 {
			return errors.New("密码总长度至少需要 8 位")
		}
	case passwordModeRandomStrong:
		if req.PasswordTemplate.RandomLength == 0 {
			req.PasswordTemplate.RandomLength = 10
		}
		if req.PasswordTemplate.RandomLength < 8 || req.PasswordTemplate.RandomLength > 32 {
			return errors.New("随机密码长度需在 8~32 之间")
		}
	default:
		return errors.New("密码模板模式无效")
	}

	return nil
}

type phoneTemplateGenerator struct {
	template     GeneratePhoneTemplateReq
	repo         *repository.UserRepository
	usedPhones   map[string]struct{}
	nextSequence int
}

func newPhoneTemplateGenerator(template GeneratePhoneTemplateReq, repo *repository.UserRepository) *phoneTemplateGenerator {
	return &phoneTemplateGenerator{
		template:     template,
		repo:         repo,
		usedPhones:   make(map[string]struct{}),
		nextSequence: template.MiddleStart,
	}
}

func (g *phoneTemplateGenerator) Next() (string, error) {
	switch g.template.MiddleMode {
	case phoneMiddleModeSequence:
		for seq := g.nextSequence; seq <= 9999; seq++ {
			g.nextSequence = seq + 1
			phone := g.format(seq)
			exists, err := g.exists(phone)
			if err != nil {
				return "", err
			}
			if exists {
				continue
			}
			g.usedPhones[phone] = struct{}{}
			return phone, nil
		}
	case phoneMiddleModeRandom:
		for attempt := 0; attempt < 2000; attempt++ {
			seq, err := secureRandomInt(10000)
			if err != nil {
				return "", err
			}
			phone := g.format(seq)
			exists, err := g.exists(phone)
			if err != nil {
				return "", err
			}
			if exists {
				continue
			}
			g.usedPhones[phone] = struct{}{}
			return phone, nil
		}
	}
	return "", errors.New("手机号模板可用空间不足")
}

func (g *phoneTemplateGenerator) format(middle int) string {
	return fmt.Sprintf("%s%0*d%s", g.template.Prefix, phoneSegmentWidth, middle, g.template.Suffix)
}

func (g *phoneTemplateGenerator) exists(phone string) (bool, error) {
	if _, ok := g.usedPhones[phone]; ok {
		return true, nil
	}
	return g.repo.ExistsByPhone(phone)
}

type nicknameTemplateGenerator struct {
	template      GenerateNicknameTemplateReq
	usedNicknames map[string]struct{}
	nextSequence  int
}

func newNicknameTemplateGenerator(template GenerateNicknameTemplateReq, repo *repository.UserRepository) *nicknameTemplateGenerator {
	return &nicknameTemplateGenerator{
		template:      template,
		usedNicknames: make(map[string]struct{}),
		nextSequence:  template.Start,
	}
}

func (g *nicknameTemplateGenerator) Next() (string, error) {
	for attempt := 0; attempt < 20000; attempt++ {
		sequence := g.nextSequence
		g.nextSequence++
		nickname := g.format(sequence)
		if _, ok := g.usedNicknames[nickname]; ok {
			continue
		}
		g.usedNicknames[nickname] = struct{}{}
		return nickname, nil
	}
	return "", errors.New("昵称模板可用空间不足")
}

func (g *nicknameTemplateGenerator) format(sequence int) string {
	if g.template.Padding > 0 {
		return fmt.Sprintf("%s%0*d", g.template.Prefix, g.template.Padding, sequence)
	}
	return fmt.Sprintf("%s%d", g.template.Prefix, sequence)
}

// GenerateUsers 基于模板批量生成用户。
func GenerateUsers(ctx context.Context, actorID uint, isAdmin bool, req GenerateUsersReq) (*GenerateUsersResult, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	ownerID, err := resolveManagedOwnerID(actorID, isAdmin, req.OwnerID)
	if err != nil {
		return nil, err
	}

	userRepo := repository.NewUserRepository()
	phoneGenerator := newPhoneTemplateGenerator(req.PhoneTemplate, userRepo)
	nicknameGenerator := newNicknameTemplateGenerator(req.NicknameTemplate, userRepo)
	imClient := tencentim.NewClient()

	result := &GenerateUsersResult{
		Requested: req.Count,
		Items:     make([]GeneratedUserItem, req.Count),
	}
	type pendingImport struct {
		index        int
		user         *model.User
		inviteCodeID uint
	}
	pendingImports := make([]pendingImport, 0, req.Count)

generationLoop:
	for i := 0; i < req.Count; i++ {
		item := GeneratedUserItem{Index: i}
		for attempt := 0; attempt < 20; attempt++ {
			phone, err := phoneGenerator.Next()
			if err != nil {
				item.Message = err.Error()
				break
			}
			nickname, err := nicknameGenerator.Next()
			if err != nil {
				item.Message = err.Error()
				break
			}
			password, err := generatePasswordFromTemplate(req.PasswordTemplate)
			if err != nil {
				return nil, err
			}

			item.Phone = phone
			item.Nickname = nickname
			item.Password = password

			createdUser, inviteCodeID, err := createGeneratedUser(ctx, userRepo, ownerID, CreateUserReq{
				Phone:          phone,
				Password:       password,
				Nickname:       nickname,
				OwnerID:        ownerID,
				CreatedBy:      actorID,
				CanCreateGroup: req.CanCreateGroup != nil && *req.CanCreateGroup,
			})
			if err == nil {
				item.IMUserID = createdUser.IMUserID
				pendingImports = append(pendingImports, pendingImport{
					index:        i,
					user:         createdUser,
					inviteCodeID: inviteCodeID,
				})
				break
			}

			if isTemplateCollisionError(err) {
				continue
			}
			if errors.Is(err, authwork.ErrBusy) {
				// If no item has been processed yet, let the handler return 503. Once
				// earlier rows exist, preserve their result and finish the pending IM
				// imports instead of abandoning locally-created orphan accounts.
				if i == 0 {
					return nil, err
				}
				for pending := i; pending < req.Count; pending++ {
					result.Items[pending] = GeneratedUserItem{
						Index:   pending,
						Message: "服务繁忙，未执行",
					}
					result.Failed++
				}
				break generationLoop
			}

			item.Message = batchCreateUserErrorMessage(err)
			break
		}

		if item.IMUserID == "" {
			if item.Message == "" {
				item.Message = "模板生成失败，请调整模板后重试"
			}
			result.Failed++
		}
		result.Items[i] = item
	}

	if len(pendingImports) > 0 {
		sem := make(chan struct{}, accountImportConcurrency)
		var wg sync.WaitGroup
		var mu sync.Mutex

		for _, pending := range pendingImports {
			wg.Add(1)
			sem <- struct{}{}
			go func(p pendingImport) {
				defer wg.Done()
				defer func() { <-sem }()

				err := imClient.AccountImport(p.user.IMUserID, p.user.Nickname, p.user.AvatarURL)
				if err == nil {
					err = imClient.SetAllowType(p.user.IMUserID, p.user.AllowType)
					if err != nil {
						_ = imClient.AccountDelete(p.user.IMUserID)
					}
				}

				mu.Lock()
				defer mu.Unlock()

				if err != nil {
					_ = userRepo.Delete(p.user.ID)
					result.Items[p.index].Success = false
					result.Items[p.index].Message = "同步 IM 失败，请稍后重试"
					result.Failed++
					return
				}

				incrementInviteCodeUsage(p.inviteCodeID)
				if req.RealNameVerified == nil || *req.RealNameVerified {
					// 批量生成默认已实名；未勾选时保持未实名
					_ = repository.NewUserRealNameRepository().UpsertAssignedVerified(p.user.ID)
				}
				result.Items[p.index].Success = true
				result.Items[p.index].Message = "创建成功"
				result.Success++
			}(pending)
		}

		wg.Wait()
	}

	return result, nil
}

func createGeneratedUser(ctx context.Context, repo *repository.UserRepository, ownerID uint, req CreateUserReq) (*model.User, uint, error) {
	phone := strings.TrimSpace(req.Phone)
	if !phonePattern.MatchString(phone) {
		return nil, 0, errors.New("手机号格式无效")
	}

	imUserID, err := idgen.GenUserID(ctx, "a")
	if err != nil {
		return nil, 0, err
	}

	hash, err := authwork.GeneratePrivateFromPassword([]byte(req.Password), authwork.DefaultCost)
	if err != nil {
		return nil, 0, err
	}

	nickname := strings.TrimSpace(req.Nickname)
	if nickname == "" {
		nickname = "用户_" + imUserID[2:]
	}

	inviteCode, inviteCodeID, err := resolveOwnerInviteCode(ownerID)
	if err != nil {
		return nil, 0, err
	}

	user := &model.User{
		BizID:              idgs.Global.UserGenerate().Int64(),
		IMUserID:           imUserID,
		Phone:              phone,
		PasswordHash:       string(hash),
		Nickname:           nickname,
		AvatarURL:          req.AvatarURL,
		OwnerID:            ownerID,
		InviteCode:         inviteCode,
		CreatedBy:          req.CreatedBy,
		AllowType:          0,
		CanCreateGroup:     req.CanCreateGroup,
		Status:             "active",
		IsActivated:        false,
		MustChangePassword: true,
	}
	if err := repo.Create(user); err != nil {
		return nil, 0, err
	}
	return user, inviteCodeID, nil
}

func isTemplateCollisionError(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "手机号已注册") ||
		strings.Contains(message, "duplicate entry")
}

func generatePasswordFromTemplate(template GeneratePasswordTemplateReq) (string, error) {
	switch template.Mode {
	case passwordModeFixed:
		return template.FixedValue, nil
	case passwordModePrefixRandom:
		suffix, err := secureRandomString(template.RandomLength)
		if err != nil {
			return "", err
		}
		return template.Prefix + suffix, nil
	case passwordModeRandomStrong:
		return secureRandomString(template.RandomLength)
	default:
		return "", errors.New("密码模板模式无效")
	}
}

func secureRandomString(length int) (string, error) {
	builder := strings.Builder{}
	builder.Grow(length)
	for i := 0; i < length; i++ {
		index, err := secureRandomInt(len(passwordCharset))
		if err != nil {
			return "", err
		}
		builder.WriteByte(passwordCharset[index])
	}
	return builder.String(), nil
}

func secureRandomInt(max int) (int, error) {
	if max <= 0 {
		return 0, errors.New("随机范围无效")
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0, err
	}
	return int(n.Int64()), nil
}
