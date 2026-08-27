package model

import "time"

type User struct {
	ID           int64      `json:"id"`
	Username     string     `json:"username"`
	PasswordHash string     `json:"-"`
	Role         string     `json:"role"`
	Enabled      bool       `json:"enabled"`
	Notes        string     `json:"notes"`
	LastLoginAt  *time.Time `json:"lastLoginAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

type Device struct {
	ID            int64     `json:"id"`
	ClientID      string    `json:"clientId"`
	GroupName     string    `json:"group"`
	Platform      string    `json:"platform"`
	Status        string    `json:"status"`
	LastSeenAt    time.Time `json:"lastSeenAt"`
	LastIP        string    `json:"lastIp"`
	ExtraJSON     string    `json:"extraJson"`
	DeviceMachine string    `json:"deviceMachine"`
	DeviceModel   string    `json:"deviceModel"`
	OSName        string    `json:"osName"`
	OSVersion     string    `json:"osVersion"`
	SDKName       string    `json:"sdkName"`
	SDKVersion    string    `json:"sdkVersion"`
	Actions       []string  `json:"actions"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type StorageSetting struct {
	Backend             string    `json:"backend"`
	LocalPath           string    `json:"localPath"`
	Endpoint            string    `json:"endpoint"`
	Region              string    `json:"region"`
	Bucket              string    `json:"bucket"`
	PathStyle           bool      `json:"pathStyle"`
	AccessKeyEncrypted  string    `json:"-"`
	SecretKeyEncrypted  string    `json:"-"`
	AccessKeyConfigured bool      `json:"accessKeyConfigured"`
	SecretKeyConfigured bool      `json:"secretKeyConfigured"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

type File struct {
	ID           string     `json:"id"`
	ObjectKey    string     `json:"-"`
	OriginalName string     `json:"originalName"`
	ContentType  string     `json:"contentType"`
	SizeBytes    int64      `json:"sizeBytes"`
	SHA256       string     `json:"sha256"`
	Backend      string     `json:"backend"`
	Status       string     `json:"status"`
	CreatedBy    int64      `json:"createdBy"`
	CreatedAt    time.Time  `json:"createdAt"`
	ExpiresAt    *time.Time `json:"expiresAt,omitempty"`
	DeletedAt    *time.Time `json:"deletedAt,omitempty"`
}

type FilePage struct {
	Items      []File `json:"items"`
	Page       int    `json:"page"`
	PageSize   int    `json:"pageSize"`
	Total      int64  `json:"total"`
	TotalPages int    `json:"totalPages"`
}

type RPCRequest struct {
	ID                  int64      `json:"id"`
	RequestID           string     `json:"requestId"`
	GroupName           string     `json:"group"`
	ActionName          string     `json:"action"`
	ClientID            string     `json:"clientId"`
	RequesterUserID     *int64     `json:"requesterUserId,omitempty"`
	RequestPayloadJSON  string     `json:"requestPayload"`
	ResponsePayloadJSON string     `json:"responsePayload"`
	Status              string     `json:"status"`
	HTTPCode            int        `json:"httpCode"`
	LatencyMS           int64      `json:"latencyMs"`
	ErrorMessage        string     `json:"errorMessage"`
	CreatedAt           time.Time  `json:"createdAt"`
	FinishedAt          *time.Time `json:"finishedAt,omitempty"`
}

type RequestFilterOptions struct {
	Groups    []string `json:"groups"`
	Actions   []string `json:"actions"`
	ClientIDs []string `json:"clientIds"`
}

type RPCRequestPage struct {
	Items      []RPCRequest `json:"items"`
	Page       int          `json:"page"`
	PageSize   int          `json:"pageSize"`
	Total      int64        `json:"total"`
	TotalPages int          `json:"totalPages"`
}

type DailyMetric struct {
	StatDate        string `json:"statDate"`
	ClientID        string `json:"clientId"`
	GroupName       string `json:"group"`
	TotalRequests   int64  `json:"totalRequests"`
	SuccessRequests int64  `json:"successRequests"`
	FailedRequests  int64  `json:"failedRequests"`
	TimeoutRequests int64  `json:"timeoutRequests"`
	TotalLatencyMS  int64  `json:"totalLatencyMs"`
	MaxLatencyMS    int64  `json:"maxLatencyMs"`
}

type WeeklyMetric struct {
	ClientID        string `json:"clientId"`
	GroupName       string `json:"group"`
	TotalRequests   int64  `json:"totalRequests"`
	SuccessRequests int64  `json:"successRequests"`
	FailedRequests  int64  `json:"failedRequests"`
	TimeoutRequests int64  `json:"timeoutRequests"`
	AvgLatencyMS    int64  `json:"avgLatencyMs"`
	MaxLatencyMS    int64  `json:"maxLatencyMs"`
}

type GroupInfo struct {
	GroupName     string     `json:"group"`
	DisplayName   string     `json:"displayName"`
	Enabled       bool       `json:"enabled"`
	DeviceKey     string     `json:"deviceKey"`
	AuthMode      string     `json:"authMode"` // none | apikey
	APIKey        string     `json:"apiKey"`
	Notes         string     `json:"notes"`
	TotalDevices  int64      `json:"totalDevices"`
	OnlineDevices int64      `json:"onlineDevices"`
	Requests7d    int64      `json:"requests7d"`
	Success7d     int64      `json:"success7d"`
	LastSeenAt    *time.Time `json:"lastSeenAt,omitempty"`
	LastRequestAt *time.Time `json:"lastRequestAt,omitempty"`
	Status        string     `json:"status"`
	StatusLabel   string     `json:"statusLabel"`
	SuccessRate   float64    `json:"successRate"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

type RealtimeBucket struct {
	Label   string `json:"t"` // HH:MM
	Success int64  `json:"success"`
	Failed  int64  `json:"failed"`
	Timeout int64  `json:"timeout"`
}

type GroupRealtimeBuckets struct {
	Group   string           `json:"group"`
	Buckets []RealtimeBucket `json:"buckets"`
}

type TrendPoint struct {
	StatDate        string  `json:"statDate"`
	TotalRequests   int64   `json:"totalRequests"`
	SuccessRequests int64   `json:"successRequests"`
	FailedRequests  int64   `json:"failedRequests"`
	TimeoutRequests int64   `json:"timeoutRequests"`
	AvgLatencyMS    int64   `json:"avgLatencyMs"`
	MaxLatencyMS    int64   `json:"maxLatencyMs"`
	SuccessRate     float64 `json:"successRate"`
}
