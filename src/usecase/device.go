package usecase

import (
	"context"
	"fmt"

	domainApp "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/app"
	domainDevice "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/device"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/ui/websocket"
)

type serviceDevice struct {
	manager    *whatsapp.DeviceManager
	appService domainApp.IAppUsecase
}

func NewDeviceService(manager *whatsapp.DeviceManager, appService domainApp.IAppUsecase) domainDevice.IDeviceUsecase {
	return &serviceDevice{
		manager:    manager,
		appService: appService,
	}
}

func (s *serviceDevice) ListDevices(_ context.Context) ([]domainDevice.Device, error) {
	if s.manager == nil {
		return []domainDevice.Device{}, nil
	}

	var result []domainDevice.Device
	for _, inst := range s.manager.ListDevices() {
		inst.UpdateStateFromClient()
		result = append(result, convertInstance(inst))
	}
	return result, nil
}

func (s *serviceDevice) GetDevice(_ context.Context, deviceID string) (*domainDevice.Device, error) {
	if s.manager == nil {
		return nil, fmt.Errorf("device manager not initialized")
	}
	if inst, ok := s.manager.GetDevice(deviceID); ok {
		device := convertInstance(inst)
		return &device, nil
	}
	return nil, fmt.Errorf("device %s not found", deviceID)
}

func (s *serviceDevice) AddDevice(ctx context.Context, deviceID string) (*domainDevice.Device, error) {
	if s.manager == nil {
		return nil, fmt.Errorf("device manager not initialized")
	}

	inst, err := s.manager.CreateDevice(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	device := convertInstance(inst)
	return &device, nil
}

func (s *serviceDevice) RemoveDevice(_ context.Context, deviceID string) error {
	if s.manager == nil {
		return fmt.Errorf("device manager not initialized")
	}
	s.manager.RemoveDevice(deviceID)
	return nil
}

func (s *serviceDevice) LoginDevice(ctx context.Context, deviceID string) (domainApp.LoginResponse, error) {
	if s.appService == nil {
		return domainApp.LoginResponse{}, fmt.Errorf("app service not initialized")
	}

	return s.appService.Login(ctx, deviceID)
}

func (s *serviceDevice) LoginDeviceWithCode(ctx context.Context, deviceID string, phone string) (string, error) {
	if s.appService == nil {
		return "", fmt.Errorf("app service not initialized")
	}

	return s.appService.LoginWithCode(ctx, deviceID, phone)
}

func (s *serviceDevice) LogoutDevice(ctx context.Context, deviceID string) error {
	if s.manager == nil {
		return fmt.Errorf("device manager not initialized")
	}

	if err := s.manager.PurgeDevice(ctx, deviceID); err != nil {
		return err
	}

	// Broadcast device removal so UI clients can refresh.
	var devices []domainDevice.Device
	if s.manager != nil {
		for _, inst := range s.manager.ListDevices() {
			inst.UpdateStateFromClient()
			devices = append(devices, convertInstance(inst))
		}
	}

	websocket.Broadcast <- websocket.BroadcastMessage{
		Code:    "DEVICE_REMOVED",
		Message: fmt.Sprintf("Device %s logged out and removed", deviceID),
		Result: map[string]any{
			"device_id": deviceID,
			"devices":   devices,
		},
	}

	return nil
}

func (s *serviceDevice) ReconnectDevice(_ context.Context, deviceID string) error {
	if s.manager == nil {
		return fmt.Errorf("device manager not initialized")
	}
	if inst, ok := s.manager.GetDevice(deviceID); ok {
		client := inst.GetClient()
		if client == nil {
			return fmt.Errorf("device %s client not initialized", deviceID)
		}

		if client.Store == nil || client.Store.ID == nil {
			return fmt.Errorf("device %s is not logged in (session deleted)", deviceID)
		}

		client.Disconnect()
		return client.Connect()
	}
	return fmt.Errorf("device %s not found", deviceID)
}

func (s *serviceDevice) GetStatus(_ context.Context, deviceID string) (bool, bool, error) {
	if s.manager == nil {
		return false, false, fmt.Errorf("device manager not initialized")
	}
	if inst, ok := s.manager.GetDevice(deviceID); ok {
		inst.UpdateStateFromClient()
		client := inst.GetClient()
		if client == nil {
			return false, false, nil
		}

		if client.Store == nil || client.Store.ID == nil {
			return false, false, nil
		}

		// Update state snapshot based on live client flags
		state := deriveState(inst)
		_ = state
		return client.IsConnected(), client.IsLoggedIn(), nil
	}
	return false, false, fmt.Errorf("device %s not found", deviceID)
}

func convertInstance(inst *whatsapp.DeviceInstance) domainDevice.Device {
	if inst == nil {
		return domainDevice.Device{}
	}

	state := deriveState(inst)

	return domainDevice.Device{
		ID:          inst.ID(),
		PhoneNumber: inst.PhoneNumber(),
		DisplayName: inst.DisplayName(),
		State:       state,
		JID:         inst.JID(),
		CreatedAt:   inst.CreatedAt(),
	}
}

func deriveState(inst *whatsapp.DeviceInstance) domainDevice.DeviceState {
	if inst == nil {
		return domainDevice.DeviceStateDisconnected
	}

	client := inst.GetClient()
	state := inst.State()
	if client != nil {
		if client.IsLoggedIn() {
			state = domainDevice.DeviceStateLoggedIn
		} else if client.IsConnected() {
			state = domainDevice.DeviceStateConnected
		} else {
			state = domainDevice.DeviceStateDisconnected
		}
		inst.SetState(state)
	}

	return state
}
