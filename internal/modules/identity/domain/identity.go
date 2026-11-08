package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID                   uuid.UUID   `json:"id"`
	Username             string      `json:"username"`
	DisplayName          string      `json:"displayName"`
	Email                string      `json:"email,omitempty"`
	Phone                string      `json:"phone,omitempty"`
	AvatarURL            string      `json:"avatarUrl,omitempty"`
	UserType             string      `json:"userType"`
	Status               string      `json:"status"`
	Gender               int16       `json:"gender"`
	LastLoginAt          *time.Time  `json:"lastLoginAt,omitempty"`
	CreatedAt            time.Time   `json:"createdAt"`
	UpdatedAt            time.Time   `json:"updatedAt"`
	Version              int64       `json:"version"`
	MFAEnabled           bool        `json:"mfaEnabled"`
	PasswordHash         string      `json:"-"`
	PasswordLoginEnabled bool        `json:"-"`
	FailedLogin          int32       `json:"-"`
	LockedUntil          *time.Time  `json:"-"`
	DepartmentIDs        []uuid.UUID `json:"departmentIds,omitempty"`
	RoleIDs              []uuid.UUID `json:"roleIds,omitempty"`
	PositionIDs          []uuid.UUID `json:"positionIds,omitempty"`
}

type Access struct {
	Roles        []string `json:"roles"`
	Permissions  []string `json:"permissions"`
	IsSuperAdmin bool     `json:"-"`
	DataScope    string   `json:"-"`
}

type Session struct {
	ID                uuid.UUID `json:"id"`
	UserID            uuid.UUID `json:"userId"`
	RefreshFamilyID   uuid.UUID `json:"refreshFamilyId"`
	RefreshGeneration int64     `json:"refreshGeneration"`
	ClientType        string    `json:"clientType"`
	DeviceName        string    `json:"deviceName"`
	IPAddress         string    `json:"ipAddress,omitempty"`
	UserAgent         string    `json:"userAgent,omitempty"`
	ExpiresAt         time.Time `json:"expiresAt"`
	RefreshExpiresAt  time.Time `json:"refreshExpiresAt"`
	LastSeenAt        time.Time `json:"lastSeenAt"`
	CreatedAt         time.Time `json:"createdAt"`
	TokenHash         []byte    `json:"-"`
	RefreshTokenHash  []byte    `json:"-"`
}

type OneTimeCodeAttempt struct {
	Valid    bool
	Attempts int32
	Consumed bool
}

type RefreshReplayResult struct {
	SessionID   uuid.UUID
	Found       bool
	WithinGrace bool
	Revoked     bool
}

type LoginInput struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	Phone       string `json:"phone,omitempty"`
	ClientType  string `json:"clientType"`
	DeviceName  string `json:"deviceName"`
	IPAddress   string `json:"-"`
	UserAgent   string `json:"-"`
	RememberMe  bool   `json:"rememberMe"`
	CaptchaID   string `json:"uuid,omitempty"`
	CaptchaCode string `json:"code,omitempty"`
	MFACode     string `json:"mfaCode,omitempty"`
	LoginType   string `json:"loginType,omitempty"`
	WXCode      string `json:"wxCode,omitempty"`
	SMSCode     string `json:"smsCode,omitempty"`
	Platform    string `json:"platform,omitempty"`
	AuthCode    string `json:"authCode,omitempty"`
}

type LoginResult struct {
	AccessToken  string    `json:"token"`
	RefreshToken string    `json:"refreshToken,omitempty"`
	CSRFToken    string    `json:"csrfToken,omitempty"`
	SessionID    uuid.UUID `json:"sessionId"`
	ExpiresAt    time.Time `json:"expiresAt"`
	User         User      `json:"user"`
	Access       Access    `json:"access"`
}

type RegisterInput struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	UserType    string `json:"userType"`
	IPAddress   string `json:"-"`
}

type Captcha struct {
	ID         uuid.UUID
	Image      string
	Piece      string
	TargetY    int
	TargetX    int
	Answer     int
	AnswerCode string
	Tolerance  int32
	Slider     bool
}

type SocialIdentity struct {
	Provider string
	Subject  string
	Nickname string
	Avatar   string
	Email    string
	Phone    string
	Raw      map[string]any
}

type SocialProvider interface {
	Exchange(ctx context.Context, code string) (SocialIdentity, error)
}

type CreateUserInput struct {
	Username            string      `json:"username"`
	Password            string      `json:"password"`
	DisplayName         string      `json:"displayName"`
	Email               string      `json:"email"`
	Phone               string      `json:"phone"`
	AvatarURL           string      `json:"avatarUrl"`
	UserType            string      `json:"userType"`
	Status              string      `json:"status"`
	Gender              int16       `json:"gender"`
	RoleIDs             []uuid.UUID `json:"roleIds"`
	DepartmentIDs       []uuid.UUID `json:"departmentIds"`
	PrimaryDepartmentID *uuid.UUID  `json:"primaryDepartmentId"`
	PositionIDs         []uuid.UUID `json:"positionIds"`
}

type UpdateUserInput struct {
	DisplayName         string      `json:"displayName"`
	Email               string      `json:"email"`
	Phone               string      `json:"phone"`
	AvatarURL           string      `json:"avatarUrl"`
	UserType            string      `json:"userType"`
	Status              string      `json:"status"`
	Gender              int16       `json:"gender"`
	Version             int64       `json:"version"`
	RoleIDs             []uuid.UUID `json:"roleIds"`
	DepartmentIDs       []uuid.UUID `json:"departmentIds"`
	PrimaryDepartmentID *uuid.UUID  `json:"primaryDepartmentId"`
	PositionIDs         []uuid.UUID `json:"positionIds"`
}

type UserFilter struct {
	Query     string
	Status    string
	UserType  string
	DataScope string
	ActorID   uuid.UUID
	Sort      string
	Page      int
	PageSize  int
}

type LoginEvent struct {
	UserID        *uuid.UUID
	Username      string
	ClientType    string
	Outcome       string
	FailureReason string
	IPAddress     string
	UserAgent     string
}

type Repository interface {
	FindUserByUsername(ctx context.Context, username string) (User, error)
	FindUserByPhone(ctx context.Context, phone string) (User, error)
	FindUserByID(ctx context.Context, id uuid.UUID) (User, error)
	FindUserByIDInDataScope(ctx context.Context, targetID, actorID uuid.UUID, dataScope string) (User, error)
	CountDepartmentsWithinScope(ctx context.Context, actorID uuid.UUID, dataScope string, departmentIDs []uuid.UUID) (int64, error)
	CountPrivilegedRoles(ctx context.Context, roleIDs []uuid.UUID) (int64, error)
	ListUsers(ctx context.Context, filter UserFilter) ([]User, int64, error)
	CreateUser(ctx context.Context, user User, roleIDs, departmentIDs, positionIDs []uuid.UUID, primaryDepartmentID *uuid.UUID) (User, error)
	UpdateUser(ctx context.Context, id uuid.UUID, input UpdateUserInput, passwordHash string) (User, error)
	SoftDeleteUser(ctx context.Context, id uuid.UUID) error
	UpdatePassword(ctx context.Context, id uuid.UUID, hash string) error
	ChangePasswordAndRevokeSessions(ctx context.Context, id uuid.UUID, hash string) error
	RecordFailedLogin(ctx context.Context, id uuid.UUID, maxAttempts int32, lockUntil time.Time) error
	ClearFailedLogin(ctx context.Context, id uuid.UUID) error
	CreateSession(ctx context.Context, session Session) error
	GetSessionByAccessHash(ctx context.Context, hash []byte) (Session, error)
	GetSessionByRefreshHash(ctx context.Context, hash []byte) (Session, error)
	RotateSession(ctx context.Context, oldRefreshHash []byte, session Session) error
	FindRefreshHistorySession(ctx context.Context, oldRefreshHash []byte) (uuid.UUID, error)
	ClassifyRefreshReplay(ctx context.Context, oldRefreshHash []byte, gracePeriod time.Duration) (RefreshReplayResult, error)
	CleanupRefreshHistory(ctx context.Context, retention time.Duration) (int64, error)
	TouchSession(ctx context.Context, id uuid.UUID) error
	RevokeSession(ctx context.Context, id uuid.UUID) error
	RevokeOwnedSession(ctx context.Context, id, userID uuid.UUID) error
	RevokeUserSessions(ctx context.Context, userID uuid.UUID) error
	ListUserSessions(ctx context.Context, userID uuid.UUID) ([]Session, error)
	ListActiveSessions(ctx context.Context) ([]Session, error)
	GetAccess(ctx context.Context, userID uuid.UUID) (Access, error)
	AccessCacheEpoch(ctx context.Context) (int64, error)
	SetUserRoles(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error
	RecordLoginEvent(ctx context.Context, event LoginEvent) error
	InsertOneTimeCode(ctx context.Context, purpose, subject string, codeHash []byte, expiresAt time.Time) error
	AttemptOneTimeCode(ctx context.Context, purpose, subject string, codeHash []byte) (OneTimeCodeAttempt, error)
	InsertCaptcha(ctx context.Context, id uuid.UUID, kind string, answerHash []byte, targetX int32, tolerance int32, expiresAt time.Time) error
	ConsumeCaptcha(ctx context.Context, id uuid.UUID, answerHash []byte, sliderX int32) (string, error)
	SetMFASecret(ctx context.Context, userID uuid.UUID, encrypted []byte) error
	EnableMFA(ctx context.Context, userID uuid.UUID) error
	DisableMFA(ctx context.Context, userID uuid.UUID) error
	MFASecret(ctx context.Context, userID uuid.UUID) ([]byte, bool, error)
	FindSocialUser(ctx context.Context, provider, subject string) (User, error)
	CreateSocialUser(ctx context.Context, user User, provider, subject string, metadata map[string]any) (User, error)
	LinkSocialIdentity(ctx context.Context, userID uuid.UUID, provider, subject string, metadata map[string]any) error
	UnlinkSocialIdentity(ctx context.Context, userID uuid.UUID, provider, subject string) error
	FindRoleByCode(ctx context.Context, code string) (uuid.UUID, error)
	EnsureBootstrapAdmin(ctx context.Context, username, passwordHash, email string) (uuid.UUID, error)
}

type RateLimiter interface {
	Allow(ctx context.Context, key string) (bool, error)
}

type Cache interface {
	Get(ctx context.Context, key string, destination any) (bool, error)
	Set(ctx context.Context, key string, value any) error
	DeletePrefix(ctx context.Context, suffix string) error
}
