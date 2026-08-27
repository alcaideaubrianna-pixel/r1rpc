package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID             int64    `json:"userId"`
	Username           string   `json:"username"`
	Role               string   `json:"role"`
	ClientID           string   `json:"clientId,omitempty"`
	Group              string   `json:"group,omitempty"`
	MaxInFlight        int      `json:"maxInFlight,omitempty"`
	ActionsKnown       bool     `json:"actionsKnown,omitempty"`
	Actions            []string `json:"actions,omitempty"`
	TokenKind          string   `json:"tokenKind,omitempty"`
	BootstrapID        string   `json:"bootstrapId,omitempty"`
	SessionIncarnation string   `json:"sessionIncarnation,omitempty"`
	SessionGeneration  uint64   `json:"sessionGeneration,omitempty"`
	jwt.RegisteredClaims
}

const (
	TokenKindBootstrap  = "bootstrap"
	TokenKindConnection = "connection"
)

type TokenManager struct {
	secret []byte
}

func NewTokenManager(secret string) *TokenManager {
	return &TokenManager{secret: []byte(secret)}
}

func (m *TokenManager) Issue(claims Claims, ttl time.Duration) (string, error) {
	now := time.Now()
	claims.RegisteredClaims = jwt.RegisteredClaims{
		ID:        randomID(),
		Subject:   claims.Username,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

func (m *TokenManager) Parse(raw string) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(raw, &Claims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("签名算法不匹配")
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, fmt.Errorf("token 无效")
	}
	return claims, nil
}

func randomID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}
