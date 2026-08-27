package rpc

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"
)

type QueueID string
type JobID string
type LeaseID string

type Delivery struct {
	QueueID    QueueID
	JobID      JobID
	LeaseID    LeaseID
	Attempt    int
	LeaseUntil time.Time
	Job        *Job
}

type QueueEventType string

const (
	QueueEventQueued       QueueEventType = "queued"
	QueueEventDequeued     QueueEventType = "dequeued"
	QueueEventRequeued     QueueEventType = "requeued"
	QueueEventExpired      QueueEventType = "expired"
	QueueEventDropped      QueueEventType = "dropped"
	QueueEventReserved     QueueEventType = "reserved"
	QueueEventLeaseRenewed QueueEventType = "lease_renewed"
	QueueEventAcked        QueueEventType = "acked"
	QueueEventLeaseExpired QueueEventType = "lease_expired"
)

type QueueEvent struct {
	EventID    string         `json:"eventId"`
	QueueID    QueueID        `json:"queueId"`
	JobID      JobID          `json:"jobId"`
	RequestID  string         `json:"requestId"`
	Group      string         `json:"group"`
	Action     string         `json:"action"`
	ClientID   string         `json:"clientId"`
	LeaseRef   string         `json:"leaseRef,omitempty"`
	Attempt    int            `json:"attempt,omitempty"`
	LeaseUntil time.Time      `json:"leaseUntil,omitempty"`
	Type       QueueEventType `json:"type"`
	At         time.Time      `json:"at"`
	Reason     string         `json:"reason,omitempty"`
}

// JobQueue keeps state transitions implementation-independent so Reserve, Ack,
// RequeueDelivery, and Sweep can map to Redis scripts and Streams later. Ack and
// RequeueDelivery must compare LeaseUntil with the queue's authoritative clock
// atomically with their state transition. Successful Ack tombstones provide
// bounded idempotency: they are retained for at most five minutes and at most
// 1024 entries per queue, with age expiry or oldest-first capacity eviction,
// whichever happens first. Sweep's caller-supplied time is only a lease-expiry
// cutoff and must not age tombstones. The memory implementation uses its
// injected clock for all other time decisions. A Redis implementation must use
// Redis TIME as its authoritative clock, including for the lease-expiry cutoff.
type JobQueue interface {
	ID() QueueID
	Len() int
	Cap() int
	Enqueue(job *Job) error
	Requeue(job *Job, reason string) error
	Dequeue(ctx context.Context) (*Job, error)
	Reserve(ctx context.Context, leaseDuration time.Duration) (Delivery, error)
	// Renew atomically extends the same reservation using the queue's
	// authoritative clock. It never shortens an existing lease.
	Renew(delivery Delivery, leaseDuration time.Duration) (Delivery, error)
	// Ack is idempotent only while its bounded successful-Ack tombstone remains.
	Ack(delivery Delivery) error
	RequeueDelivery(delivery Delivery, reason string) error
	// Sweep uses now only to select leases whose LeaseUntil has elapsed.
	Sweep(now time.Time) int
	RemoveUnsupported(allowed map[string]struct{}, reason string) []*Job
	Events() []QueueEvent
}

var (
	ErrQueueIdentityMismatch = errors.New("任务所属队列与目标队列不匹配")
	ErrInvalidLeaseDuration  = errors.New("lease duration must be positive")
	ErrLeaseIdentityMismatch = errors.New("delivery job does not match lease")
	ErrLeaseNotFound         = errors.New("delivery lease does not exist")
	ErrLeaseExpired          = errors.New("delivery lease has expired")
)

func newOpaqueID(prefix string) string {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err == nil {
		return prefix + hex.EncodeToString(buffer)
	}
	return prefix + time.Now().UTC().Format("20060102150405.000000000")
}

func prepareJobForQueue(queueID QueueID, job *Job) {
	if job.JobID == "" {
		job.JobID = JobID(job.RequestID)
	}
	job.QueueID = queueID
}

func leaseReference(leaseID LeaseID) string {
	if leaseID == "" {
		return ""
	}
	digest := sha256.Sum256([]byte(leaseID))
	return hex.EncodeToString(digest[:16])
}
