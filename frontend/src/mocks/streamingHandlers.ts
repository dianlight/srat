/**
 * Manual MSW handlers for Server-Sent Events (SSE) and WebSocket connections.
 * These handlers provide mock streaming data for testing real-time features.
 */

import { http, HttpResponse, type RequestHandler } from "msw";
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
 * Mock data generators for SSE events
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
		last_release: "",
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
 * SSE Handler for /api/sse endpoint
 * Emits JSON events every 500ms using ReadableStream
 */
export const sseHandler: RequestHandler = http.get(
	"/api/sse",
	async ({ request }) => {
		// Create a ReadableStream that emits events periodically
		const stream = new ReadableStream({
			async start(controller) {
				// Send initial hello event
				const helloEvent = `event: ${Supported_events.Hello}\ndata: ${JSON.stringify(mockEventData.hello())}\n\n`;
				controller.enqueue(new TextEncoder().encode(helloEvent));

				// Set up interval to send heartbeat events
				const intervalId = setInterval(() => {
					try {
						const heartbeatEvent = `event: ${Supported_events.Heartbeat}\ndata: ${JSON.stringify(mockEventData.heartbeat())}\n\n`;
						controller.enqueue(new TextEncoder().encode(heartbeatEvent));
					} catch (error) {
						// Stream was closed
						clearInterval(intervalId);
					}
				}, 500);

				// Clean up on stream cancel
				request.signal.addEventListener("abort", () => {
					clearInterval(intervalId);
					controller.close();
				});
			},
		});

		return new HttpResponse(stream, {
			status: 200,
			headers: {
				"Content-Type": "text/event-stream",
				"Cache-Control": "no-cache",
				Connection: "keep-alive",
			},
		});
	},
);

/**
 * WebSocket link for /ws endpoint
 * Handles SUBSCRIBE messages and responds with mocked data
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
 */
export const streamingHandlers = [sseHandler];
