/**
 * Manual MSW handlers for WebSocket connections.
 * These handlers provide mock streaming data for testing real-time features.
 * 
 * Note: SSE (Server-Sent Events) is deprecated for this project and not implemented.
 * Use WebSocket for real-time streaming instead.
 */

import { ws } from "msw";
import type {
	DataDirtyTracker,
	Disk,
	HealthPing,
	SharedResource,
	SmartTestStatus,
	UpdateProgress,
	Welcome,
} from "../store/sratApi";
import { Supported_events, Update_channel, Update_process_state } from "../store/sratApi";

/**
 * Mock data generators for WebSocket events
 */
const mockEventData = {
	hello: (): Welcome => ({
		message: "Welcome to SRAT (Mocked)",
		active_clients: 1,
		supported_events: Object.values(Supported_events),
		update_channel: Update_channel.Develop,
		build_version: "2026.1.0-dev-mock",
		secure_mode: true,
		protected_mode: false,
		read_only: false,
		startTime: Date.now(),
	}),

	heartbeat: (): HealthPing => ({
		alive: true,
		aliveTime: Date.now(),
		samba_process_status: {},
		last_error: "",
		dirty_tracking: {
			shares: false,
			users: false,
			settings: false,
		},
		update_available: false,
		addon_stats: {
			cpu_percent: 5.2,
			memory_percent: 12.5,
			memory_usage: 104857600,
			memory_limit: 1073741824,
			network_rx: 1024,
			network_tx: 2048,
			blk_read: 512,
			blk_write: 1024,
		},
		disk_health: {
			global: {
				total_iops: 150,
				total_read_latency_ms: 5.2,
				total_write_latency_ms: 3.8,
			},
			per_disk_io: [],
			per_partition_info: {},
			hdidle_running: false,
		},
		network_health: {
			global: {
				totalInboundTraffic: 1024,
				totalOutboundTraffic: 2048,
			},
			perNicIO: [],
		},
		samba_status: {
			timestamp: new Date().toISOString(),
			version: "4.23.0",
			smb_conf: "/etc/samba/smb.conf",
			sessions: {},
			tcons: {},
		},
		uptime: 3600,
	}),

	volumes: (): Disk[] => [],

	shares: (): SharedResource[] => [],

	updating: (): UpdateProgress => ({
		update_process_state: Update_process_state.Idle,
		progress: 0,
		//last_release: "",
		error_message: "",
	}),

	dirty_data_tracker: (): DataDirtyTracker => ({
		shares: false,
		users: false,
		settings: false,
	}),

	smart_test_status: (): SmartTestStatus => ({
		disk_id: "test-disk-1",
		running: false,
		status: "completed",
		test_type: "short",
		percent_complete: 100,
		lba_of_first_error: "",
	}),
};

/**
 * WebSocket handler for /ws endpoint using MSW's experimental ws API
 * 
 * This implementation uses MSW's native WebSocket support to mock WebSocket connections.
 * The handler listens for SUBSCRIBE messages and responds with mocked event data.
 * 
 * Features:
 * - Automatic hello message on connection
 * - Periodic heartbeat messages (500ms intervals)
 * - SUBSCRIBE message handling for on-demand event data
 * - Proper cleanup on disconnect
 */
const wsUrl = "ws://localhost:8080/ws";

export const wsLink = ws.link(wsUrl);

export const wsHandler = wsLink.addEventListener("connection", ({ client }) => {
	// Send initial hello message when connection is established
	const helloMessage = `id: 1\nevent: ${Supported_events.Hello}\ndata: ${JSON.stringify(mockEventData.hello())}`;
	client.send(helloMessage);

	// Set up periodic heartbeat
	const heartbeatInterval = setInterval(() => {
		const heartbeatMessage = `id: ${Date.now()}\nevent: ${Supported_events.Heartbeat}\ndata: ${JSON.stringify(mockEventData.heartbeat())}`;
		client.send(heartbeatMessage);
	}, 500);

	// Listen for SUBSCRIBE messages
	client.addEventListener("message", (event) => {
		try {
			const message = JSON.parse(event.data.toString());
			
			if (message.type === "SUBSCRIBE") {
				// Respond with the requested event type
				const eventType = message.event as Supported_events;
				const eventData = mockEventData[eventType as keyof typeof mockEventData];
				
				if (eventData) {
					const responseMessage = `id: ${Date.now()}\nevent: ${eventType}\ndata: ${JSON.stringify(eventData())}`;
					client.send(responseMessage);
				}
			}
		} catch (error) {
			console.error("Error handling WebSocket message:", error);
		}
	});

	// Clean up on disconnect
	client.addEventListener("close", () => {
		clearInterval(heartbeatInterval);
	});
});

/**
 * Export all streaming handlers
 * 
 * Note: Only WebSocket handlers are exported. SSE is deprecated for this project.
 */
export const streamingHandlers: never[] = [];
