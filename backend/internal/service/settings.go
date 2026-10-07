package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// SettingsService resolves runtime settings from the database, falling back to
// defaults seeded from the environment. Values are cached in-process for a few
// seconds so hot paths (rate limiting, checkout) do not query the database on
// every request, while an operator's change still propagates quickly across
// replicas.
type SettingsService struct {
	repo     port.SettingsRepository
	clock    port.Clock
	defaults map[string]string
	ttl      time.Duration

	mu       sync.RWMutex
	values   map[string]string
	loadedAt time.Time
}

func NewSettingsService(repo port.SettingsRepository, clock port.Clock, defaults map[string]string) *SettingsService {
	d := make(map[string]string, len(domain.SettingDefs))
	for _, def := range domain.SettingDefs {
		d[def.Key] = def.Default
	}
	for k, v := range defaults {
		d[k] = v
	}
	return &SettingsService{repo: repo, clock: clock, defaults: d, ttl: 5 * time.Second}
}

// load returns the merged defaults+overrides, refreshing at most every ttl.
func (s *SettingsService) load(ctx context.Context) map[string]string {
	s.mu.RLock()
	if s.values != nil && s.clock.Now().Sub(s.loadedAt) < s.ttl {
		v := s.values
		s.mu.RUnlock()
		return v
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.values != nil && s.clock.Now().Sub(s.loadedAt) < s.ttl {
		return s.values
	}
	merged := make(map[string]string, len(s.defaults))
	for k, v := range s.defaults {
		merged[k] = v
	}
	if s.repo != nil {
		if stored, err := s.repo.All(ctx); err == nil {
			for k, v := range stored {
				if _, ok := domain.SettingDefByKey(k); ok {
					merged[k] = v
				}
			}
		}
	}
	s.values = merged
	s.loadedAt = s.clock.Now()
	return merged
}

// String returns a string setting.
func (s *SettingsService) String(ctx context.Context, key string) string {
	return s.load(ctx)[key]
}

// Int returns an integer setting, or 0 if it is not a valid integer.
func (s *SettingsService) Int(ctx context.Context, key string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s.load(ctx)[key]))
	if err != nil {
		return 0
	}
	return n
}

// Bool returns a boolean setting.
func (s *SettingsService) Bool(ctx context.Context, key string) bool {
	b, err := strconv.ParseBool(strings.TrimSpace(s.load(ctx)[key]))
	if err != nil {
		return false
	}
	return b
}

// List returns every known setting with its current and default value.
func (s *SettingsService) List(ctx context.Context) []domain.Setting {
	values := s.load(ctx)
	out := make([]domain.Setting, 0, len(domain.SettingDefs))
	for _, def := range domain.SettingDefs {
		out = append(out, domain.Setting{
			Key: def.Key, Group: def.Group, Type: def.Type,
			Value: values[def.Key], Default: s.defaults[def.Key],
			Description: def.Description, Min: def.Min, Max: def.Max,
		})
	}
	return out
}

// Update validates and persists a batch of setting overrides.
func (s *SettingsService) Update(ctx context.Context, values map[string]string) error {
	if len(values) == 0 {
		return nil
	}
	clean := make(map[string]string, len(values))
	for k, raw := range values {
		def, ok := domain.SettingDefByKey(k)
		if !ok {
			return fmt.Errorf("%w: unknown setting %q", domain.ErrInvalidArgument, k)
		}
		v, err := validateSetting(def, raw)
		if err != nil {
			return err
		}
		clean[k] = v
	}
	if s.repo != nil {
		if err := s.repo.Upsert(ctx, clean); err != nil {
			return err
		}
	}
	s.mu.Lock()
	if s.values == nil {
		s.values = map[string]string{}
	}
	for k, v := range clean {
		s.values[k] = v
	}
	s.loadedAt = s.clock.Now()
	s.mu.Unlock()
	return nil
}

func validateSetting(def domain.SettingDef, raw string) (string, error) {
	v := strings.TrimSpace(raw)
	switch def.Type {
	case domain.SettingInt:
		n, err := strconv.Atoi(v)
		if err != nil {
			return "", fmt.Errorf("%w: %s must be an integer", domain.ErrInvalidArgument, def.Key)
		}
		if n < def.Min {
			return "", fmt.Errorf("%w: %s must be at least %d", domain.ErrInvalidArgument, def.Key, def.Min)
		}
		if def.Max > 0 && n > def.Max {
			return "", fmt.Errorf("%w: %s must be at most %d", domain.ErrInvalidArgument, def.Key, def.Max)
		}
		return strconv.Itoa(n), nil
	case domain.SettingBool:
		b, err := strconv.ParseBool(v)
		if err != nil {
			return "", fmt.Errorf("%w: %s must be a boolean", domain.ErrInvalidArgument, def.Key)
		}
		return strconv.FormatBool(b), nil
	default:
		return v, nil
	}
}
