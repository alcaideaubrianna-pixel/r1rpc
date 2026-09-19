package app

import (
	"errors"
	"testing"
	"time"
)

func TestDebugPairingSingleUseAndExpiry(t *testing.T) {
	now := time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)
	manager := NewDebugPairingManager(time.Minute)
	pairing, err := manager.Create(now)
	if err != nil {
		t.Fatal(err)
	}
	claimed, err := manager.Claim(pairing.Code, "ios-device", now.Add(time.Second))
	if err != nil || claimed.ClientID != "ios-device" {
		t.Fatalf("claim=%+v err=%v", claimed, err)
	}
	if _, err := manager.Claim(pairing.Code, "other", now.Add(2*time.Second)); !errors.Is(err, ErrDebugPairingClaimed) {
		t.Fatalf("second claim err=%v", err)
	}

	expired, err := manager.Create(now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Get(expired.ID, now.Add(2*time.Minute)); !errors.Is(err, ErrDebugPairingExpired) {
		t.Fatalf("expired get err=%v", err)
	}
}

func TestSecureTextEqual(t *testing.T) {
	if !secureTextEqual("dk_secret", "dk_secret") {
		t.Fatal("equal device keys did not match")
	}
	if secureTextEqual("", "") || secureTextEqual("wrong", "dk_secret") {
		t.Fatal("invalid device key matched")
	}
}
