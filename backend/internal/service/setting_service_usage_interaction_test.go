package service

import (
	"context"
	"testing"
)

func TestUsageInteractionSettings_DefaultsAndPersistence(t *testing.T) {
	settings := SystemSettings{
		UsageInteractionRecordingEnabled: true,
		UsageInteractionStoreRawEnabled:  true,
		UsageInteractionRetentionDays:    13,
	}
	svc := NewSettingService(nil, nil)
	updates, err := svc.buildSystemSettingsUpdates(context.Background(), &settings)
	if err != nil {
		t.Fatal(err)
	}
	if updates[SettingKeyUsageInteractionRecordingEnabled] != "true" {
		t.Fatalf("recording setting not persisted: %v", updates)
	}
	if updates[SettingKeyUsageInteractionStoreRawEnabled] != "true" {
		t.Fatalf("raw setting not persisted: %v", updates)
	}
	if updates[SettingKeyUsageInteractionRetentionDays] != "13" {
		t.Fatalf("retention setting not persisted: %v", updates)
	}
}
