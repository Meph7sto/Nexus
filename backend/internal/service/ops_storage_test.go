package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Wei-Shaw/nexus/internal/config"
)

func TestOpsServiceGetStorageUsageIncludesDatabaseAndConfiguredPaths(t *testing.T) {
	root := t.TempDir()
	appData := filepath.Join(root, "app-data")
	postgresData := filepath.Join(root, "postgres-data")
	dockerData := filepath.Join(root, "docker")
	mustWriteTestFile(t, filepath.Join(appData, "a.bin"), 10)
	mustWriteTestFile(t, filepath.Join(postgresData, "base", "b.bin"), 20)
	mustWriteTestFile(t, filepath.Join(dockerData, "overlay2", "c.bin"), 30)

	t.Setenv("DATA_DIR", appData)
	t.Setenv("OPS_STORAGE_PATHS", "docker="+dockerData)

	repo := &opsRepoMock{
		GetCurrentDatabaseSizeBytesFn: func(ctx context.Context) (int64, error) {
			return 40, nil
		},
	}
	svc := NewOpsService(repo, nil, &config.Config{
		Ops: config.OpsConfig{
			Enabled: true,
			Storage: config.OpsStorageConfig{
				Paths: []config.OpsStoragePathConfig{
					{Key: "postgres_data", Label: "postgres_data", Path: postgresData},
				},
			},
		},
	}, nil, nil, nil, nil, nil, nil, nil, nil)

	out, err := svc.GetStorageUsage(context.Background())
	if err != nil {
		t.Fatalf("GetStorageUsage() error: %v", err)
	}

	items := storageItemsByKey(out.Items)
	assertStorageItem(t, items["postgres_db"], OpsStorageStatusOK, 40)
	assertStorageItem(t, items["app_data"], OpsStorageStatusOK, 10)
	assertStorageItem(t, items["postgres_data"], OpsStorageStatusOK, 20)
	assertStorageItem(t, items["docker"], OpsStorageStatusOK, 30)
	if out.TotalUsedBytes != 100 {
		t.Fatalf("TotalUsedBytes=%d, want 100", out.TotalUsedBytes)
	}
}

func TestOpsServiceGetStorageUsageBestEffortForUnavailableItems(t *testing.T) {
	root := t.TempDir()
	appData := filepath.Join(root, "app-data")
	mustWriteTestFile(t, filepath.Join(appData, "a.bin"), 12)
	missing := filepath.Join(root, "missing")

	t.Setenv("DATA_DIR", appData)
	repo := &opsRepoMock{
		GetCurrentDatabaseSizeBytesFn: func(ctx context.Context) (int64, error) {
			return 0, errors.New("db unavailable")
		},
	}
	svc := NewOpsService(repo, nil, &config.Config{
		Ops: config.OpsConfig{
			Enabled: true,
			Storage: config.OpsStorageConfig{
				Paths: []config.OpsStoragePathConfig{
					{Key: "postgres_data", Path: missing},
				},
			},
		},
	}, nil, nil, nil, nil, nil, nil, nil, nil)

	out, err := svc.GetStorageUsage(context.Background())
	if err != nil {
		t.Fatalf("GetStorageUsage() error: %v", err)
	}

	items := storageItemsByKey(out.Items)
	if items["postgres_db"].Status != OpsStorageStatusUnavailable || items["postgres_db"].Error == "" {
		t.Fatalf("postgres_db item should be unavailable with an error: %+v", items["postgres_db"])
	}
	if items["postgres_data"].Status != OpsStorageStatusUnavailable || items["postgres_data"].Error == "" {
		t.Fatalf("postgres_data item should be unavailable with an error: %+v", items["postgres_data"])
	}
	assertStorageItem(t, items["app_data"], OpsStorageStatusOK, 12)
	if out.TotalUsedBytes != 12 {
		t.Fatalf("TotalUsedBytes=%d, want 12", out.TotalUsedBytes)
	}
}

func TestParseOpsStorageEnvPaths(t *testing.T) {
	specs := parseOpsStorageEnvPaths("postgres_data=/var/lib/postgresql/data;docker=/var/lib/docker\n/cache")
	if len(specs) != 3 {
		t.Fatalf("len(specs)=%d, want 3", len(specs))
	}
	if specs[0].Key != "postgres_data" || specs[0].Path != "/var/lib/postgresql/data" || specs[0].Source != "env" {
		t.Fatalf("unexpected postgres spec: %+v", specs[0])
	}
	if specs[1].Key != "docker" || specs[1].Path != "/var/lib/docker" {
		t.Fatalf("unexpected docker spec: %+v", specs[1])
	}
	if specs[2].Key != "cache" || specs[2].Path != "/cache" {
		t.Fatalf("unexpected implicit-key spec: %+v", specs[2])
	}
}

func TestOpsServiceGetStorageUsageMonitoringDisabled(t *testing.T) {
	svc := NewOpsService(&opsRepoMock{}, nil, &config.Config{
		Ops: config.OpsConfig{Enabled: false},
	}, nil, nil, nil, nil, nil, nil, nil, nil)

	_, err := svc.GetStorageUsage(context.Background())
	if err == nil {
		t.Fatalf("expected disabled error")
	}
}

func mustWriteTestFile(t *testing.T, path string, size int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, make([]byte, size), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}

func storageItemsByKey(items []*OpsStorageUsageItem) map[string]*OpsStorageUsageItem {
	out := make(map[string]*OpsStorageUsageItem, len(items))
	for _, item := range items {
		out[item.Key] = item
	}
	return out
}

func assertStorageItem(t *testing.T, item *OpsStorageUsageItem, status string, usedBytes int64) {
	t.Helper()
	if item == nil {
		t.Fatalf("expected storage item")
	}
	if item.Status != status {
		t.Fatalf("%s status=%q, want %q", item.Key, item.Status, status)
	}
	if item.UsedBytes == nil {
		t.Fatalf("%s UsedBytes=nil, want %d", item.Key, usedBytes)
	}
	if *item.UsedBytes != usedBytes {
		t.Fatalf("%s UsedBytes=%d, want %d", item.Key, *item.UsedBytes, usedBytes)
	}
}
