package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Wei-Shaw/nexus/internal/config"
)

const (
	OpsStorageStatusOK           = "ok"
	OpsStorageStatusUnavailable  = "unavailable"
	OpsStorageStatusUnconfigured = "unconfigured"

	opsStorageCollectionTimeout = 5 * time.Second
	opsStorageEnvPaths          = "OPS_STORAGE_PATHS"
)

var opsStorageKeySanitizer = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

type OpsStorageUsageResponse struct {
	GeneratedAt    time.Time              `json:"generated_at"`
	TotalUsedBytes int64                  `json:"total_used_bytes"`
	Items          []*OpsStorageUsageItem `json:"items"`
}

type OpsStorageUsageItem struct {
	Key       string `json:"key"`
	Label     string `json:"label"`
	Kind      string `json:"kind"`
	Source    string `json:"source"`
	Path      string `json:"path,omitempty"`
	UsedBytes *int64 `json:"used_bytes"`
	Status    string `json:"status"`
	Error     string `json:"error,omitempty"`
}

type opsStoragePathSpec struct {
	Key    string
	Label  string
	Kind   string
	Source string
	Path   string
}

func (s *OpsService) GetStorageUsage(ctx context.Context) (*OpsStorageUsageResponse, error) {
	if err := s.RequireMonitoringEnabled(ctx); err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, opsStorageCollectionTimeout)
	defer cancel()

	resp := &OpsStorageUsageResponse{
		GeneratedAt: time.Now().UTC(),
		Items:       make([]*OpsStorageUsageItem, 0, 4),
	}

	resp.Items = append(resp.Items, s.collectPostgresDatabaseSize(ctx))
	for _, spec := range s.storagePathSpecs() {
		resp.Items = append(resp.Items, collectOpsStoragePath(ctx, spec))
	}

	for _, item := range resp.Items {
		if item != nil && item.Status == OpsStorageStatusOK && item.UsedBytes != nil {
			resp.TotalUsedBytes += *item.UsedBytes
		}
	}
	return resp, nil
}

func (s *OpsService) collectPostgresDatabaseSize(ctx context.Context) *OpsStorageUsageItem {
	item := &OpsStorageUsageItem{
		Key:    "postgres_db",
		Label:  "PostgreSQL DB",
		Kind:   "database",
		Source: "postgresql",
		Status: OpsStorageStatusUnavailable,
	}
	if s == nil || s.opsRepo == nil {
		item.Error = "ops repository unavailable"
		return item
	}
	size, err := s.opsRepo.GetCurrentDatabaseSizeBytes(ctx)
	if err != nil {
		item.Error = truncateString(err.Error(), 256)
		return item
	}
	if size < 0 {
		size = 0
	}
	item.UsedBytes = &size
	item.Status = OpsStorageStatusOK
	return item
}

func (s *OpsService) storagePathSpecs() []opsStoragePathSpec {
	specs := []opsStoragePathSpec{{
		Key:    "app_data",
		Label:  "app_data",
		Kind:   "directory",
		Source: "filesystem",
		Path:   defaultOpsAppDataPath(),
	}}

	if s != nil && s.cfg != nil {
		for _, p := range s.cfg.Ops.Storage.Paths {
			specs = upsertOpsStoragePathSpec(specs, normalizeOpsStoragePathConfig(p, "config"))
		}
	}
	for _, p := range parseOpsStorageEnvPaths(os.Getenv(opsStorageEnvPaths)) {
		specs = upsertOpsStoragePathSpec(specs, p)
	}
	return specs
}

func normalizeOpsStoragePathConfig(p config.OpsStoragePathConfig, source string) opsStoragePathSpec {
	key := normalizeOpsStorageKey(p.Key)
	path := strings.TrimSpace(p.Path)
	if key == "" {
		key = normalizeOpsStorageKey(filepath.Base(filepath.Clean(path)))
	}
	label := strings.TrimSpace(p.Label)
	if label == "" {
		label = key
	}
	kind := strings.TrimSpace(p.Kind)
	if kind == "" {
		kind = "directory"
	}
	return opsStoragePathSpec{
		Key:    key,
		Label:  label,
		Kind:   kind,
		Source: source,
		Path:   path,
	}
}

func parseOpsStorageEnvPaths(raw string) []opsStoragePathSpec {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n'
	})
	out := make([]opsStoragePathSpec, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, path, ok := strings.Cut(part, "=")
		if !ok {
			path = part
			key = filepath.Base(filepath.Clean(path))
		}
		spec := normalizeOpsStoragePathConfig(config.OpsStoragePathConfig{
			Key:  key,
			Path: path,
		}, "env")
		out = append(out, spec)
	}
	return out
}

func upsertOpsStoragePathSpec(specs []opsStoragePathSpec, next opsStoragePathSpec) []opsStoragePathSpec {
	if strings.TrimSpace(next.Key) == "" {
		return specs
	}
	for i := range specs {
		if specs[i].Key == next.Key {
			specs[i] = next
			return specs
		}
	}
	return append(specs, next)
}

func defaultOpsAppDataPath() string {
	if dataDir := strings.TrimSpace(os.Getenv("DATA_DIR")); dataDir != "" {
		return dataDir
	}
	if _, err := os.Stat("/app/data"); err == nil {
		return "/app/data"
	}
	return "./data"
}

func normalizeOpsStorageKey(raw string) string {
	key := strings.ToLower(strings.TrimSpace(raw))
	key = strings.ReplaceAll(key, " ", "_")
	key = opsStorageKeySanitizer.ReplaceAllString(key, "_")
	key = strings.Trim(key, "_-")
	return key
}

func collectOpsStoragePath(ctx context.Context, spec opsStoragePathSpec) *OpsStorageUsageItem {
	item := &OpsStorageUsageItem{
		Key:    spec.Key,
		Label:  firstNonEmptyOpsStorageString(spec.Label, spec.Key),
		Kind:   firstNonEmptyOpsStorageString(spec.Kind, "directory"),
		Source: firstNonEmptyOpsStorageString(spec.Source, "filesystem"),
		Path:   strings.TrimSpace(spec.Path),
		Status: OpsStorageStatusUnavailable,
	}
	if item.Key == "" {
		item.Key = normalizeOpsStorageKey(filepath.Base(filepath.Clean(item.Path)))
	}
	if item.Path == "" {
		item.Status = OpsStorageStatusUnconfigured
		item.Error = "path is not configured"
		return item
	}

	size, err := calculateOpsPathSize(ctx, item.Path)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			item.Error = "storage scan timed out"
		} else if size > 0 {
			item.UsedBytes = &size
			item.Status = OpsStorageStatusOK
			item.Error = truncateString(err.Error(), 256)
			return item
		} else {
			item.Error = truncateString(err.Error(), 256)
		}
		return item
	}
	item.UsedBytes = &size
	item.Status = OpsStorageStatusOK
	return item
}

func calculateOpsPathSize(ctx context.Context, path string) (int64, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	info, err := os.Lstat(path)
	if err != nil {
		return 0, err
	}
	if !info.IsDir() {
		return info.Size(), nil
	}

	var total int64
	var firstErr error
	err = filepath.WalkDir(path, func(p string, d os.DirEntry, walkErr error) error {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		if walkErr != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("%s: %w", p, walkErr)
			}
			return nil
		}
		if d == nil || d.IsDir() {
			return nil
		}
		info, infoErr := d.Info()
		if infoErr != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("%s: %w", p, infoErr)
			}
			return nil
		}
		total += info.Size()
		return nil
	})
	if err != nil {
		return 0, err
	}
	return total, firstErr
}

func firstNonEmptyOpsStorageString(values ...string) string {
	for _, v := range values {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
