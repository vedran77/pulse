package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vedran77/pulse/internal/domain"
	"github.com/vedran77/pulse/internal/repository"
)

var (
	ErrCannotRequestSelf     = errors.New("cannot send a pulsemate request to yourself")
	ErrUserNotFoundForRequest = errors.New("user not found")
	ErrRequestAlreadyExists  = errors.New("a pending request already exists")
	ErrAlreadyPulsemates     = errors.New("you are already pulsemates")
	ErrRequestNotFound       = errors.New("pulsemate request not found")
	ErrNotRequestReceiver    = errors.New("only the request receiver can perform this action")
	ErrNotRequestSender      = errors.New("only the request sender can cancel")
)

type PulsemateService struct {
	pmRepo   repository.PulsemateRepository
	userRepo repository.UserRepository
}

func NewPulsemateService(pmRepo repository.PulsemateRepository, userRepo repository.UserRepository) *PulsemateService {
	return &PulsemateService{
		pmRepo:   pmRepo,
		userRepo: userRepo,
	}
}

// SendRequest sends a pulsemate request by target username.
// Auto-accepts if the other user already sent a request to the sender.
func (s *PulsemateService) SendRequest(ctx context.Context, senderID uuid.UUID, targetUsername string) (*domain.PulsemateRequest, error) {
	// Look up target user
	target, err := s.userRepo.GetByUsername(ctx, targetUsername)
	if err != nil {
		return nil, fmt.Errorf("looking up user: %w", err)
	}
	if target == nil {
		return nil, ErrUserNotFoundForRequest
	}

	if senderID == target.ID {
		return nil, ErrCannotRequestSelf
	}

	// Check if already pulsemates
	already, err := s.pmRepo.ArePulsemates(ctx, senderID, target.ID)
	if err != nil {
		return nil, err
	}
	if already {
		return nil, ErrAlreadyPulsemates
	}

	// Check if sender already sent a request to target
	existing, err := s.pmRepo.GetRequestByUsers(ctx, senderID, target.ID)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.Status == "pending" {
		return nil, ErrRequestAlreadyExists
	}

	// Check if target already sent a request to sender → auto-accept
	reverse, err := s.pmRepo.GetRequestByUsers(ctx, target.ID, senderID)
	if err != nil {
		return nil, err
	}
	if reverse != nil && reverse.Status == "pending" {
		// Auto-accept: create pulsemate and delete the reverse request
		if err := s.createPulsemate(ctx, senderID, target.ID); err != nil {
			return nil, err
		}
		if err := s.pmRepo.DeleteRequest(ctx, reverse.ID); err != nil {
			return nil, err
		}
		// Return nil to indicate auto-accepted (no pending request created)
		return nil, nil
	}

	// Create new request
	req := &domain.PulsemateRequest{
		ID:         uuid.New(),
		SenderID:   senderID,
		ReceiverID: target.ID,
		Status:     "pending",
		CreatedAt:  time.Now(),
	}

	if err := s.pmRepo.CreateRequest(ctx, req); err != nil {
		return nil, fmt.Errorf("creating pulsemate request: %w", err)
	}

	return req, nil
}

// AcceptRequest accepts a pending pulsemate request.
func (s *PulsemateService) AcceptRequest(ctx context.Context, userID uuid.UUID, requestID uuid.UUID) error {
	req, err := s.pmRepo.GetRequestByID(ctx, requestID)
	if err != nil {
		return err
	}
	if req == nil {
		return ErrRequestNotFound
	}
	if req.ReceiverID != userID {
		return ErrNotRequestReceiver
	}

	if err := s.createPulsemate(ctx, req.SenderID, req.ReceiverID); err != nil {
		return err
	}

	return s.pmRepo.DeleteRequest(ctx, requestID)
}

// RejectRequest rejects (deletes) a pending pulsemate request.
func (s *PulsemateService) RejectRequest(ctx context.Context, userID uuid.UUID, requestID uuid.UUID) error {
	req, err := s.pmRepo.GetRequestByID(ctx, requestID)
	if err != nil {
		return err
	}
	if req == nil {
		return ErrRequestNotFound
	}
	if req.ReceiverID != userID {
		return ErrNotRequestReceiver
	}

	return s.pmRepo.DeleteRequest(ctx, requestID)
}

// CancelRequest cancels a pending request sent by the user.
func (s *PulsemateService) CancelRequest(ctx context.Context, userID uuid.UUID, requestID uuid.UUID) error {
	req, err := s.pmRepo.GetRequestByID(ctx, requestID)
	if err != nil {
		return err
	}
	if req == nil {
		return ErrRequestNotFound
	}
	if req.SenderID != userID {
		return ErrNotRequestSender
	}

	return s.pmRepo.DeleteRequest(ctx, requestID)
}

// ListPulsemates returns all pulsemates for a user.
func (s *PulsemateService) ListPulsemates(ctx context.Context, userID uuid.UUID) ([]domain.Pulsemate, error) {
	pms, err := s.pmRepo.ListPulsemates(ctx, userID)
	if err != nil {
		return nil, err
	}
	if pms == nil {
		pms = []domain.Pulsemate{}
	}
	return pms, nil
}

// ListIncomingRequests returns pending requests received by the user.
func (s *PulsemateService) ListIncomingRequests(ctx context.Context, userID uuid.UUID) ([]domain.PulsemateRequest, error) {
	reqs, err := s.pmRepo.ListIncomingRequests(ctx, userID)
	if err != nil {
		return nil, err
	}
	if reqs == nil {
		reqs = []domain.PulsemateRequest{}
	}
	return reqs, nil
}

// ListOutgoingRequests returns pending requests sent by the user.
func (s *PulsemateService) ListOutgoingRequests(ctx context.Context, userID uuid.UUID) ([]domain.PulsemateRequest, error) {
	reqs, err := s.pmRepo.ListOutgoingRequests(ctx, userID)
	if err != nil {
		return nil, err
	}
	if reqs == nil {
		reqs = []domain.PulsemateRequest{}
	}
	return reqs, nil
}

// RemovePulsemate removes a pulsemate relationship.
func (s *PulsemateService) RemovePulsemate(ctx context.Context, userID, otherUserID uuid.UUID) error {
	u1, u2 := userID, otherUserID
	if u1.String() > u2.String() {
		u1, u2 = u2, u1
	}
	return s.pmRepo.DeletePulsemate(ctx, u1, u2)
}

// createPulsemate creates a pulsemate with canonical ordering.
func (s *PulsemateService) createPulsemate(ctx context.Context, userA, userB uuid.UUID) error {
	u1, u2 := userA, userB
	if u1.String() > u2.String() {
		u1, u2 = u2, u1
	}

	pm := &domain.Pulsemate{
		ID:        uuid.New(),
		User1ID:   u1,
		User2ID:   u2,
		CreatedAt: time.Now(),
	}

	return s.pmRepo.CreatePulsemate(ctx, pm)
}
