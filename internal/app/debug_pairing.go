package app

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

var (
	ErrDebugPairingNotFound = errors.New("调试配对不存在")
	ErrDebugPairingExpired  = errors.New("调试配对已过期")
	ErrDebugPairingClaimed  = errors.New("调试配对已被使用")
)

type DebugPairing struct {
	ID        string    `json:"id"`
	ExpiresAt time.Time `json:"expiresAt"`
	ClaimedAt time.Time `json:"claimedAt,omitempty"`
	ClientID  string    `json:"clientId,omitempty"`
	Code      string    `json:"-"`
	codeHash  [32]byte
}

type DebugPairingManager struct {
	mu       sync.Mutex
	ttl      time.Duration
	pairings map[string]*DebugPairing
}

func NewDebugPairingManager(ttl time.Duration) *DebugPairingManager {
	return &DebugPairingManager{ttl: ttl, pairings: make(map[string]*DebugPairing)}
}

func (m *DebugPairingManager) Create(now time.Time) (*DebugPairing, error) {
	id, err := randomHex(16)
	if err != nil {
		return nil, err
	}
	code, err := randomHex(32)
	if err != nil {
		return nil, err
	}
	pairing := &DebugPairing{ID: id, Code: code, ExpiresAt: now.Add(m.ttl), codeHash: sha256.Sum256([]byte(code))}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cleanupLocked(now)
	m.pairings[id] = pairing
	copy := *pairing
	return &copy, nil
}

func (m *DebugPairingManager) Get(id string, now time.Time) (*DebugPairing, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	pairing, ok := m.pairings[id]
	if !ok {
		return nil, ErrDebugPairingNotFound
	}
	if now.After(pairing.ExpiresAt) {
		delete(m.pairings, id)
		return nil, ErrDebugPairingExpired
	}
	copy := *pairing
	return &copy, nil
}

func (m *DebugPairingManager) Claim(code, clientID string, now time.Time) (*DebugPairing, error) {
	wanted := sha256.Sum256([]byte(code))
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, pairing := range m.pairings {
		if now.After(pairing.ExpiresAt) {
			delete(m.pairings, id)
			continue
		}
		if subtle.ConstantTimeCompare(wanted[:], pairing.codeHash[:]) != 1 {
			continue
		}
		if !pairing.ClaimedAt.IsZero() {
			return nil, ErrDebugPairingClaimed
		}
		pairing.ClaimedAt = now
		pairing.ClientID = clientID
		copy := *pairing
		return &copy, nil
	}
	return nil, ErrDebugPairingNotFound
}

func (m *DebugPairingManager) cleanupLocked(now time.Time) {
	for id, pairing := range m.pairings {
		if now.After(pairing.ExpiresAt) {
			delete(m.pairings, id)
		}
	}
}

func randomHex(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func secureTextEqual(got, want string) bool {
	gotHash := sha256.Sum256([]byte(got))
	wantHash := sha256.Sum256([]byte(want))
	return subtle.ConstantTimeCompare(gotHash[:], wantHash[:]) == 1 && got != "" && want != ""
}
