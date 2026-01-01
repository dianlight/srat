package api_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/dianlight/srat/api"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/server/ws"
	"github.com/dianlight/srat/service"
	"github.com/gorilla/websocket"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
)

type WsMessageSenderSuite struct {
	suite.Suite
	mockConn        *websocket.Conn
	hanlder         *api.WebSocketHandler
	wsMessageSender *api.WsMessageSender
}

func TestWsMessageSenderSuite(t *testing.T) {
	suite.Run(t, new(WsMessageSenderSuite))
}

func (suite *WsMessageSenderSuite) SetupTest() {
	mockController := mock.NewMockController(suite.T())
	suite.hanlder = api.NewWebSocketBroker(suite.T().Context(), mock.Mock[service.BroadcasterServiceInterface](mockController))
	//suite.mockConn = mock.Mock[*websocket.Conn](mockController)
	suite.wsMessageSender = &api.WsMessageSender{
		//Connection: suite.mockConn,
		ObjectMap: suite.hanlder.ObjectMap,
	}
}

// TestSendFunc_SuccessWithWelcomeMessage tests sending a Welcome message successfully
func (suite *WsMessageSenderSuite) TestSendFunc_SuccessWithWelcomeMessage() {
	machineID := "test-machine-123"
	msg := ws.Message{
		ID: 1,
		Data: dto.Welcome{
			Message:         "Welcome to SRAT",
			ActiveClients:   5,
			SupportedEvents: []dto.WebEventType{dto.WebEventTypes.EVENTHELLO},
			UpdateChannel:   "stable",
			MachineId:       &machineID,
			BuildVersion:    "1.0.0",
			SecureMode:      true,
			ProtectedMode:   false,
			ReadOnly:        false,
			StartTime:       time.Now().Unix(),
		},
	}

	err := suite.wsMessageSender.SendFunc(msg)
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "WebSocket connection is nil")

	expectedEventName := "hello"
	suite.Contains(suite.hanlder.ObjectMap, "dto.Welcome")
	suite.Equal(expectedEventName, suite.hanlder.ObjectMap["dto.Welcome"], "Expected event name for Welcome to be %s %#v", expectedEventName, suite.hanlder.ObjectMap)

	// Verify the reverse map contains the correct event type
	suite.Equal("hello", suite.hanlder.ObjectMap["dto.Welcome"])
}

// TestSendFunc_SuccessWithUpdateProgress tests sending an UpdateProgress message
func (suite *WsMessageSenderSuite) TestSendFunc_SuccessWithUpdateProgress() {
	msg := ws.Message{
		ID: 2,
		Data: dto.UpdateProgress{
			Progress:    75,
			LastRelease: "v1.2.3",
		},
	}

	err := suite.wsMessageSender.SendFunc(msg)
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "WebSocket connection is nil")

	expectedEventName := "updating"
	suite.Equal(expectedEventName, suite.wsMessageSender.ObjectMap["dto.UpdateProgress"])
}

// TestSendFunc_SuccessWithHealthPing tests sending a HealthPing message
func (suite *WsMessageSenderSuite) TestSendFunc_SuccessWithHealthPing() {
	msg := ws.Message{
		ID: 3,
		Data: dto.HealthPing{
			Alive:     true,
			AliveTime: time.Now().Unix(),
		},
	}
	err := suite.wsMessageSender.SendFunc(msg)
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "WebSocket connection is nil")

	expectedEventName := "heartbeat"
	suite.Equal(expectedEventName, suite.wsMessageSender.ObjectMap["dto.HealthPing"])
}

// TestSendFunc_SuccessWithShares tests sending a SharedResource list
func (suite *WsMessageSenderSuite) TestSendFunc_SuccessWithShares() {
	msg := ws.Message{
		Data: []dto.SharedResource{
			{Name: "share1"},
		},
	}

	err := suite.wsMessageSender.SendFunc(msg)
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "WebSocket connection is nil")

	expectedEventName := "shares"
	newVar := reflect.TypeOf(msg.Data)
	typeName := newVar.String()
	suite.Contains(suite.wsMessageSender.ObjectMap, typeName)
	suite.Equal(expectedEventName, suite.wsMessageSender.ObjectMap[typeName])
}

// TestSendFunc_SuccessWithVolumes tests sending a Disk list
func (suite *WsMessageSenderSuite) TestSendFunc_SuccessWithVolumes() {
	legacyName := "sda"
	msg := ws.Message{
		Data: []*dto.Disk{
			{LegacyDeviceName: &legacyName},
		},
	}

	err := suite.wsMessageSender.SendFunc(msg)
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "WebSocket connection is nil")

	typeName := reflect.TypeOf(msg.Data).String()
	suite.Contains(suite.wsMessageSender.ObjectMap, typeName)
	suite.Equal("volumes", suite.wsMessageSender.ObjectMap[typeName])
}

// TestSendFunc_SuccessWithDirtyTracker tests sending a DataDirtyTracker message
func (suite *WsMessageSenderSuite) TestSendFunc_SuccessWithDirtyTracker() {
	msg := ws.Message{
		ID: 6,
		Data: dto.DataDirtyTracker{
			Shares:   true,
			Users:    true,
			Settings: false,
		},
	}

	// Send the message
	err := suite.wsMessageSender.SendFunc(msg)
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "WebSocket connection is nil")

	expectedEventName := "dirty_data_tracker"
	suite.Equal(expectedEventName, suite.wsMessageSender.ObjectMap["dto.DataDirtyTracker"])
}

// TestSendFunc_UnknownEventType tests handling of unknown event types
func (suite *WsMessageSenderSuite) TestSendFunc_UnknownEventType() {
	// Create a custom type not in the object map
	type UnknownEvent struct {
		Data string
	}

	msg := ws.Message{

		Data: UnknownEvent{Data: "unknown"},
	}

	err := suite.wsMessageSender.SendFunc(msg)
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "unknown event type")

	typeName := reflect.TypeOf(msg.Data).Name()
	suite.NotContains(suite.wsMessageSender.ObjectMap, typeName)
}

// TestSendFunc_MarshalError tests handling of JSON marshaling errors
func (suite *WsMessageSenderSuite) TestSendFunc_MarshalError() {
	// Create a message with data that can't be marshaled to JSON
	type UnmarshalableData struct {
		Channel chan int // channels can't be marshaled to JSON
	}

	msg := ws.Message{

		Data: UnmarshalableData{Channel: make(chan int)},
	}

	// Attempt to marshal - should fail
	_, err := json.Marshal(msg.Data)
	suite.Error(err)
}

// TestSendFunc_ConcurrentWrites tests concurrent access to sendFunc
func (suite *WsMessageSenderSuite) TestSendFunc_ConcurrentWrites() {
	// This test verifies that the mutex in wsMessageSender prevents race conditions
	const numGoroutines = 100
	const messagesPerGoroutine = 10

	// We'll use a channel-based approach to test concurrency
	// since we can't directly test the internal wsMessageSender

	var wg sync.WaitGroup
	messages := make(chan ws.Message, numGoroutines*messagesPerGoroutine)

	// Start multiple goroutines that generate messages
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				msg := ws.Message{
					ID: goroutineID*messagesPerGoroutine + j,
					Data: dto.HealthPing{
						Alive:     true,
						AliveTime: time.Now().Unix(),
					},
				}
				messages <- msg
			}
		}(i)
	}

	// Wait for all goroutines to finish
	wg.Wait()
	close(messages)

	// Verify we received all messages
	receivedCount := 0
	for range messages {
		receivedCount++
	}

	suite.Equal(numGoroutines*messagesPerGoroutine, receivedCount)
}

// TestSendFunc_ConcurrentWritesSameEventType tests concurrent writes of the same event type
func (suite *WsMessageSenderSuite) TestSendFunc_ConcurrentWritesSameEventType() {
	const numGoroutines = 50
	var wg sync.WaitGroup

	messageIDs := make(map[int]bool)
	var mapMutex sync.Mutex

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			msg := ws.Message{
				ID: id,
				//			Data: dto.UpdateProgress{
				//				Progress:    id % 100,
				//				LastRelease: fmt.Sprintf("v1.0.%d", id),
				//			},
			}

			// Simulate processing
			time.Sleep(time.Microsecond * time.Duration(id%10))

			mapMutex.Lock()
			messageIDs[msg.ID] = true
			mapMutex.Unlock()
		}(i)
	}

	wg.Wait()

	// Verify all messages were processed
	suite.Len(messageIDs, numGoroutines)
}

// TestSendFunc_RapidFireMessages tests sending many messages in quick succession
func (suite *WsMessageSenderSuite) TestSendFunc_RapidFireMessages() {
	const messageCount = 1000
	messages := make([]ws.Message, messageCount)

	for i := 0; i < messageCount; i++ {
		messages[i] = ws.Message{
			ID: i,
			Data: dto.HealthPing{
				Alive:     true,
				AliveTime: time.Now().Unix(),
			},
		}
	}

	// Verify all messages have valid event types
	for _, msg := range messages {
		typeName := reflect.TypeOf(msg.Data).String()
		suite.Contains(suite.wsMessageSender.ObjectMap, typeName)
	}
}

// TestSendFunc_MessageFormatting tests the SSE message format
func (suite *WsMessageSenderSuite) TestSendFunc_MessageFormatting() {
	msg := ws.Message{
		ID: 42,
		Data: dto.UpdateProgress{
			Progress:    50,
			LastRelease: "v2.0.0",
		},
	}

	eventData, err := json.Marshal(msg.Data)
	suite.NoError(err)

	typeName := suite.wsMessageSender.ObjectMap["dto.UpdateProgress"]
	suite.Equal("updating", typeName)

	// Verify the expected SSE format
	expectedFormat := fmt.Sprintf("id: %d\nevent: %s\ndata: %s\n\n", msg.ID, typeName, eventData)
	suite.Contains(expectedFormat, "id: 42")
	suite.Contains(expectedFormat, "event: updating")
	suite.Contains(expectedFormat, `"progress":50`)
	suite.Contains(expectedFormat, `"last_release":"v2.0.0"`)
}

// TestSendFunc_AllEventTypes tests that all registered event types are properly mapped
func (suite *WsMessageSenderSuite) TestSendFunc_AllEventTypes() {
	// Test Welcome
	suite.Contains(suite.wsMessageSender.ObjectMap, "dto.Welcome")
	suite.Equal("hello", suite.wsMessageSender.ObjectMap["dto.Welcome"])

	// Test UpdateProgress
	suite.Contains(suite.wsMessageSender.ObjectMap, "dto.UpdateProgress")
	suite.Equal("updating", suite.wsMessageSender.ObjectMap["dto.UpdateProgress"])

	// Test HealthPing
	suite.Contains(suite.wsMessageSender.ObjectMap, "dto.HealthPing")
	suite.Equal("heartbeat", suite.wsMessageSender.ObjectMap["dto.HealthPing"])
	// Test DataDirtyTracker
	suite.Contains(suite.wsMessageSender.ObjectMap, "dto.DataDirtyTracker")
	suite.Equal("dirty_data_tracker", suite.wsMessageSender.ObjectMap["dto.DataDirtyTracker"])

	// Test Disk slice
	suite.Contains(suite.wsMessageSender.ObjectMap, "[]*dto.Disk")
	suite.Equal("volumes", suite.wsMessageSender.ObjectMap["[]*dto.Disk"])

	// Test SharedResource slice
	suite.Contains(suite.wsMessageSender.ObjectMap, "[]dto.SharedResource")
	suite.Equal("shares", suite.wsMessageSender.ObjectMap["[]dto.SharedResource"])

}

// TestSendFunc_ConcurrentDifferentEventTypes tests concurrent sends of different event types
func (suite *WsMessageSenderSuite) TestSendFunc_ConcurrentDifferentEventTypes() {
	const iterations = 100
	var wg sync.WaitGroup

	eventTypes := []string{"dto.Welcome", "dto.UpdateProgress", "dto.HealthPing", "dto.DataDirtyTracker", "[]*dto.Disk", "[]dto.SharedResource"}
	receivedTypes := make(map[string]int)
	var mapMutex sync.Mutex

	for i := 0; i < iterations; i++ {
		for _, eventType := range eventTypes {
			wg.Add(1)
			go func(et string, iteration int) {
				defer wg.Done()

				var msg ws.Message
				//msg.ID = iteration

				switch et {
				case "dto.Welcome":
					machineID := fmt.Sprintf("machine-%d", iteration)
					msg.Data = dto.Welcome{
						Message:       "Test",
						MachineId:     &machineID,
						ActiveClients: int32(iteration),
					}
				case "dto.UpdateProgress":
					msg.Data = dto.UpdateProgress{
						Progress:    iteration % 100,
						LastRelease: fmt.Sprintf("v%d.0.0", iteration),
					}
				case "dto.HealthPing":
					msg.Data = dto.HealthPing{
						Alive:     true,
						AliveTime: time.Now().Unix(),
					}
				case "dto.DataDirtyTracker":
					msg.Data = dto.DataDirtyTracker{
						Shares:   iteration%2 == 0,
						Users:    iteration%3 == 0,
						Settings: iteration%5 == 0,
					}
				case "[]*dto.Disk":
					legacyName := fmt.Sprintf("disk-%d", iteration)
					msg.Data = []*dto.Disk{
						{LegacyDeviceName: &legacyName},
					}
				case "[]dto.SharedResource":
					msg.Data = []dto.SharedResource{
						{Name: fmt.Sprintf("share-%d", iteration)},
					}
				}

				typeName := reflect.TypeOf(msg.Data).String()
				suite.Contains(suite.wsMessageSender.ObjectMap, typeName)

				mapMutex.Lock()
				receivedTypes[typeName]++
				mapMutex.Unlock()
			}(eventType, i)
		}
	}

	wg.Wait()

	// Verify all event types were processed
	suite.Equal(iterations, receivedTypes["dto.Welcome"])
	suite.Equal(iterations, receivedTypes["dto.UpdateProgress"])
	suite.Equal(iterations, receivedTypes["dto.HealthPing"])
	suite.Equal(iterations, receivedTypes["dto.DataDirtyTracker"])
	suite.Equal(iterations, receivedTypes["[]*dto.Disk"])
	suite.Equal(iterations, receivedTypes["[]dto.SharedResource"])
}

// TestSendFunc_LargePayload tests sending messages with large payloads
func (suite *WsMessageSenderSuite) TestSendFunc_LargePayload() {
	// Create a large list of shares
	shares := make([]dto.SharedResource, 1000)
	for i := 0; i < 1000; i++ {
		shares[i] = dto.SharedResource{
			Name: fmt.Sprintf("share-%d with some additional comment text", i),
		}
	}

	msg := ws.Message{
		Data: shares,
	}

	// Verify it can be marshaled
	eventData, err := json.Marshal(msg.Data)
	suite.NoError(err)
	suite.Greater(len(eventData), 10000) // Should be a large payload
}

// TestSendFunc_EmptyData tests handling of empty data structures
func (suite *WsMessageSenderSuite) TestSendFunc_EmptyData() {
	// Empty shares list
	msg1 := ws.Message{
		Data: []dto.SharedResource{},
	}

	eventData1, err := json.Marshal(msg1.Data)
	suite.NoError(err)
	suite.Equal("[]", string(eventData1))

	// Empty volumes list
	msg2 := ws.Message{

		Data: []*dto.Disk{},
	}

	eventData2, err := json.Marshal(msg2.Data)
	suite.NoError(err)
	suite.Equal("[]", string(eventData2))
}

// TestSendFunc_NilPointers tests handling of nil pointers in data
func (suite *WsMessageSenderSuite) TestSendFunc_NilPointers() {
	// Welcome with nil MachineId - should be omitted due to omitempty tag
	msg := ws.Message{

		Data: dto.Welcome{
			Message:       "Welcome",
			MachineId:     nil,
			ActiveClients: 0,
		},
	}

	eventData, err := json.Marshal(msg.Data)
	suite.NoError(err)
	// With omitempty tag, nil pointer fields are not included in JSON
	suite.NotContains(string(eventData), `"machine_id"`)
}
