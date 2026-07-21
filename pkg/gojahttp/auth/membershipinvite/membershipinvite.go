// Package membershipinvite implements the narrow application operation that
// accepts a durable organization invitation and grants app membership.
package membershipinvite

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
)

var (
	ErrUnauthenticated = errors.New("membership invite requires an authenticated actor")
	ErrEmailUnverified = errors.New("membership invite requires a verified email")
	ErrEmailMismatch   = errors.New("membership invite email does not match the actor")
	ErrRoleNotAllowed  = errors.New("membership invite role is not allowed")
)

// Result is the non-secret result of an atomic invitation acceptance.
type Result struct {
	CapabilityID string
	UserID       string
	TenantID     string
	Role         string
}

// Pending is a short-lived, non-capability continuation returned after a raw
// invite is presented. Only HashToken(Handle) is persisted.
type Pending struct {
	Handle       string
	CapabilityID string
	TenantID     string
	Email        string
	Role         string
	ExpiresAt    time.Time
}

// Acceptor owns the transaction spanning capability consumption and membership
// creation. Implementations must not make either mutation visible alone.
type Acceptor interface {
	Begin(ctx context.Context, token string, now time.Time) (Pending, error)
	Accept(ctx context.Context, token, actorUserID string, now time.Time) (Result, error)
	AcceptPending(ctx context.Context, handle, actorUserID string, now time.Time) (Result, error)
}

// Service validates the host-facing call and records a bounded audit event.
type Service struct {
	Acceptor Acceptor
	Audit    gojahttp.AuditSink
	Now      func() time.Time
}

func (s Service) Begin(ctx context.Context, token string) (Pending, error) {
	if s.Acceptor == nil {
		return Pending{}, fmt.Errorf("membership invite: acceptor is required")
	}
	if token = strings.TrimSpace(token); token == "" {
		return Pending{}, fmt.Errorf("membership invite: token is required")
	}
	return s.Acceptor.Begin(ctx, token, s.now())
}

func (s Service) AcceptPending(ctx context.Context, handle, actorUserID string) (Result, error) {
	if s.Acceptor == nil {
		return Result{}, fmt.Errorf("membership invite: acceptor is required")
	}
	handle, actorUserID = strings.TrimSpace(handle), strings.TrimSpace(actorUserID)
	if handle == "" {
		return Result{}, fmt.Errorf("membership invite: pending handle is required")
	}
	if actorUserID == "" {
		return Result{}, ErrUnauthenticated
	}
	result, err := s.Acceptor.AcceptPending(ctx, handle, actorUserID, s.now())
	s.record(ctx, result, actorUserID, err)
	return result, err
}

func (s Service) Accept(ctx context.Context, token, actorUserID string) (Result, error) {
	if s.Acceptor == nil {
		return Result{}, fmt.Errorf("membership invite: acceptor is required")
	}
	token = strings.TrimSpace(token)
	actorUserID = strings.TrimSpace(actorUserID)
	if token == "" {
		return Result{}, fmt.Errorf("membership invite: token is required")
	}
	if actorUserID == "" {
		return Result{}, ErrUnauthenticated
	}
	result, err := s.Acceptor.Accept(ctx, token, actorUserID, s.now())
	s.record(ctx, result, actorUserID, err)
	return result, err
}

func (s Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func (s Service) record(ctx context.Context, result Result, actorID string, err error) {
	if s.Audit == nil {
		return
	}
	outcome := "completed"
	reason := ""
	if err != nil {
		outcome = "denied"
		reason = err.Error()
	}
	_ = s.Audit.RecordAudit(ctx, gojahttp.AuditEvent{
		Event: "org.invite.accepted", Outcome: outcome, Reason: reason,
		Actor:      &gojahttp.Actor{ID: actorID},
		Resource:   &gojahttp.ResourceRef{Type: "org", ID: result.TenantID},
		Attributes: map[string]any{"capabilityId": result.CapabilityID, "role": result.Role},
	})
}
