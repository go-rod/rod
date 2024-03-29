// This file is generated by "./lib/proto/generate"

package proto

/*

BackgroundService

Defines events for background web platform features.

*/

// BackgroundServiceServiceName The Background Service that will be associated with the commands/events.
// Every Background Service operates independently, but they share the same
// API.
type BackgroundServiceServiceName string

const (
	// BackgroundServiceServiceNameBackgroundFetch enum const.
	BackgroundServiceServiceNameBackgroundFetch BackgroundServiceServiceName = "backgroundFetch"

	// BackgroundServiceServiceNameBackgroundSync enum const.
	BackgroundServiceServiceNameBackgroundSync BackgroundServiceServiceName = "backgroundSync"

	// BackgroundServiceServiceNamePushMessaging enum const.
	BackgroundServiceServiceNamePushMessaging BackgroundServiceServiceName = "pushMessaging"

	// BackgroundServiceServiceNameNotifications enum const.
	BackgroundServiceServiceNameNotifications BackgroundServiceServiceName = "notifications"

	// BackgroundServiceServiceNamePaymentHandler enum const.
	BackgroundServiceServiceNamePaymentHandler BackgroundServiceServiceName = "paymentHandler"

	// BackgroundServiceServiceNamePeriodicBackgroundSync enum const.
	BackgroundServiceServiceNamePeriodicBackgroundSync BackgroundServiceServiceName = "periodicBackgroundSync"
)

// BackgroundServiceEventMetadata A key-value pair for additional event information to pass along.
type BackgroundServiceEventMetadata struct {
	// Key ...
	Key string `json:"key"`

	// Value ...
	Value string `json:"value"`
}

// BackgroundServiceBackgroundServiceEvent ...
type BackgroundServiceBackgroundServiceEvent struct {
	// Timestamp of the event (in seconds).
	Timestamp TimeSinceEpoch `json:"timestamp"`

	// Origin The origin this event belongs to.
	Origin string `json:"origin"`

	// ServiceWorkerRegistrationID The Service Worker ID that initiated the event.
	ServiceWorkerRegistrationID ServiceWorkerRegistrationID `json:"serviceWorkerRegistrationId"`

	// Service The Background Service this event belongs to.
	Service BackgroundServiceServiceName `json:"service"`

	// EventName A description of the event.
	EventName string `json:"eventName"`

	// InstanceID An identifier that groups related events together.
	InstanceID string `json:"instanceId"`

	// EventMetadata A list of event-specific information.
	EventMetadata []*BackgroundServiceEventMetadata `json:"eventMetadata"`

	// StorageKey Storage key this event belongs to.
	StorageKey string `json:"storageKey"`
}

// BackgroundServiceStartObserving Enables event updates for the service.
type BackgroundServiceStartObserving struct {
	// Service ...
	Service BackgroundServiceServiceName `json:"service"`
}

// ProtoReq name.
func (m BackgroundServiceStartObserving) ProtoReq() string { return "BackgroundService.startObserving" }

// Call sends the request.
func (m BackgroundServiceStartObserving) Call(c Client) error {
	return call(m.ProtoReq(), m, nil, c)
}

// BackgroundServiceStopObserving Disables event updates for the service.
type BackgroundServiceStopObserving struct {
	// Service ...
	Service BackgroundServiceServiceName `json:"service"`
}

// ProtoReq name.
func (m BackgroundServiceStopObserving) ProtoReq() string { return "BackgroundService.stopObserving" }

// Call sends the request.
func (m BackgroundServiceStopObserving) Call(c Client) error {
	return call(m.ProtoReq(), m, nil, c)
}

// BackgroundServiceSetRecording Set the recording state for the service.
type BackgroundServiceSetRecording struct {
	// ShouldRecord ...
	ShouldRecord bool `json:"shouldRecord"`

	// Service ...
	Service BackgroundServiceServiceName `json:"service"`
}

// ProtoReq name.
func (m BackgroundServiceSetRecording) ProtoReq() string { return "BackgroundService.setRecording" }

// Call sends the request.
func (m BackgroundServiceSetRecording) Call(c Client) error {
	return call(m.ProtoReq(), m, nil, c)
}

// BackgroundServiceClearEvents Clears all stored data for the service.
type BackgroundServiceClearEvents struct {
	// Service ...
	Service BackgroundServiceServiceName `json:"service"`
}

// ProtoReq name.
func (m BackgroundServiceClearEvents) ProtoReq() string { return "BackgroundService.clearEvents" }

// Call sends the request.
func (m BackgroundServiceClearEvents) Call(c Client) error {
	return call(m.ProtoReq(), m, nil, c)
}

// BackgroundServiceRecordingStateChanged Called when the recording state for the service has been updated.
type BackgroundServiceRecordingStateChanged struct {
	// IsRecording ...
	IsRecording bool `json:"isRecording"`

	// Service ...
	Service BackgroundServiceServiceName `json:"service"`
}

// ProtoEvent name.
func (evt BackgroundServiceRecordingStateChanged) ProtoEvent() string {
	return "BackgroundService.recordingStateChanged"
}

// BackgroundServiceBackgroundServiceEventReceived Called with all existing backgroundServiceEvents when enabled, and all new
// events afterwards if enabled and recording.
type BackgroundServiceBackgroundServiceEventReceived struct {
	// BackgroundServiceEvent ...
	BackgroundServiceEvent *BackgroundServiceBackgroundServiceEvent `json:"backgroundServiceEvent"`
}

// ProtoEvent name.
func (evt BackgroundServiceBackgroundServiceEventReceived) ProtoEvent() string {
	return "BackgroundService.backgroundServiceEventReceived"
}
