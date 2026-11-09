package http

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	accessapp "mars-admin-go/internal/modules/access/application"
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

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var input identitydomain.LoginInput
	if !httpserver.DecodeJSON(w, r, &input) {
		return
	}
	h.loginWithInput(w, r, input)
}

func (h *Handler) loginWithInput(w http.ResponseWriter, r *http.Request, input identitydomain.LoginInput) {
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
	input.IPAddress = httpserver.ClientIP(r, h.config.Auth.TrustedProxyCIDRs)
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
		"user": user, "roles": actor.Roles, "permissions": actor.Permissions,
		"isSuperAdmin": actor.IsSuperAdmin, "menus": menus,
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
	departmentID, err := optionalQueryUUID(r.URL.Query().Get("departmentId"))
	if err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	positionID, err := optionalQueryUUID(r.URL.Query().Get("positionId"))
	if err != nil {
		httpserver.WriteError(w, r, bootstrapLogger(), err)
		return
	}
	users, total, err := h.identity.ListUsers(r.Context(), actor, identitydomain.UserFilter{
		Query: r.URL.Query().Get("keyword"), Status: r.URL.Query().Get("status"),
		UserType: r.URL.Query().Get("userType"), Sort: r.URL.Query().Get("sort"),
		DepartmentID: departmentID, PositionID: positionID,
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
	sameSite := h.sessionSameSite()
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
		Secure: h.config.Auth.CookieSecure, SameSite: sameSite, MaxAge: int(h.config.Auth.RefreshTTL.Seconds()),
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
			HttpOnly: cookie.httpOnly, Secure: h.config.Auth.CookieSecure, SameSite: h.sessionSameSite(),
		})
	}
}

func (h *Handler) sessionSameSite() http.SameSite {
	switch h.config.Auth.CookieSameSite {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
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

func optionalQueryUUID(value string) (*uuid.UUID, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func bootstrapLogger() *slog.Logger {
	return slog.Default()
}
