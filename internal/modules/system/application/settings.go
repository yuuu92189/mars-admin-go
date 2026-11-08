package application

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"mars-admin-go/internal/modules/system/domain"
	"mars-admin-go/internal/platform/apperror"
	"mars-admin-go/internal/platform/security"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type SettingsService struct {
	repo      domain.Repository
	secretBox *security.SecretBox
	cache     domain.Cache
}

func NewSettingsService(repo domain.Repository, secretBox *security.SecretBox, caches ...domain.Cache) *SettingsService {
	service := &SettingsService{repo: repo, secretBox: secretBox}
	if len(caches) > 0 {
		service.cache = caches[0]
	}
	return service
}

func (s *SettingsService) Groups(ctx context.Context) ([]domain.SettingGroup, error) {
	var cached []domain.SettingGroup
	if s.cache != nil {
		if hit, _ := s.cache.Get(ctx, "settings:groups", &cached); hit {
			return cached, nil
		}
	}
	items, err := s.repo.SettingGroups(ctx)
	if err == nil && s.cache != nil {
		_ = s.cache.Set(ctx, "settings:groups", items)
	}
	return items, err
}

func (s *SettingsService) Group(ctx context.Context, code string, includeSensitive bool) (domain.SettingGroup, error) {
	key := "settings:group:" + code
	if includeSensitive {
		key += ":admin"
	}
	var cached domain.SettingGroup
	if s.cache != nil && !includeSensitive {
		if hit, _ := s.cache.Get(ctx, key, &cached); hit {
			return cached, nil
		}
	}
	group, err := s.repo.SettingGroupByCode(ctx, code, includeSensitive)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.SettingGroup{}, apperror.NotFound("Setting group")
	}
	if err != nil {
		return domain.SettingGroup{}, err
	}
	group.Values = make(map[string]any, len(group.Settings))
	for _, setting := range group.Settings {
		group.Values[setting.Key] = setting.Value
	}
	if s.cache != nil && !includeSensitive {
		_ = s.cache.Set(ctx, key, group)
	}
	return group, nil
}

func (s *SettingsService) Public(ctx context.Context) (map[string]any, error) {
	var cached map[string]any
	if s.cache != nil {
		if hit, _ := s.cache.Get(ctx, "settings:public", &cached); hit {
			return cached, nil
		}
	}
	values, err := s.repo.PublicSettings(ctx)
	if err == nil && s.cache != nil {
		_ = s.cache.Set(ctx, "settings:public", values)
	}
	return values, err
}

func (s *SettingsService) Invalidate(ctx context.Context) {
	if s.cache != nil {
		_ = s.cache.DeletePrefix(ctx, "settings:")
	}
}

func (s *SettingsService) Update(ctx context.Context, groupCode string, values map[string]any, actorID uuid.UUID) error {
	if len(values) == 0 {
		return apperror.Validation("settings", "At least one setting is required.")
	}
	updates := make([]domain.SettingUpdate, 0, len(values))
	for key, value := range values {
		key = strings.TrimSpace(key)
		if key == "" || len(key) > 120 {
			return apperror.Validation("settings", "Setting keys must contain 1 to 120 characters.")
		}
		valueType, raw, sensitive, err := encodeSetting(key, value)
		if err != nil {
			return apperror.Validation("settings."+key, err.Error())
		}
		if sensitive {
			if s.secretBox == nil {
				return apperror.New(apperror.KindNotConfigured, "settings_encryption_not_configured", "Sensitive settings encryption is not configured.")
			}
			encrypted, err := s.secretBox.Encrypt(raw)
			if err != nil {
				return err
			}
			raw = []byte(base64.RawStdEncoding.EncodeToString(encrypted))
		}
		updates = append(updates, domain.SettingUpdate{
			Key: key, ValueType: valueType, Raw: raw, Sensitive: sensitive,
		})
	}
	if err := s.repo.UpsertSettings(ctx, groupCode, updates, actorID); err != nil {
		return err
	}
	if s.cache != nil {
		_ = s.cache.DeletePrefix(ctx, "settings:")
	}
	return nil
}

func (s *SettingsService) Decrypt(setting domain.Setting, raw string) (any, error) {
	if !setting.Sensitive {
		return setting.Value, nil
	}
	if s.secretBox == nil {
		return nil, apperror.New(apperror.KindNotConfigured, "settings_encryption_not_configured", "Sensitive settings encryption is not configured.")
	}
	encoded, err := base64.RawStdEncoding.DecodeString(strings.TrimSpace(raw))
	if err != nil {
		encoded, err = base64.StdEncoding.DecodeString(strings.TrimSpace(raw))
	}
	if err != nil {
		return nil, err
	}
	plaintext, err := s.secretBox.Decrypt(encoded)
	if err != nil {
		return nil, err
	}
	return decodeByType(setting.ValueType, plaintext)
}

func encodeSetting(key string, value any) (string, []byte, bool, error) {
	sensitive := IsSensitiveSettingKey(key)
	switch typed := value.(type) {
	case string:
		return "string", []byte(typed), sensitive, nil
	case bool:
		return "boolean", []byte(strconv.FormatBool(typed)), sensitive, nil
	case float64:
		if typed != float64(int64(typed)) {
			return "", nil, false, errors.New("numeric settings must be integers")
		}
		return "integer", []byte(strconv.FormatInt(int64(typed), 10)), sensitive, nil
	case json.Number:
		if _, err := typed.Int64(); err != nil {
			return "", nil, false, errors.New("numeric settings must be integers")
		}
		return "integer", []byte(typed.String()), sensitive, nil
	default:
		encoded, err := json.Marshal(value)
		if err != nil {
			return "", nil, false, errors.New("setting value cannot be encoded as JSON")
		}
		return "json", encoded, sensitive, nil
	}
}

func IsSensitiveSettingKey(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	for _, marker := range []string{
		"secret", "password", "private_key", "privatekey", "api_key", "apikey",
		"access_key", "accesskey", "token", "credential", "signing_key", "signingkey",
	} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func decodeByType(valueType string, raw []byte) (any, error) {
	switch valueType {
	case "string", "secret":
		return string(raw), nil
	case "integer":
		return strconv.ParseInt(string(raw), 10, 64)
	case "boolean":
		return strconv.ParseBool(string(raw))
	case "json":
		var value any
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, err
		}
		return value, nil
	default:
		return string(raw), nil
	}
}
