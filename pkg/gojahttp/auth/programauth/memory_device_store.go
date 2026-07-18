package programauth

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

type MemoryDeviceAuthorizationStore struct {
	mu      sync.Mutex
	devices map[string]DeviceAuthorization
}

func NewMemoryDeviceAuthorizationStore() *MemoryDeviceAuthorizationStore {
	return &MemoryDeviceAuthorizationStore{devices: map[string]DeviceAuthorization{}}
}

func (s *MemoryDeviceAuthorizationStore) CreateDeviceAuthorization(_ context.Context, device DeviceAuthorization) (DeviceAuthorization, error) {
	if s == nil {
		return DeviceAuthorization{}, fmt.Errorf("programauth memory device authorization store is nil")
	}
	device = cloneDeviceAuthorization(device)
	if device.ID == "" {
		return DeviceAuthorization{}, fmt.Errorf("device authorization id is required")
	}
	if device.DeviceCodePrefix == "" || len(device.DeviceCodeHash) == 0 || len(device.UserCodeHash) == 0 {
		return DeviceAuthorization{}, fmt.Errorf("device authorization hashes and prefix are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.devices == nil {
		s.devices = map[string]DeviceAuthorization{}
	}
	if _, exists := s.devices[device.ID]; exists {
		return DeviceAuthorization{}, fmt.Errorf("device authorization %q already exists", device.ID)
	}
	for _, existing := range s.devices {
		if bytes.Equal(existing.UserCodeHash, device.UserCodeHash) {
			return DeviceAuthorization{}, fmt.Errorf("device authorization user code already exists")
		}
	}
	s.devices[device.ID] = device
	return cloneDeviceAuthorization(device), nil
}

func (s *MemoryDeviceAuthorizationStore) FindDeviceAuthorizationByDeviceCodePrefix(_ context.Context, prefix string) ([]DeviceAuthorization, error) {
	if s == nil {
		return nil, fmt.Errorf("programauth memory device authorization store is nil")
	}
	prefix = strings.TrimSpace(prefix)
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []DeviceAuthorization{}
	for _, device := range s.devices {
		if device.DeviceCodePrefix == prefix {
			out = append(out, cloneDeviceAuthorization(device))
		}
	}
	return out, nil
}

func (s *MemoryDeviceAuthorizationStore) GetDeviceAuthorizationByUserCodeHash(_ context.Context, hash []byte) (DeviceAuthorization, error) {
	if s == nil {
		return DeviceAuthorization{}, fmt.Errorf("programauth memory device authorization store is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, device := range s.devices {
		if bytes.Equal(device.UserCodeHash, hash) {
			return cloneDeviceAuthorization(device), nil
		}
	}
	return DeviceAuthorization{}, ErrDeviceNotFound
}

func (s *MemoryDeviceAuthorizationStore) RecordDevicePoll(_ context.Context, id string, polledAt time.Time, interval time.Duration) (DeviceAuthorization, error) {
	if s == nil {
		return DeviceAuthorization{}, fmt.Errorf("programauth memory device authorization store is nil")
	}
	id = strings.TrimSpace(id)
	s.mu.Lock()
	defer s.mu.Unlock()
	device, ok := s.devices[id]
	if !ok {
		return DeviceAuthorization{}, ErrDeviceNotFound
	}
	polledAt = polledAt.UTC()
	device.LastPolledAt = &polledAt
	if interval > 0 {
		device.PollInterval = interval
	}
	device.UpdatedAt = polledAt
	s.devices[id] = device
	return cloneDeviceAuthorization(device), nil
}

func (s *MemoryDeviceAuthorizationStore) ApproveDeviceAuthorization(_ context.Context, id string, approved DeviceAuthorization, approvedAt time.Time) (DeviceAuthorization, error) {
	if s == nil {
		return DeviceAuthorization{}, fmt.Errorf("programauth memory device authorization store is nil")
	}
	id = strings.TrimSpace(id)
	approved = cloneDeviceAuthorization(approved)
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.devices[id]
	if !ok {
		return DeviceAuthorization{}, ErrDeviceNotFound
	}
	if current.Approved() {
		return DeviceAuthorization{}, fmt.Errorf("device authorization already approved")
	}
	if current.Denied() {
		return DeviceAuthorization{}, ErrDeviceDenied
	}
	if current.Consumed() {
		return DeviceAuthorization{}, ErrDeviceConsumed
	}
	approvedAt = approvedAt.UTC()
	current.AgentID = strings.TrimSpace(approved.AgentID)
	current.SubjectUserID = strings.TrimSpace(approved.SubjectUserID)
	current.TenantID = strings.TrimSpace(approved.TenantID)
	current.Grants = approved.Grants.Clone()
	current.ApprovedAt = &approvedAt
	current.UpdatedAt = approvedAt
	s.devices[id] = current
	return cloneDeviceAuthorization(current), nil
}

func (s *MemoryDeviceAuthorizationStore) DenyDeviceAuthorization(_ context.Context, id string, deniedAt time.Time) (DeviceAuthorization, error) {
	if s == nil {
		return DeviceAuthorization{}, fmt.Errorf("programauth memory device authorization store is nil")
	}
	id = strings.TrimSpace(id)
	s.mu.Lock()
	defer s.mu.Unlock()
	device, ok := s.devices[id]
	if !ok {
		return DeviceAuthorization{}, ErrDeviceNotFound
	}
	deniedAt = deniedAt.UTC()
	device.DeniedAt = &deniedAt
	device.UpdatedAt = deniedAt
	s.devices[id] = device
	return cloneDeviceAuthorization(device), nil
}

func (s *MemoryDeviceAuthorizationStore) ConsumeDeviceAuthorization(_ context.Context, id string, consumedAt time.Time) (DeviceAuthorization, error) {
	if s == nil {
		return DeviceAuthorization{}, fmt.Errorf("programauth memory device authorization store is nil")
	}
	id = strings.TrimSpace(id)
	s.mu.Lock()
	defer s.mu.Unlock()
	device, ok := s.devices[id]
	if !ok {
		return DeviceAuthorization{}, ErrDeviceNotFound
	}
	if device.Consumed() {
		return DeviceAuthorization{}, ErrDeviceConsumed
	}
	consumedAt = consumedAt.UTC()
	device.ConsumedAt = &consumedAt
	device.UpdatedAt = consumedAt
	s.devices[id] = device
	return cloneDeviceAuthorization(device), nil
}
