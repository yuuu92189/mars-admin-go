package application

import (
	"context"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"time"

	"mars-admin-go/internal/modules/identity/domain"
	"mars-admin-go/internal/platform/apperror"
	"mars-admin-go/internal/platform/requestctx"
	"mars-admin-go/internal/platform/security"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pquerna/otp/totp"
)

type Service struct {
	repo             domain.Repository
	rateLimiter      domain.RateLimiter
	sessionTTL       time.Duration
	refreshTTL       time.Duration
	maxLoginAttempts int32
	lockDuration     time.Duration
	now              func() time.Time
	codeSender       CodeSender
	captchaEnabled   bool
	secretBox        *security.SecretBox
	socialProviders  map[string]domain.SocialProvider
	accessCache      domain.Cache
}

type Options struct {
	SessionTTL       time.Duration
	RefreshTTL       time.Duration
	MaxLoginAttempts int32
	LockDuration     time.Duration
	Now              func() time.Time
	CodeSender       CodeSender
	CaptchaEnabled   bool
	SecretBox        *security.SecretBox
	SocialProviders  map[string]domain.SocialProvider
	AccessCache      domain.Cache
}

type CodeSender interface {
	SendCode(ctx context.Context, phone, code string) error
}

type policyRateLimiter interface {
	AllowPolicy(ctx context.Context, key string, limit int64, window time.Duration) (bool, error)
}

func New(repo domain.Repository, rateLimiter domain.RateLimiter, opts Options) *Service {
	if opts.Now == nil {
		opts.Now = func() time.Time { return time.Now().UTC() }
	}
	if opts.SessionTTL <= 0 {
		opts.SessionTTL = 30 * time.Minute
	}
	if opts.RefreshTTL <= 0 {
		opts.RefreshTTL = 30 * 24 * time.Hour
	}
	if opts.MaxLoginAttempts <= 0 {
		opts.MaxLoginAttempts = 5
	}
	if opts.LockDuration <= 0 {
		opts.LockDuration = 30 * time.Minute
	}
	return &Service{
		repo:             repo,
		rateLimiter:      rateLimiter,
		sessionTTL:       opts.SessionTTL,
		refreshTTL:       opts.RefreshTTL,
		maxLoginAttempts: opts.MaxLoginAttempts,
		lockDuration:     opts.LockDuration,
		now:              opts.Now,
		codeSender:       opts.CodeSender,
		captchaEnabled:   opts.CaptchaEnabled,
		secretBox:        opts.SecretBox,
		socialProviders:  opts.SocialProviders,
		accessCache:      opts.AccessCache,
	}
}

func (s *Service) accessFor(ctx context.Context, userID uuid.UUID) (domain.Access, error) {
	key := "access:" + userID.String()
	type cachedAccess struct {
		Epoch        int64    `json:"epoch"`
		Roles        []string `json:"roles"`
		Permissions  []string `json:"permissions"`
		IsSuperAdmin bool     `json:"isSuperAdmin"`
		DataScope    string   `json:"dataScope"`
	}
	epoch, err := s.repo.AccessCacheEpoch(ctx)
	if err != nil {
		return domain.Access{}, err
	}
	var cached cachedAccess
	if s.accessCache != nil {
		if hit, cacheErr := s.accessCache.Get(ctx, key, &cached); cacheErr == nil && hit && cached.Epoch == epoch {
			return domain.Access{
				Roles: cached.Roles, Permissions: cached.Permissions,
				IsSuperAdmin: cached.IsSuperAdmin, DataScope: cached.DataScope,
			}, nil
		}
	}
	access, err := s.repo.GetAccess(ctx, userID)
	if err != nil {
		return domain.Access{}, err
	}
	if s.accessCache != nil {
		_ = s.accessCache.Set(ctx, key, cachedAccess{
			Epoch: epoch,
			Roles: access.Roles, Permissions: access.Permissions,
			IsSuperAdmin: access.IsSuperAdmin, DataScope: access.DataScope,
		})
	}
	return access, nil
}

func (s *Service) LoginSocial(ctx context.Context, providerName, code, clientType, deviceName, ip, userAgent string) (domain.LoginResult, error) {
	providerName = strings.ToLower(strings.TrimSpace(providerName))
	provider := s.socialProviders[providerName]
	if provider == nil {
		return domain.LoginResult{}, apperror.New(apperror.KindNotConfigured, "social_provider_not_configured", "The social login provider is not configured.")
	}
	socialIdentity, err := provider.Exchange(ctx, code)
	if err != nil {
		return domain.LoginResult{}, apperror.Wrap(apperror.KindUnauthorized, "social_login_failed", "Social login failed.", err)
	}
	if socialIdentity.Provider == "" {
		socialIdentity.Provider = providerName
	}
	if strings.TrimSpace(socialIdentity.Subject) == "" {
		return domain.LoginResult{}, apperror.Unauthorized("The social provider did not return a subject.")
	}
	user, err := s.repo.FindSocialUser(ctx, socialIdentity.Provider, socialIdentity.Subject)
	if errors.Is(err, pgx.ErrNoRows) {
		randomPassword, tokenErr := security.GenerateToken(32)
		if tokenErr != nil {
			return domain.LoginResult{}, tokenErr
		}
		displayName := strings.TrimSpace(socialIdentity.Nickname)
		if displayName == "" {
			displayName = "Social User"
		}
		passwordHash, hashErr := security.HashPassword(randomPassword)
		if hashErr != nil {
			return domain.LoginResult{}, hashErr
		}
		suffix := strings.ReplaceAll(mustUUIDv7().String(), "-", "")[:12]
		created, createErr := s.repo.CreateSocialUser(ctx, domain.User{
			ID:           mustUUIDv7(),
			Username:     "social_" + suffix,
			DisplayName:  displayName,
			Email:        strings.TrimSpace(socialIdentity.Email),
			Phone:        strings.TrimSpace(socialIdentity.Phone),
			AvatarURL:    strings.TrimSpace(socialIdentity.Avatar),
			UserType:     "app",
			Status:       "active",
			PasswordHash: passwordHash,
		}, socialIdentity.Provider, socialIdentity.Subject, socialIdentity.Raw)
		if createErr != nil {
			existing, findErr := s.repo.FindSocialUser(ctx, socialIdentity.Provider, socialIdentity.Subject)
			if findErr != nil {
				return domain.LoginResult{}, createErr
			}
			user = existing
		} else {
			user = created
		}
	} else if err != nil {
		return domain.LoginResult{}, err
	}
	if user.Status != "active" {
		return domain.LoginResult{}, apperror.Forbidden("This account is not active.")
	}
	access, err := s.accessFor(ctx, user.ID)
	if err != nil {
		return domain.LoginResult{}, err
	}
	if clientType == "" {
		clientType = "app"
	}
	input := domain.LoginInput{ClientType: clientType, DeviceName: deviceName, IPAddress: ip, UserAgent: userAgent}
	result, err := s.issueSession(ctx, user, access, input)
	if err != nil {
		return domain.LoginResult{}, err
	}
	s.recordEvent(ctx, &user.ID, user.Username, clientType, "success", "social login", input)
	return result, nil
}

func (s *Service) UnbindSocialIdentity(ctx context.Context, actor requestctx.Actor, provider, subject string) error {
	if actor.UserID == uuid.Nil {
		return apperror.Unauthorized("Authentication is required.")
	}
	provider = strings.ToLower(strings.TrimSpace(provider))
	subject = strings.TrimSpace(subject)
	if provider == "" || len(provider) > 40 || subject == "" || len(subject) > 255 {
		return apperror.Validation("socialIdentity", "A valid social provider and subject are required.")
	}
	if err := s.repo.UnlinkSocialIdentity(ctx, actor.UserID, provider, subject); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apperror.NotFound("Social identity")
		}
		return err
	}
	return nil
}

func (s *Service) SendLoginCode(ctx context.Context, phone, ip, deviceName, userAgent string) error {
	phone = strings.TrimSpace(phone)
	if len(phone) < 8 || len(phone) > 32 {
		return apperror.Validation("phone", "A valid phone number is required.")
	}
	if s.codeSender == nil {
		return apperror.New(apperror.KindNotConfigured, "sms_not_configured", "SMS login is not configured.")
	}
	device := rateKeyPart(deviceName + "|" + userAgent)
	ipKey := rateKeyPart(ip)
	policies := []struct {
		key    string
		limit  int64
		window time.Duration
	}{
		{key: "sms:send:phone:" + rateKeyPart(phone), limit: 1, window: time.Minute},
		{key: "sms:send:ip:" + ipKey, limit: 5, window: 10 * time.Minute},
		{key: "sms:send:device:" + device, limit: 3, window: 10 * time.Minute},
	}
	if err := s.enforceRateLimits(ctx, policies); err != nil {
		return err
	}
	code, err := security.GenerateNumericCode(6)
	if err != nil {
		return err
	}
	if err := s.repo.InsertOneTimeCode(ctx, "login_sms", phone, security.HashToken(code), s.now().Add(5*time.Minute)); err != nil {
		return err
	}
	if err := s.codeSender.SendCode(ctx, phone, code); err != nil {
		return err
	}
	return nil
}

func (s *Service) IssueCaptcha(ctx context.Context, kind string) (domain.Captcha, error) {
	if kind == "" {
		kind = "image"
	}
	if kind != "image" && kind != "slider" {
		return domain.Captcha{}, apperror.Validation("type", "Captcha type must be image or slider.")
	}
	captcha, err := generateCaptcha(kind)
	if err != nil {
		return domain.Captcha{}, err
	}
	var targetX int32
	if captcha.Slider {
		targetX = int32(captcha.Answer) // #nosec G115 -- slider target is generated within the image width.
	}
	if err := s.repo.InsertCaptcha(
		ctx, captcha.ID, kind, security.HashToken(captcha.AnswerCode), targetX, captcha.Tolerance, s.now().Add(5*time.Minute),
	); err != nil {
		return domain.Captcha{}, err
	}
	return captcha, nil
}

func (s *Service) LoginSMS(ctx context.Context, phone, code, clientType, deviceName, ip, userAgent string) (domain.LoginResult, error) {
	phone = strings.TrimSpace(phone)
	code = strings.TrimSpace(code)
	if phone == "" || code == "" {
		return domain.LoginResult{}, apperror.Validation("code", "Phone and verification code are required.")
	}
	device := rateKeyPart(deviceName + "|" + userAgent)
	ipKey := rateKeyPart(ip)
	policies := []struct {
		key    string
		limit  int64
		window time.Duration
	}{
		{key: "sms:verify:phone:" + rateKeyPart(phone), limit: 10, window: 10 * time.Minute},
		{key: "sms:verify:ip:" + ipKey, limit: 20, window: 10 * time.Minute},
		{key: "sms:verify:device:" + device, limit: 20, window: 10 * time.Minute},
	}
	if err := s.enforceRateLimits(ctx, policies); err != nil {
		return domain.LoginResult{}, err
	}
	attempt, err := s.repo.AttemptOneTimeCode(ctx, "login_sms", phone, security.HashToken(code))
	if err != nil {
		return domain.LoginResult{}, err
	}
	if !attempt.Valid {
		return domain.LoginResult{}, apperror.Unauthorized("The phone number or verification code is invalid or expired.")
	}
	user, err := s.repo.FindUserByPhone(ctx, phone)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.LoginResult{}, apperror.Unauthorized("The phone number or verification code is invalid or expired.")
		}
		return domain.LoginResult{}, err
	}
	if user.Status != "active" {
		return domain.LoginResult{}, apperror.Unauthorized("The phone number or verification code is invalid or expired.")
	}
	access, err := s.accessFor(ctx, user.ID)
	if err != nil {
		return domain.LoginResult{}, err
	}
	if err := s.repo.ClearFailedLogin(ctx, user.ID); err != nil {
		return domain.LoginResult{}, err
	}
	input := domain.LoginInput{ClientType: clientType, DeviceName: deviceName, IPAddress: ip, UserAgent: userAgent}
	result, err := s.issueSession(ctx, user, access, input)
	if err != nil {
		return domain.LoginResult{}, err
	}
	s.recordEvent(ctx, &user.ID, user.Username, clientType, "success", "sms login", input)
	return result, nil
}

func (s *Service) Login(ctx context.Context, input domain.LoginInput) (domain.LoginResult, error) {
	if input.ClientType == "" {
		input.ClientType = "admin"
	}
	if input.ClientType != "admin" && input.ClientType != "web" && input.ClientType != "app" {
		return domain.LoginResult{}, apperror.Validation("clientType", "clientType must be admin, web, or app")
	}
	input.Username = strings.TrimSpace(input.Username)
	if input.Username == "" || input.Password == "" {
		return domain.LoginResult{}, apperror.Validation("username", "Username and password are required.")
	}
	if s.rateLimiter != nil {
		allowed, err := s.rateLimiter.Allow(ctx, "login:"+strings.ToLower(input.Username))
		if err != nil {
			return domain.LoginResult{}, apperror.Wrap(apperror.KindUnavailable, "rate_limiter_unavailable", "Login is temporarily unavailable.", err)
		}
		if !allowed {
			s.recordEvent(ctx, nil, input.Username, input.ClientType, "failure", "rate limited", input)
			return domain.LoginResult{}, apperror.New(apperror.KindRateLimited, "login_rate_limited", "Too many login attempts. Try again later.")
		}
	}
	if err := s.enforceLoginSourceRateLimits(ctx, input); err != nil {
		s.recordEvent(ctx, nil, input.Username, input.ClientType, "failure", "source rate limited", input)
		return domain.LoginResult{}, err
	}
	if s.captchaEnabled {
		if input.CaptchaID == "" || input.CaptchaCode == "" {
			return domain.LoginResult{}, apperror.Validation("code", "A captcha is required.")
		}
		captchaID, parseErr := uuid.Parse(input.CaptchaID)
		if parseErr != nil {
			return domain.LoginResult{}, apperror.Unauthorized("The captcha is invalid or expired.")
		}
		sliderX := int64(-1)
		if parsed, err := strconv.ParseInt(input.CaptchaCode, 10, 32); err == nil {
			sliderX = parsed
		}
		if _, err := s.repo.ConsumeCaptcha(ctx, captchaID, security.HashToken(input.CaptchaCode), int32(sliderX)); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.LoginResult{}, apperror.Unauthorized("The captcha is invalid or expired.")
			}
			return domain.LoginResult{}, err
		}
	}

	user, err := s.repo.FindUserByUsername(ctx, input.Username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.recordEvent(ctx, nil, input.Username, input.ClientType, "failure", "user not found", input)
			return domain.LoginResult{}, apperror.Unauthorized("Username or password is incorrect.")
		}
		return domain.LoginResult{}, err
	}
	now := s.now()
	if user.Status != "active" {
		s.recordEvent(ctx, &user.ID, input.Username, input.ClientType, "failure", "status "+user.Status, input)
		return domain.LoginResult{}, apperror.Unauthorized("This account is not active.")
	}
	if !user.PasswordLoginEnabled {
		s.recordEvent(ctx, &user.ID, input.Username, input.ClientType, "failure", "password login disabled", input)
		return domain.LoginResult{}, apperror.Unauthorized("Username or password is incorrect.")
	}
	if user.LockedUntil != nil && user.LockedUntil.After(now) {
		s.recordEvent(ctx, &user.ID, input.Username, input.ClientType, "failure", "account locked", input)
		return domain.LoginResult{}, apperror.New(apperror.KindLocked, "account_locked", "The account is temporarily locked.")
	}
	valid, err := security.VerifyPassword(input.Password, user.PasswordHash)
	if err != nil && !errors.Is(err, security.ErrInvalidPasswordHash) {
		return domain.LoginResult{}, err
	}
	if !valid {
		lockUntil := now.Add(s.lockDuration)
		if err := s.repo.RecordFailedLogin(ctx, user.ID, s.maxLoginAttempts, lockUntil); err != nil {
			return domain.LoginResult{}, err
		}
		s.recordEvent(ctx, &user.ID, input.Username, input.ClientType, "failure", "password mismatch", input)
		return domain.LoginResult{}, apperror.Unauthorized("Username or password is incorrect.")
	}
	if security.NeedsRehash(user.PasswordHash) {
		rehash, hashErr := security.HashPassword(input.Password)
		if hashErr != nil {
			return domain.LoginResult{}, hashErr
		}
		if updateErr := s.repo.UpdatePassword(ctx, user.ID, rehash); updateErr != nil {
			return domain.LoginResult{}, updateErr
		}
	}
	if user.MFAEnabled {
		secret, _, err := s.repo.MFASecret(ctx, user.ID)
		if err != nil {
			return domain.LoginResult{}, err
		}
		if s.secretBox == nil {
			return domain.LoginResult{}, apperror.New(apperror.KindNotConfigured, "mfa_not_configured", "Multi-factor authentication is not configured.")
		}
		plaintext, err := s.secretBox.Decrypt(secret)
		if err != nil {
			return domain.LoginResult{}, err
		}
		if input.MFACode == "" {
			return domain.LoginResult{}, apperror.New(apperror.KindMFARequired, "mfa_required", "A TOTP code is required.")
		}
		if !totp.Validate(input.MFACode, string(plaintext)) {
			s.recordEvent(ctx, &user.ID, input.Username, input.ClientType, "failure", "invalid totp", input)
			return domain.LoginResult{}, apperror.Unauthorized("The TOTP code is invalid.")
		}
	}
	if err := s.repo.ClearFailedLogin(ctx, user.ID); err != nil {
		return domain.LoginResult{}, err
	}
	access, err := s.accessFor(ctx, user.ID)
	if err != nil {
		return domain.LoginResult{}, err
	}
	result, err := s.issueSession(ctx, user, access, input)
	if err != nil {
		return domain.LoginResult{}, err
	}
	s.recordEvent(ctx, &user.ID, input.Username, input.ClientType, "success", "", input)
	return result, nil
}

func (s *Service) issueSession(ctx context.Context, user domain.User, access domain.Access, input domain.LoginInput) (domain.LoginResult, error) {
	rawToken, err := security.GenerateToken(32)
	if err != nil {
		return domain.LoginResult{}, err
	}
	rawRefresh, err := security.GenerateToken(48)
	if err != nil {
		return domain.LoginResult{}, err
	}
	csrf, err := security.GenerateToken(32)
	if err != nil {
		return domain.LoginResult{}, err
	}
	now := s.now()
	refreshTTL := s.refreshTTL
	if !input.RememberMe && refreshTTL > 7*24*time.Hour {
		refreshTTL = 7 * 24 * time.Hour
	}
	session := domain.Session{
		ID:                mustUUIDv7(),
		UserID:            user.ID,
		RefreshFamilyID:   uuid.Nil,
		RefreshGeneration: 0,
		ClientType:        input.ClientType,
		DeviceName:        strings.TrimSpace(input.DeviceName),
		IPAddress:         input.IPAddress,
		UserAgent:         input.UserAgent,
		ExpiresAt:         now.Add(s.sessionTTL),
		RefreshExpiresAt:  now.Add(refreshTTL),
		LastSeenAt:        now,
		CreatedAt:         now,
		TokenHash:         security.HashToken(rawToken),
		RefreshTokenHash:  security.HashToken(rawRefresh),
	}
	session.RefreshFamilyID = session.ID
	if err := s.repo.CreateSession(ctx, session); err != nil {
		return domain.LoginResult{}, err
	}
	return domain.LoginResult{
		AccessToken:  rawToken,
		RefreshToken: rawRefresh,
		CSRFToken:    csrf,
		SessionID:    session.ID,
		ExpiresAt:    session.ExpiresAt,
		User:         user,
		Access:       access,
	}, nil
}

func (s *Service) Authenticate(ctx context.Context, token string) (requestctx.Actor, error) {
	session, err := s.repo.GetSessionByAccessHash(ctx, security.HashToken(token))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return requestctx.Actor{}, apperror.Unauthorized("The session is invalid or expired.")
		}
		return requestctx.Actor{}, err
	}
	user, err := s.repo.FindUserByID(ctx, session.UserID)
	if err != nil || user.Status != "active" {
		return requestctx.Actor{}, apperror.Unauthorized("The session is invalid or expired.")
	}
	access, err := s.accessFor(ctx, session.UserID)
	if err != nil {
		return requestctx.Actor{}, err
	}
	if err := s.repo.TouchSession(ctx, session.ID); err != nil {
		return requestctx.Actor{}, err
	}
	return requestctx.Actor{
		UserID:       user.ID,
		SessionID:    session.ID,
		Username:     user.Username,
		DisplayName:  user.DisplayName,
		ClientType:   session.ClientType,
		Roles:        access.Roles,
		Permissions:  access.Permissions,
		IsSuperAdmin: access.IsSuperAdmin,
	}, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (domain.LoginResult, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return domain.LoginResult{}, apperror.Unauthorized("The refresh token is invalid or expired.")
	}
	oldRefreshHash := security.HashToken(strings.TrimSpace(refreshToken))
	session, err := s.repo.GetSessionByRefreshHash(ctx, oldRefreshHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			if _, replayErr := s.repo.RevokeRefreshReplay(ctx, oldRefreshHash); replayErr != nil {
				return domain.LoginResult{}, replayErr
			}
			return domain.LoginResult{}, apperror.Unauthorized("The refresh token is invalid or expired.")
		}
		return domain.LoginResult{}, err
	}
	user, err := s.repo.FindUserByID(ctx, session.UserID)
	if err != nil || user.Status != "active" {
		return domain.LoginResult{}, apperror.Unauthorized("The refresh token is invalid or expired.")
	}
	access, err := s.accessFor(ctx, session.UserID)
	if err != nil {
		return domain.LoginResult{}, err
	}
	rawToken, err := security.GenerateToken(32)
	if err != nil {
		return domain.LoginResult{}, err
	}
	rawRefresh, err := security.GenerateToken(48)
	if err != nil {
		return domain.LoginResult{}, err
	}
	csrf, err := security.GenerateToken(32)
	if err != nil {
		return domain.LoginResult{}, err
	}
	now := s.now()
	session.TokenHash = security.HashToken(rawToken)
	session.RefreshTokenHash = security.HashToken(rawRefresh)
	session.ExpiresAt = now.Add(s.sessionTTL)
	session.RefreshExpiresAt = now.Add(s.refreshTTL)
	if err := s.repo.RotateSession(ctx, oldRefreshHash, session); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			if _, replayErr := s.repo.RevokeRefreshReplay(ctx, oldRefreshHash); replayErr != nil {
				return domain.LoginResult{}, replayErr
			}
			return domain.LoginResult{}, apperror.Unauthorized("The refresh token is invalid or expired.")
		}
		return domain.LoginResult{}, err
	}
	return domain.LoginResult{
		AccessToken:  rawToken,
		RefreshToken: rawRefresh,
		CSRFToken:    csrf,
		SessionID:    session.ID,
		ExpiresAt:    session.ExpiresAt,
		User:         user,
		Access:       access,
	}, nil
}

func (s *Service) Logout(ctx context.Context, actor requestctx.Actor) error {
	return s.repo.RevokeSession(ctx, actor.SessionID)
}

func (s *Service) ListSessions(ctx context.Context, userID uuid.UUID) ([]domain.Session, error) {
	return s.repo.ListUserSessions(ctx, userID)
}

func (s *Service) ListActiveSessions(ctx context.Context) ([]domain.Session, error) {
	return s.repo.ListActiveSessions(ctx)
}

func (s *Service) RevokeSession(ctx context.Context, sessionID uuid.UUID) error {
	return s.repo.RevokeSession(ctx, sessionID)
}

func (s *Service) RevokeOwnedSession(ctx context.Context, actor requestctx.Actor, sessionID uuid.UUID) error {
	if actor.UserID == uuid.Nil {
		return apperror.Unauthorized("Authentication is required.")
	}
	if err := s.repo.RevokeOwnedSession(ctx, sessionID, actor.UserID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apperror.NotFound("Session")
		}
		return err
	}
	return nil
}

func (s *Service) Profile(ctx context.Context, userID uuid.UUID) (domain.User, error) {
	user, err := s.repo.FindUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, apperror.NotFound("User")
		}
		return domain.User{}, err
	}
	return user, nil
}

func (s *Service) UpdateProfile(ctx context.Context, userID uuid.UUID, input domain.UpdateUserInput) (domain.User, error) {
	current, err := s.Profile(ctx, userID)
	if err != nil {
		return domain.User{}, err
	}
	if input.DisplayName == "" {
		input.DisplayName = current.DisplayName
	}
	if input.UserType == "" {
		input.UserType = current.UserType
	}
	if input.Status == "" {
		input.Status = current.Status
	}
	if input.Version == 0 {
		input.Version = current.Version
	}
	if input.Gender == 0 && current.Gender != 0 {
		input.Gender = current.Gender
	}
	input.RoleIDs = nil
	input.DepartmentIDs = nil
	input.PositionIDs = nil
	input.PrimaryDepartmentID = nil
	return s.repo.UpdateUser(ctx, userID, input, "")
}

func (s *Service) ListUsers(ctx context.Context, actor requestctx.Actor, filter domain.UserFilter) ([]domain.User, int64, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 100 {
		filter.PageSize = 20
	}
	allowedSorts := map[string]struct{}{"": {}, "username_asc": {}, "username_desc": {}, "created_desc": {}, "created_asc": {}}
	if _, ok := allowedSorts[filter.Sort]; !ok {
		return nil, 0, apperror.Validation("sort", "Unsupported sort field.")
	}
	filter.ActorID = actor.UserID
	access, err := s.accessFor(ctx, actor.UserID)
	if err != nil {
		return nil, 0, err
	}
	if access.IsSuperAdmin || actor.IsSuperAdmin {
		filter.DataScope = "all"
	} else {
		filter.DataScope = access.DataScope
		if filter.DataScope == "" {
			filter.DataScope = "self"
		}
	}
	return s.repo.ListUsers(ctx, filter)
}

func (s *Service) GetUser(ctx context.Context, actor requestctx.Actor, id uuid.UUID) (domain.User, error) {
	user, _, err := s.authorizeTargetUser(ctx, actor, id, false)
	return user, err
}

func (s *Service) CreateUserAsActor(ctx context.Context, actor requestctx.Actor, input domain.CreateUserInput) (domain.User, error) {
	access, err := s.actorAccess(ctx, actor)
	if err != nil {
		return domain.User{}, err
	}
	if err := s.validateRoleAssignment(ctx, access, input.RoleIDs); err != nil {
		return domain.User{}, err
	}
	if err := s.validateDepartmentScope(ctx, actor.UserID, access, input.DepartmentIDs); err != nil {
		return domain.User{}, err
	}
	return s.CreateUser(ctx, input)
}

func (s *Service) CreateUser(ctx context.Context, input domain.CreateUserInput) (domain.User, error) {
	input.Username = strings.TrimSpace(input.Username)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	if len(input.Username) < 4 || len(input.Username) > 50 {
		return domain.User{}, apperror.Validation("username", "Username must contain 4 to 50 characters.")
	}
	if input.DisplayName == "" {
		input.DisplayName = input.Username
	}
	if input.UserType == "" {
		input.UserType = "admin"
	}
	if input.Status == "" {
		input.Status = "active"
	}
	if input.Gender < 0 || input.Gender > 2 {
		return domain.User{}, apperror.Validation("gender", "Gender must be 0, 1, or 2.")
	}
	hash, err := security.HashPassword(input.Password)
	if err != nil {
		return domain.User{}, apperror.Validation("password", err.Error())
	}
	now := s.now()
	user := domain.User{
		ID:           mustUUIDv7(),
		Username:     input.Username,
		DisplayName:  input.DisplayName,
		Email:        strings.TrimSpace(input.Email),
		Phone:        strings.TrimSpace(input.Phone),
		AvatarURL:    strings.TrimSpace(input.AvatarURL),
		UserType:     input.UserType,
		Status:       input.Status,
		Gender:       input.Gender,
		PasswordHash: hash,
		CreatedAt:    now,
		UpdatedAt:    now,
		Version:      1,
	}
	return s.repo.CreateUser(ctx, user, input.RoleIDs, input.DepartmentIDs, input.PositionIDs, input.PrimaryDepartmentID)
}

func (s *Service) UpdateUser(ctx context.Context, actor requestctx.Actor, id uuid.UUID, input domain.UpdateUserInput) (domain.User, error) {
	_, access, err := s.authorizeTargetUser(ctx, actor, id, true)
	if err != nil {
		return domain.User{}, err
	}
	if err := s.validateRoleAssignment(ctx, access, input.RoleIDs); err != nil {
		return domain.User{}, err
	}
	if err := s.validateDepartmentScope(ctx, actor.UserID, access, input.DepartmentIDs); err != nil {
		return domain.User{}, err
	}
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	if input.DisplayName == "" {
		return domain.User{}, apperror.Validation("displayName", "Display name is required.")
	}
	if input.UserType != "admin" && input.UserType != "pc" && input.UserType != "app" {
		return domain.User{}, apperror.Validation("userType", "userType must be admin, pc, or app.")
	}
	if !validUserStatus(input.Status) {
		return domain.User{}, apperror.Validation("status", "Unsupported user status.")
	}
	if input.Gender < 0 || input.Gender > 2 {
		return domain.User{}, apperror.Validation("gender", "Gender must be 0, 1, or 2.")
	}
	user, err := s.repo.UpdateUser(ctx, id, input, "")
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, apperror.Conflict("The user changed since it was loaded. Refresh and try again.")
		}
		return domain.User{}, err
	}
	s.invalidateUserAccess(ctx, id)
	return user, nil
}

func (s *Service) DeleteUser(ctx context.Context, actor requestctx.Actor, id uuid.UUID) error {
	if _, _, err := s.authorizeTargetUser(ctx, actor, id, true); err != nil {
		return err
	}
	if err := s.repo.SoftDeleteUser(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apperror.NotFound("User")
		}
		return err
	}
	s.invalidateUserAccess(ctx, id)
	return nil
}

func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error {
	user, err := s.repo.FindUserByID(ctx, userID)
	if err != nil {
		return err
	}
	valid, err := security.VerifyPassword(oldPassword, user.PasswordHash)
	if err != nil || !valid {
		return apperror.Unauthorized("Current password is incorrect.")
	}
	hash, err := security.HashPassword(newPassword)
	if err != nil {
		return apperror.Validation("newPassword", err.Error())
	}
	return s.repo.ChangePasswordAndRevokeSessions(ctx, userID, hash)
}

func (s *Service) ResetPassword(ctx context.Context, actor requestctx.Actor, userID uuid.UUID, newPassword string) (string, error) {
	if _, _, err := s.authorizeTargetUser(ctx, actor, userID, true); err != nil {
		return "", err
	}
	if newPassword == "" {
		generated, err := security.GenerateToken(12)
		if err != nil {
			return "", err
		}
		newPassword = generated
	}
	hash, err := security.HashPassword(newPassword)
	if err != nil {
		return "", apperror.Validation("password", err.Error())
	}
	if err := s.repo.ChangePasswordAndRevokeSessions(ctx, userID, hash); err != nil {
		return "", err
	}
	return newPassword, nil
}

func (s *Service) RevokeUserSessions(ctx context.Context, actor requestctx.Actor, userID uuid.UUID) error {
	if _, _, err := s.authorizeTargetUser(ctx, actor, userID, true); err != nil {
		return err
	}
	return s.repo.RevokeUserSessions(ctx, userID)
}

func (s *Service) Register(ctx context.Context, input domain.RegisterInput) (domain.User, error) {
	if input.UserType == "" {
		input.UserType = "pc"
	}
	user, err := s.CreateUser(ctx, domain.CreateUserInput{
		Username:    input.Username,
		Password:    input.Password,
		DisplayName: input.DisplayName,
		Email:       input.Email,
		Phone:       input.Phone,
		UserType:    input.UserType,
		Status:      "pending",
	})
	if err != nil {
		return domain.User{}, err
	}
	if roleID, err := s.repo.FindRoleByCode(ctx, "user"); err == nil {
		_ = s.repo.SetUserRoles(ctx, user.ID, []uuid.UUID{roleID})
		s.invalidateUserAccess(ctx, user.ID)
	}
	return user, nil
}

func (s *Service) BootstrapAdmin(ctx context.Context, username, password, email string) (uuid.UUID, error) {
	if strings.TrimSpace(username) == "" || password == "" {
		return uuid.Nil, apperror.Validation("bootstrap", "Bootstrap admin requires a username and password.")
	}
	hash, err := security.HashPassword(password)
	if err != nil {
		return uuid.Nil, apperror.Validation("bootstrapPassword", err.Error())
	}
	return s.repo.EnsureBootstrapAdmin(ctx, strings.TrimSpace(username), hash, strings.TrimSpace(email))
}

type MFASetup struct {
	Secret string `json:"secret"`
	URL    string `json:"url"`
}

func (s *Service) SetupMFA(ctx context.Context, userID uuid.UUID) (MFASetup, error) {
	if s.secretBox == nil {
		return MFASetup{}, apperror.New(apperror.KindNotConfigured, "mfa_not_configured", "Multi-factor authentication is not configured.")
	}
	user, err := s.Profile(ctx, userID)
	if err != nil {
		return MFASetup{}, err
	}
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer: "Mars Admin", AccountName: user.Username,
	})
	if err != nil {
		return MFASetup{}, err
	}
	encrypted, err := s.secretBox.Encrypt([]byte(key.Secret()))
	if err != nil {
		return MFASetup{}, err
	}
	if err := s.repo.SetMFASecret(ctx, userID, encrypted); err != nil {
		return MFASetup{}, err
	}
	return MFASetup{Secret: key.Secret(), URL: key.URL()}, nil
}

func (s *Service) EnableMFA(ctx context.Context, userID uuid.UUID, code string) error {
	secret, _, err := s.repo.MFASecret(ctx, userID)
	if err != nil {
		return err
	}
	if s.secretBox == nil {
		return apperror.New(apperror.KindNotConfigured, "mfa_not_configured", "Multi-factor authentication is not configured.")
	}
	plaintext, err := s.secretBox.Decrypt(secret)
	if err != nil {
		return err
	}
	if !totp.Validate(code, string(plaintext)) {
		return apperror.Unauthorized("The TOTP code is invalid.")
	}
	return s.repo.EnableMFA(ctx, userID)
}

func (s *Service) DisableMFA(ctx context.Context, userID uuid.UUID, password, code string) error {
	user, err := s.Profile(ctx, userID)
	if err != nil {
		return err
	}
	valid, err := security.VerifyPassword(password, user.PasswordHash)
	if err != nil || !valid {
		return apperror.Unauthorized("Password is incorrect.")
	}
	secret, enabled, err := s.repo.MFASecret(ctx, userID)
	if err != nil {
		return err
	}
	if enabled {
		if s.secretBox == nil {
			return apperror.New(apperror.KindNotConfigured, "mfa_not_configured", "Multi-factor authentication is not configured.")
		}
		plaintext, err := s.secretBox.Decrypt(secret)
		if err != nil {
			return err
		}
		if !totp.Validate(code, string(plaintext)) {
			return apperror.Unauthorized("The TOTP code is invalid.")
		}
	}
	return s.repo.DisableMFA(ctx, userID)
}

func (s *Service) recordEvent(ctx context.Context, userID *uuid.UUID, username, clientType, outcome, reason string, input domain.LoginInput) {
	_ = s.repo.RecordLoginEvent(ctx, domain.LoginEvent{
		UserID:        userID,
		Username:      username,
		ClientType:    clientType,
		Outcome:       outcome,
		FailureReason: reason,
		IPAddress:     input.IPAddress,
		UserAgent:     input.UserAgent,
	})
}

func (s *Service) invalidateUserAccess(ctx context.Context, userID uuid.UUID) {
	if s.accessCache != nil {
		_ = s.accessCache.DeletePrefix(ctx, "access:"+userID.String())
	}
}

func (s *Service) actorAccess(ctx context.Context, actor requestctx.Actor) (domain.Access, error) {
	if actor.UserID == uuid.Nil {
		return domain.Access{}, apperror.Unauthorized("Authentication is required.")
	}
	access, err := s.accessFor(ctx, actor.UserID)
	if err != nil {
		return domain.Access{}, err
	}
	if access.IsSuperAdmin {
		access.DataScope = "all"
	}
	if access.DataScope == "" {
		access.DataScope = "self"
	}
	return access, nil
}

func (s *Service) authorizeTargetUser(ctx context.Context, actor requestctx.Actor, targetID uuid.UUID, denySuperAdminMutation bool) (domain.User, domain.Access, error) {
	actorAccess, err := s.actorAccess(ctx, actor)
	if err != nil {
		return domain.User{}, domain.Access{}, err
	}
	if actorAccess.IsSuperAdmin {
		target, err := s.repo.FindUserByID(ctx, targetID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.User{}, domain.Access{}, apperror.NotFound("User")
			}
			return domain.User{}, domain.Access{}, err
		}
		return target, actorAccess, nil
	}
	scoped, err := s.repo.FindUserByIDInDataScope(ctx, targetID, actor.UserID, actorAccess.DataScope)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, domain.Access{}, apperror.Forbidden("The target user is outside your data scope.")
		}
		return domain.User{}, domain.Access{}, err
	}
	if denySuperAdminMutation {
		targetAccess, accessErr := s.accessFor(ctx, targetID)
		if accessErr != nil {
			return domain.User{}, domain.Access{}, accessErr
		}
		if targetAccess.IsSuperAdmin {
			return domain.User{}, domain.Access{}, apperror.Forbidden("A regular administrator cannot modify a super administrator.")
		}
	}
	return scoped, actorAccess, nil
}

func (s *Service) validateRoleAssignment(ctx context.Context, access domain.Access, roleIDs []uuid.UUID) error {
	if roleIDs == nil {
		return nil
	}
	for _, roleID := range roleIDs {
		if roleID == uuid.Nil {
			return apperror.Validation("roleIds", "Role identifiers must be valid UUIDs.")
		}
	}
	if access.IsSuperAdmin {
		return nil
	}
	privileged, err := s.repo.CountPrivilegedRoles(ctx, roleIDs)
	if err != nil {
		return err
	}
	if privileged > 0 {
		return apperror.Forbidden("Only a super administrator can assign super administrator roles.")
	}
	return nil
}

func (s *Service) validateDepartmentScope(ctx context.Context, actorID uuid.UUID, access domain.Access, departmentIDs []uuid.UUID) error {
	if departmentIDs == nil || access.IsSuperAdmin {
		return nil
	}
	unique := make(map[uuid.UUID]struct{}, len(departmentIDs))
	for _, departmentID := range departmentIDs {
		if departmentID == uuid.Nil {
			return apperror.Validation("departmentIds", "Department identifiers must be valid UUIDs.")
		}
		unique[departmentID] = struct{}{}
	}
	if len(unique) == 0 {
		return nil
	}
	if access.DataScope == "self" {
		return apperror.Forbidden("A self-scoped user cannot assign departments.")
	}
	allowed, err := s.repo.CountDepartmentsWithinScope(ctx, actorID, access.DataScope, mapKeys(unique))
	if err != nil {
		return err
	}
	if allowed != int64(len(unique)) {
		return apperror.Forbidden("One or more departments are outside your data scope.")
	}
	return nil
}

func mapKeys(values map[uuid.UUID]struct{}) []uuid.UUID {
	keys := make([]uuid.UUID, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}

func (s *Service) enforceRateLimits(ctx context.Context, policies []struct {
	key    string
	limit  int64
	window time.Duration
}) error {
	if s.rateLimiter == nil {
		return nil
	}
	allowed := true
	for _, policy := range policies {
		var (
			ok  bool
			err error
		)
		if limiter, implementsPolicy := s.rateLimiter.(policyRateLimiter); implementsPolicy {
			ok, err = limiter.AllowPolicy(ctx, policy.key, policy.limit, policy.window)
		} else {
			ok, err = s.rateLimiter.Allow(ctx, policy.key)
		}
		if err != nil {
			return apperror.Wrap(apperror.KindUnavailable, "rate_limiter_unavailable", "SMS verification is temporarily unavailable.", err)
		}
		allowed = allowed && ok
	}
	if !allowed {
		return apperror.New(apperror.KindRateLimited, "sms_rate_limited", "Too many SMS requests or verification attempts. Try again later.")
	}
	return nil
}

func (s *Service) enforceLoginSourceRateLimits(ctx context.Context, input domain.LoginInput) error {
	if s.rateLimiter == nil {
		return nil
	}
	device := rateKeyPart(input.DeviceName + "|" + input.UserAgent)
	policies := []struct {
		key    string
		limit  int64
		window time.Duration
	}{
		{key: "login:ip:" + rateKeyPart(input.IPAddress), limit: 30, window: 5 * time.Minute},
		{key: "login:device:" + device, limit: 20, window: 5 * time.Minute},
	}
	allowed := true
	for _, policy := range policies {
		var (
			ok  bool
			err error
		)
		if limiter, implementsPolicy := s.rateLimiter.(policyRateLimiter); implementsPolicy {
			ok, err = limiter.AllowPolicy(ctx, policy.key, policy.limit, policy.window)
		} else {
			ok, err = s.rateLimiter.Allow(ctx, policy.key)
		}
		if err != nil {
			return apperror.Wrap(apperror.KindUnavailable, "rate_limiter_unavailable", "Login is temporarily unavailable.", err)
		}
		allowed = allowed && ok
	}
	if !allowed {
		return apperror.New(apperror.KindRateLimited, "login_rate_limited", "Too many login attempts. Try again later.")
	}
	return nil
}

func rateKeyPart(value string) string {
	return base64.RawURLEncoding.EncodeToString(security.HashToken(strings.TrimSpace(value)))
}

func validUserStatus(value string) bool {
	switch value {
	case "pending", "active", "suspended", "rejected", "left":
		return true
	default:
		return false
	}
}

func mustUUIDv7() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.New()
	}
	return id
}
