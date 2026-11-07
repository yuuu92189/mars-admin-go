package http

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	accessapp "mars-admin-go/internal/modules/access/application"
	accessdomain "mars-admin-go/internal/modules/access/domain"
	identityapp "mars-admin-go/internal/modules/identity/application"
	identitydomain "mars-admin-go/internal/modules/identity/domain"
	"mars-admin-go/internal/platform/config"
	"mars-admin-go/internal/platform/httpserver"
	"mars-admin-go/internal/platform/requestctx"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	identity identityapp.Service
	access   *accessapp.Service
	config   config.Config
	auth     func(http.Handler) http.Handler
}

func New(identity *identityapp.Service, access *accessapp.Service, cfg config.Config) *Handler {
	return &Handler{identity: *identity, access: access, config: cfg}
}

func (h *Handler) SetAuthMiddleware(middleware func(http.Handler) http.Handler) {
	h.auth = middleware
}

func (h *Handler) Routes() chi.Router {
	router := chi.NewRouter()
	router.Post("/login", h.login)
	router.Get("/captcha", h.captcha)
	router.Post("/sms-code", h.sendSMS)
	router.Post("/login/sms", h.loginSMS)
	router.Post("/social/{provider}", h.loginSocial)
	router.Post("/refresh", h.refresh)
	router.Post("/register", h.register)
	router.Get("/login-types", h.loginTypes)

	router.Group(func(protected chi.Router) {
		protected.Use(h.auth)
		protected.Post("/logout", h.logout)
		protected.Get("/me", h.me)
		protected.Get("/info", h.me)
		protected.Get("/profile", h.profile)
		protected.Put("/profile", h.updateProfile)
		protected.Post("/password", h.changePassword)
		protected.Post("/mfa/setup", h.setupMFA)
		protected.Post("/mfa/enable", h.enableMFA)
		protected.Post("/mfa/disable", h.disableMFA)
		protected.Post("/social-identities/unbind", h.unbindSocialIdentity)
		protected.Get("/sessions", h.sessions)
		protected.Delete("/sessions/{id}", h.revokeSession)
	})
	return router
}

func (h *Handler) AdminRoutes() chi.Router {
	router := chi.NewRouter()
	router.Use(h.auth)
	router.With(httpserver.RequirePermission("user:read")).Get("/", h.listUsers)
	router.With(httpserver.RequirePermission("user:read")).Get("/{id}", h.getUser)
	router.With(httpserver.RequirePermission("user:create")).Post("/", h.createUser)
	router.With(httpserver.RequirePermission("user:update")).Put("/{id}", h.updateUser)
	router.With(httpserver.RequirePermission("user:delete")).Delete("/{id}", h.deleteUser)
	router.With(httpserver.RequirePermission("user:update")).Post("/{id}/reset-password", h.resetPassword)
	router.With(httpserver.RequirePermission("user:update")).Post("/{id}/revoke-sessions", h.revokeUserSessions)
	return router
}

func (h *Handler) LegacyRoutes() chi.Router {
	router := chi.NewRouter()
	router.Get("/captcha", h.captcha)
	router.Post("/sms-code", h.sendSMS)
	router.Post("/login", h.login)
	router.Post("/register", h.register)
	router.Group(func(protected chi.Router) {
		protected.Use(h.auth)
		protected.Post("/logout", h.logout)
		protected.Get("/info", h.legacyMe)
		protected.Get("/profile", h.legacyProfile)
		protected.Put("/profile", h.legacyUpdateProfile)
		protected.Post("/password", h.changePassword)
	})
	return router
}

func (h *Handler) LegacyAdminRoutes() chi.Router {
	router := chi.NewRouter()
	router.Use(h.auth)
	router.With(httpserver.RequirePermission("user:read")).Get("/page", h.legacyListUsers)
	router.With(httpserver.RequirePermission("user:read")).Get("/{id}", h.legacyGetUser)
	router.With(httpserver.RequirePermission("user:create")).Post("/", h.legacyCreateUser)
	router.With(httpserver.RequirePermission("user:update")).Put("/", h.legacyUpdateUser)
	router.With(httpserver.RequirePermission("user:delete")).Delete("/batch", h.legacyDeleteUsers)
	router.With(httpserver.RequirePermission("user:delete")).Delete("/{id}", h.deleteUser)
	router.With(httpserver.RequirePermission("user:update")).Post("/{id}/reset-password", h.resetPassword)
	router.With(httpserver.RequirePermission("user:update")).Post("/{id}/approve", h.legacyApproveUser)
	router.With(httpserver.RequirePermission("user:update")).Post("/{id}/reject", h.legacyRejectUser)
	router.With(httpserver.RequirePermission("user:update")).Post("/{id}/quit", h.legacyQuitUser)
	return router
}

func (h *Handler) LegacyOnlineRoutes() chi.Router {
	router := chi.NewRouter()
	router.Use(h.auth)
	router.With(httpserver.RequirePermission("user:read")).Get("/list", h.legacyOnlineSessions)
	router.With(httpserver.RequirePermission("user:update")).Delete("/{tokenId}", h.legacyForceLogout)
	return router
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var input identitydomain.LoginInput
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	input.IPAddress = httpserver.ClientIP(r, h.config.Auth.TrustedProxyCIDRs)
	input.UserAgent = r.UserAgent()
	if input.ClientType == "" {
		input.ClientType = clientTypeFromPath(r.URL.Path)
	}
	var result identitydomain.LoginResult
	var err error
	switch strings.ToUpper(strings.TrimSpace(input.LoginType)) {
	case "SMS":
		result, err = h.identity.LoginSMS(
			r.Context(), input.Phone, firstNonEmpty(input.SMSCode, input.CaptchaCode),
			input.ClientType, input.DeviceName, input.IPAddress, input.UserAgent,
		)
	case "MINIPROGRAM", "WECHAT_MINIPROGRAM", "WECHAT_MINI_PROGRAM":
		result, err = h.identity.LoginSocial(
			r.Context(), "wechat_miniprogram", firstNonEmpty(input.WXCode, input.AuthCode, input.CaptchaCode),
			input.ClientType, input.DeviceName, input.IPAddress, input.UserAgent,
		)
	case "SOCIAL":
		provider := normalizeSocialProvider(input.Platform)
		result, err = h.identity.LoginSocial(
			r.Context(), provider, firstNonEmpty(input.AuthCode, input.CaptchaCode),
			input.ClientType, input.DeviceName, input.IPAddress, input.UserAgent,
		)
	default:
		result, err = h.identity.Login(r.Context(), input)
	}
	if err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	h.setSessionCookies(w, result)
	httpserver.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) loginSocial(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Code       string `json:"code"`
		AuthCode   string `json:"authCode"`
		ClientType string `json:"clientType"`
		DeviceName string `json:"deviceName"`
	}
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	result, err := h.identity.LoginSocial(
		r.Context(), normalizeSocialProvider(chi.URLParam(r, "provider")),
		firstNonEmpty(input.AuthCode, input.Code), firstNonEmpty(input.ClientType, "app"),
		input.DeviceName, httpserver.ClientIP(r, h.config.Auth.TrustedProxyCIDRs), r.UserAgent(),
	)
	if err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	h.setSessionCookies(w, result)
	httpserver.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) unbindSocialIdentity(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Provider string `json:"provider"`
		Subject  string `json:"subject"`
	}
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	if err := h.identity.UnbindSocialIdentity(r.Context(), actor, input.Provider, input.Subject); err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) loginSMS(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Phone      string `json:"phone"`
		Code       string `json:"code"`
		SMS        string `json:"smsCode"`
		ClientType string `json:"clientType"`
		DeviceName string `json:"deviceName"`
	}
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	code := input.Code
	if code == "" {
		code = input.SMS
	}
	result, err := h.identity.LoginSMS(
		r.Context(), input.Phone, code, firstNonEmpty(input.ClientType, clientTypeFromPath(r.URL.Path)),
		input.DeviceName, httpserver.ClientIP(r, h.config.Auth.TrustedProxyCIDRs), r.UserAgent(),
	)
	if err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	h.setSessionCookies(w, result)
	httpserver.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) sendSMS(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Phone      string `json:"phone"`
		DeviceName string `json:"deviceName"`
	}
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	if err := h.identity.SendLoginCode(
		r.Context(), input.Phone, httpserver.ClientIP(r, h.config.Auth.TrustedProxyCIDRs),
		input.DeviceName, r.UserAgent(),
	); err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	httpserver.WriteJSON(w, http.StatusAccepted, map[string]string{"status": "sent"})
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	token := ""
	if cookie, err := r.Cookie("mars_refresh"); err == nil {
		token = cookie.Value
	}
	if token == "" {
		var input struct {
			RefreshToken string `json:"refreshToken"`
		}
		if !httpserver.DecodeJSON(w, r, &input) {
			return
		}
		token = input.RefreshToken
	}
	result, err := h.identity.Refresh(r.Context(), token)
	if err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	h.setSessionCookies(w, result)
	httpserver.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	actor, _ := requestctx.ActorFrom(r.Context())
	if err := h.identity.Logout(r.Context(), actor); err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	h.clearSessionCookies(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var input identitydomain.RegisterInput
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	user, err := h.identity.Register(r.Context(), input)
	if err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	httpserver.WriteJSON(w, http.StatusCreated, user)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	actor, _ := requestctx.ActorFrom(r.Context())
	user, err := h.identity.Profile(r.Context(), actor.UserID)
	if err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	menus, err := h.access.Menus(r.Context(), actor.UserID, actor.IsSuperAdmin)
	if err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, map[string]any{
		"user": user, "roles": actor.Roles, "permissions": actor.Permissions, "menus": menus,
	})
}

func (h *Handler) legacyMe(w http.ResponseWriter, r *http.Request) {
	actor, _ := requestctx.ActorFrom(r.Context())
	user, err := h.identity.Profile(r.Context(), actor.UserID)
	if err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	menus, err := h.access.Menus(r.Context(), actor.UserID, actor.IsSuperAdmin)
	if err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, map[string]any{
		"user":        legacyUser(user),
		"roles":       actor.Roles,
		"permissions": actor.Permissions,
		"menus":       legacyMenus(menus),
	})
}

func (h *Handler) profile(w http.ResponseWriter, r *http.Request) {
	actor, _ := requestctx.ActorFrom(r.Context())
	user, err := h.identity.Profile(r.Context(), actor.UserID)
	if err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, user)
}

func (h *Handler) legacyProfile(w http.ResponseWriter, r *http.Request) {
	actor, _ := requestctx.ActorFrom(r.Context())
	user, err := h.identity.Profile(r.Context(), actor.UserID)
	if err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, legacyUser(user))
}

func (h *Handler) legacyUpdateProfile(w http.ResponseWriter, r *http.Request) {
	var input legacyUserPayload
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	current, err := h.identity.Profile(r.Context(), actor.UserID)
	if err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	nickname := firstNonEmpty(input.Nickname, input.DisplayName, current.DisplayName)
	email := input.Email
	if email == "" {
		email = current.Email
	}
	phone := input.Phone
	if phone == "" {
		phone = current.Phone
	}
	version := input.Version
	if version == 0 {
		version = current.Version
	}
	updated, err := h.identity.UpdateUser(r.Context(), actor, actor.UserID, identitydomain.UpdateUserInput{
		DisplayName: nickname, Email: email, Phone: phone,
		AvatarURL: firstNonEmpty(input.Avatar, input.AvatarURL, current.AvatarURL),
		UserType:  current.UserType,
		Status:    current.Status, Gender: input.Gender, Version: version,
		RoleIDs: current.RoleIDs, DepartmentIDs: current.DepartmentIDs, PositionIDs: current.PositionIDs,
	})
	if err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, legacyUser(updated))
}

func (h *Handler) updateProfile(w http.ResponseWriter, r *http.Request) {
	var input identitydomain.UpdateUserInput
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	user, err := h.identity.UpdateProfile(r.Context(), actor.UserID, input)
	if err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, user)
}

func (h *Handler) changePassword(w http.ResponseWriter, r *http.Request) {
	var input struct {
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	}
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	if err := h.identity.ChangePassword(r.Context(), actor.UserID, input.OldPassword, input.NewPassword); err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	h.clearSessionCookies(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) setupMFA(w http.ResponseWriter, r *http.Request) {
	actor, _ := requestctx.ActorFrom(r.Context())
	setup, err := h.identity.SetupMFA(r.Context(), actor.UserID)
	if err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, setup)
}

func (h *Handler) enableMFA(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Code string `json:"code"`
	}
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	if err := h.identity.EnableMFA(r.Context(), actor.UserID, input.Code); err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) disableMFA(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Password string `json:"password"`
		Code     string `json:"code"`
	}
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	if err := h.identity.DisableMFA(r.Context(), actor.UserID, input.Password, input.Code); err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) sessions(w http.ResponseWriter, r *http.Request) {
	actor, _ := requestctx.ActorFrom(r.Context())
	sessions, err := h.identity.ListSessions(r.Context(), actor.UserID)
	if err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, sessions)
}

func (h *Handler) revokeSession(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	if err := h.identity.RevokeOwnedSession(r.Context(), actor, id); err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := httpserver.ParsePagination(r, 100)
	if err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	users, total, err := h.identity.ListUsers(r.Context(), actor, identitydomain.UserFilter{
		Query: r.URL.Query().Get("keyword"), Status: r.URL.Query().Get("status"),
		UserType: r.URL.Query().Get("userType"), Sort: r.URL.Query().Get("sort"),
		Page: page, PageSize: pageSize,
	})
	if err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, httpserver.NewPage(users, total, page, pageSize))
}

func (h *Handler) getUser(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	user, err := h.identity.GetUser(r.Context(), actor, id)
	if err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, user)
}

func (h *Handler) createUser(w http.ResponseWriter, r *http.Request) {
	var input identitydomain.CreateUserInput
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	user, err := h.identity.CreateUserAsActor(r.Context(), actor, input)
	if err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	httpserver.WriteJSON(w, http.StatusCreated, user)
}

func (h *Handler) updateUser(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	var input identitydomain.UpdateUserInput
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	user, err := h.identity.UpdateUser(r.Context(), actor, id, input)
	if err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, user)
}

func (h *Handler) deleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	if err := h.identity.DeleteUser(r.Context(), actor, id); err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) resetPassword(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	password, err := h.identity.ResetPassword(r.Context(), actor, id, "")
	if err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, map[string]string{"temporaryPassword": password})
}

func (h *Handler) revokeUserSessions(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	if err := h.identity.RevokeUserSessions(r.Context(), actor, id); err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) legacyOnlineSessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := h.identity.ListActiveSessions(r.Context())
	if err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	result := make([]map[string]any, 0, len(sessions))
	names := make(map[uuid.UUID]string)
	for _, session := range sessions {
		name, ok := names[session.UserID]
		if !ok {
			user, profileErr := h.identity.Profile(r.Context(), session.UserID)
			if profileErr == nil {
				name = user.Username
			}
			names[session.UserID] = name
		}
		browser, osName := legacyBrowserAndOS(session.UserAgent)
		result = append(result, map[string]any{
			"tokenId": session.ID, "loginName": name, "deptName": "",
			"ipaddr": session.IPAddress, "loginLocation": "",
			"browser": browser, "os": osName, "status": 1,
			"loginTime":      session.CreatedAt.Format("2006-01-02 15:04:05"),
			"lastAccessTime": session.LastSeenAt.Format("2006-01-02 15:04:05"),
		})
	}
	httpserver.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) legacyForceLogout(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "tokenId")
	if err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	if err := h.identity.RevokeSession(r.Context(), id); err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func legacyBrowserAndOS(userAgent string) (string, string) {
	browser := "Unknown"
	osName := "Unknown"
	switch {
	case strings.Contains(userAgent, "Edg/"):
		browser = "Edge"
	case strings.Contains(userAgent, "Chrome/"):
		browser = "Chrome"
	case strings.Contains(userAgent, "Firefox/"):
		browser = "Firefox"
	case strings.Contains(userAgent, "Safari/"):
		browser = "Safari"
	}
	switch {
	case strings.Contains(userAgent, "Windows"):
		osName = "Windows"
	case strings.Contains(userAgent, "Android"):
		osName = "Android"
	case strings.Contains(userAgent, "iPhone"), strings.Contains(userAgent, "iPad"):
		osName = "iOS"
	case strings.Contains(userAgent, "Mac OS"):
		osName = "macOS"
	case strings.Contains(userAgent, "Linux"):
		osName = "Linux"
	}
	return browser, osName
}

type legacyUserPayload struct {
	ID          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	Password    string    `json:"password"`
	Nickname    string    `json:"nickname"`
	DisplayName string    `json:"displayName"`
	Email       string    `json:"email"`
	Phone       string    `json:"phone"`
	Avatar      string    `json:"avatar"`
	AvatarURL   string    `json:"avatarUrl"`
	Gender      int16     `json:"gender"`
	Status      any       `json:"status"`
	UserType    string    `json:"userType"`
	Version     int64     `json:"version"`
	Remark      string    `json:"remark"`
}

type legacyUserMutation struct {
	User      legacyUserPayload `json:"user"`
	RoleIDs   []uuid.UUID       `json:"roleIds"`
	PostIDs   []uuid.UUID       `json:"postIds"`
	DeptIDs   []uuid.UUID       `json:"deptIds"`
	MenuIDs   []uuid.UUID       `json:"menuIds"`
	Version   int64             `json:"version"`
	Password  string            `json:"password"`
	Nickname  string            `json:"nickname"`
	Status    any               `json:"status"`
	Username  string            `json:"username"`
	Display   string            `json:"displayName"`
	AvatarURL string            `json:"avatar"`
}

func (h *Handler) legacyListUsers(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := httpserver.ParsePagination(r, 100)
	if err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	query := firstNonEmpty(r.URL.Query().Get("username"), r.URL.Query().Get("keyword"), r.URL.Query().Get("query"))
	status := legacyStatusName(r.URL.Query().Get("status"))
	actor, _ := requestctx.ActorFrom(r.Context())
	users, total, err := h.identity.ListUsers(r.Context(), actor, identitydomain.UserFilter{
		Query: query, Status: status, UserType: r.URL.Query().Get("userType"),
		Sort: r.URL.Query().Get("sort"), Page: page, PageSize: pageSize,
	})
	if err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	items := make([]map[string]any, 0, len(users))
	for _, user := range users {
		items = append(items, legacyUser(user))
	}
	httpserver.WriteJSON(w, http.StatusOK, map[string]any{
		"list": items, "total": total, "page": page, "pageSize": pageSize,
	})
}

func (h *Handler) legacyGetUser(w http.ResponseWriter, r *http.Request) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	user, err := h.identity.GetUser(r.Context(), actor, id)
	if err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, map[string]any{
		"user": legacyUser(user), "roleIds": user.RoleIDs, "postIds": user.PositionIDs,
	})
}

func (h *Handler) legacyCreateUser(w http.ResponseWriter, r *http.Request) {
	var input legacyUserMutation
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	payload := input.User
	if payload.Username == "" {
		payload.Username = input.Username
	}
	if payload.Password == "" {
		payload.Password = input.Password
	}
	if payload.Nickname == "" {
		payload.Nickname = firstNonEmpty(input.Nickname, payload.DisplayName, input.Display, payload.Username)
	}
	if payload.Status == nil {
		payload.Status = input.Status
	}
	if payload.UserType == "" {
		payload.UserType = "admin"
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	user, err := h.identity.CreateUserAsActor(r.Context(), actor, identitydomain.CreateUserInput{
		Username: payload.Username, Password: payload.Password,
		DisplayName: payload.Nickname, Email: payload.Email, Phone: payload.Phone,
		AvatarURL: firstNonEmpty(payload.Avatar, payload.AvatarURL, input.AvatarURL),
		UserType:  payload.UserType, Status: legacyStatusNameFromAny(payload.Status),
		Gender: payload.Gender, RoleIDs: input.RoleIDs, DepartmentIDs: input.DeptIDs,
		PositionIDs: input.PostIDs,
	})
	if err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	httpserver.WriteJSON(w, http.StatusCreated, legacyUser(user))
}

func (h *Handler) legacyUpdateUser(w http.ResponseWriter, r *http.Request) {
	var input legacyUserMutation
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	if input.User.ID == uuid.Nil {
		httpserver.WriteJSON(w, http.StatusBadRequest, map[string]string{"detail": "User id is required."})
		return
	}
	payload := input.User
	if payload.Nickname == "" {
		payload.Nickname = firstNonEmpty(input.Nickname, payload.DisplayName, input.Display)
	}
	if payload.Status == nil {
		payload.Status = input.Status
	}
	version := payload.Version
	if version == 0 {
		version = input.Version
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	user, err := h.identity.UpdateUser(r.Context(), actor, input.User.ID, identitydomain.UpdateUserInput{
		DisplayName: payload.Nickname, Email: payload.Email, Phone: payload.Phone,
		AvatarURL: firstNonEmpty(payload.Avatar, payload.AvatarURL, input.AvatarURL),
		UserType:  payload.UserType, Status: legacyStatusNameFromAny(payload.Status),
		Gender: payload.Gender, Version: version, RoleIDs: input.RoleIDs,
		DepartmentIDs: input.DeptIDs, PositionIDs: input.PostIDs,
	})
	if err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, legacyUser(user))
}

func (h *Handler) legacyDeleteUsers(w http.ResponseWriter, r *http.Request) {
	var ids []uuid.UUID
	if !httpserver.DecodeJSON(w, r, &ids) {
		return
	}
	for _, id := range ids {
		actor, _ := requestctx.ActorFrom(r.Context())
		if err := h.identity.DeleteUser(r.Context(), actor, id); err != nil {
			httpserver.WriteError(w, r, bootstrapLogger(), err)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) legacyApproveUser(w http.ResponseWriter, r *http.Request) {
	h.legacySetUserStatus(w, r, "active")
}

func (h *Handler) legacyRejectUser(w http.ResponseWriter, r *http.Request) {
	h.legacySetUserStatus(w, r, "rejected")
}

func (h *Handler) legacyQuitUser(w http.ResponseWriter, r *http.Request) {
	h.legacySetUserStatus(w, r, "left")
}

func (h *Handler) legacySetUserStatus(w http.ResponseWriter, r *http.Request, status string) {
	id, err := httpserver.PathUUID(r, "id")
	if err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	actor, _ := requestctx.ActorFrom(r.Context())
	user, err := h.identity.GetUser(r.Context(), actor, id)
	if err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	user, err = h.identity.UpdateUser(r.Context(), actor, id, identitydomain.UpdateUserInput{
		DisplayName: user.DisplayName, Email: user.Email, Phone: user.Phone,
		AvatarURL: user.AvatarURL,
		UserType:  user.UserType, Status: status, Gender: user.Gender, Version: user.Version,
		RoleIDs: user.RoleIDs, DepartmentIDs: user.DepartmentIDs, PositionIDs: user.PositionIDs,
	})
	if err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, legacyUser(user))
}

func legacyUser(user identitydomain.User) map[string]any {
	item := map[string]any{
		"id": user.ID, "username": user.Username, "nickname": user.DisplayName,
		"avatar": user.AvatarURL, "email": user.Email, "phone": user.Phone,
		"gender": user.Gender, "status": legacyStatusCode(user.Status),
		"userType": user.UserType, "isQuit": boolInt(user.Status == "left"),
		"version": user.Version, "createTime": user.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	if len(user.DepartmentIDs) > 0 {
		item["deptId"] = user.DepartmentIDs[0]
	}
	return item
}

func legacyMenus(menus []accessdomain.Menu) []map[string]any {
	items := make([]map[string]any, 0, len(menus))
	for _, menu := range menus {
		if menu.ParentID == nil && menu.Path == "dashboard" {
			continue
		}
		item := map[string]any{
			"id": menu.ID, "name": menu.Name, "type": menu.Type, "path": menu.Path,
			"component": menu.Component, "permission": menu.Permission, "icon": menu.Icon,
			"sort": menu.Sort, "visible": boolInt(menu.Visible), "status": 1,
			"isFrame": boolInt(menu.IsFrame),
		}
		if menu.ParentID != nil {
			item["parentId"] = *menu.ParentID
		} else {
			item["parentId"] = 0
		}
		if len(menu.Children) > 0 {
			item["children"] = legacyMenus(menu.Children)
		}
		items = append(items, item)
	}
	return items
}

func legacyStatusName(value string) string {
	switch value {
	case "1":
		return "active"
	case "0":
		return "suspended"
	case "2":
		return "pending"
	case "3":
		return "rejected"
	default:
		return value
	}
}

func legacyStatusNameFromAny(value any) string {
	switch typed := value.(type) {
	case string:
		return legacyStatusName(typed)
	case float64:
		return legacyStatusName(strconv.Itoa(int(typed)))
	case int:
		return legacyStatusName(strconv.Itoa(typed))
	default:
		return "active"
	}
}

func legacyStatusCode(value string) int {
	switch value {
	case "active":
		return 1
	case "pending":
		return 2
	case "rejected":
		return 3
	default:
		return 0
	}
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func (h *Handler) loginTypes(w http.ResponseWriter, _ *http.Request) {
	httpserver.WriteJSON(w, http.StatusOK, []string{"password", "sms", "social"})
}

func (h *Handler) captcha(w http.ResponseWriter, r *http.Request) {
	captcha, err := h.identity.IssueCaptcha(r.Context(), r.URL.Query().Get("type"))
	if err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, map[string]any{
		"uuid": captcha.ID, "img": captcha.Image, "piece": captcha.Piece,
		"targetY": captcha.TargetY, "slider": captcha.Slider,
	})
}

func (h *Handler) setSessionCookies(w http.ResponseWriter, result identitydomain.LoginResult) {
	sameSite := http.SameSiteLaxMode
	switch h.config.Auth.CookieSameSite {
	case "strict":
		sameSite = http.SameSiteStrictMode
	case "none":
		sameSite = http.SameSiteNoneMode
	}
	maxAge := int(time.Until(result.ExpiresAt).Seconds())
	http.SetCookie(w, &http.Cookie{ // #nosec G124 -- HttpOnly, Secure and SameSite are explicitly configured below.
		Name: "mars_session", Value: result.AccessToken, Path: "/", HttpOnly: true,
		Secure: h.config.Auth.CookieSecure, SameSite: sameSite, MaxAge: maxAge,
	})
	http.SetCookie(w, &http.Cookie{ // #nosec G124 -- HttpOnly, Secure and SameSite are explicitly configured below.
		Name: "mars_refresh", Value: result.RefreshToken, Path: "/api/v1/auth/refresh", HttpOnly: true,
		Secure: h.config.Auth.CookieSecure, SameSite: sameSite, MaxAge: int(h.config.Auth.RefreshTTL.Seconds()),
	})
	http.SetCookie(w, &http.Cookie{ // #nosec G124 -- HttpOnly, Secure and SameSite are explicitly configured below.
		Name: "mars_csrf", Value: result.CSRFToken, Path: "/", HttpOnly: false,
		Secure: h.config.Auth.CookieSecure, SameSite: sameSite, MaxAge: maxAge,
	})
}

func (h *Handler) clearSessionCookies(w http.ResponseWriter) {
	cookies := []struct {
		name     string
		path     string
		httpOnly bool
	}{
		{name: "mars_session", path: "/", httpOnly: true},
		{name: "mars_refresh", path: "/api/v1/auth/refresh", httpOnly: true},
		{name: "mars_csrf", path: "/", httpOnly: false},
	}
	for _, cookie := range cookies {
		http.SetCookie(w, &http.Cookie{ // #nosec G124 -- deletion mirrors the original cookie scope.
			Name: cookie.name, Value: "", Path: cookie.path, MaxAge: -1,
			HttpOnly: cookie.httpOnly, Secure: h.config.Auth.CookieSecure, SameSite: http.SameSiteLaxMode,
		})
	}
}

func clientTypeFromPath(path string) string {
	switch {
	case strings.Contains(path, "/app/"):
		return "app"
	case strings.Contains(path, "/web/"):
		return "web"
	default:
		return "admin"
	}
}

func normalizeSocialProvider(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "miniprogram", "wechat", "wechat-miniprogram", "wechat_miniprogram", "wechat_mini_program":
		return "wechat_miniprogram"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func bootstrapLogger() *slog.Logger {
	return slog.Default()
}
