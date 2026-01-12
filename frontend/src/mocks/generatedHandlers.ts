/**
 * Auto-generated REST API handlers using msw-auto-mock.
 * 
 * This file is generated from the OpenAPI 3.1 specification and provides
 * mock handlers for all REST endpoints defined in the API.
 * 
 * To regenerate this file, run:
 *   bunx msw-auto-mock <path-to-openapi-spec> -o src/mocks/generatedHandlers.ts
 * 
 * @see https://www.npmjs.com/package/msw-auto-mock
 */

import { http, type RequestHandler } from "msw";

/**
 * OpenAPI specification path
 * This should point to your OpenAPI 3.1 JSON or YAML file
 */
const OPENAPI_SPEC_PATH = "../../backend/docs/openapi.json";

/**
 * Auto-generated handlers from OpenAPI specification
 * 
 * These handlers will automatically mock all REST endpoints defined in the OpenAPI spec
 * with realistic data that matches the schema definitions.
 */

// Placeholder for generated handlers
// To generate, run: npm run gen:mocks
export const generatedHandlers: RequestHandler[] = [
	// Example: Health endpoint mock
	http.get("/api/health", () => {
		return new Response(
			JSON.stringify({
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
			{
				status: 200,
				headers: {
					"Content-Type": "application/json",
				},
			},
		);
	}),

	// Example: Shares list endpoint mock
	http.get("/api/shares", () => {
		return new Response(
			JSON.stringify([
				{
					name: "share1",
					disabled: false,
					guest_ok: false,
					timemachine: false,
					timemachine_max_size: "0",
					usage: "share",
					users: [],
					ro_users: [],
					veto_files: [],
					recycle_bin_enabled: true,
					status: {
						is_valid: true,
						is_ha_mounted: false,
					},
				},
			]),
			{
				status: 200,
				headers: {
					"Content-Type": "application/json",
				},
			},
		);
	}),

	// Example: Volumes list endpoint mock
	http.get("/api/volumes", () => {
		return new Response(JSON.stringify([]), {
			status: 200,
			headers: {
				"Content-Type": "application/json",
			},
		});
	}),

	// Example: Settings endpoint mock
	http.get("/api/settings", () => {
		return new Response(
			JSON.stringify({
				workgroup: "WORKGROUP",
				hostname: "srat-mock",
				allow_guest: false,
				compatibility_mode: false,
				log_level: "info",
				interfaces: [],
				allow_hosts: [],
				bind_all_interfaces: true,
				local_master: true,
				multi_channel: false,
				smb_over_quic: true,
				export_stats_to_ha: true,
				telemetry_mode: "Disabled",
				hdidle_enabled: false,
				hdidle_default_idle_time: 600,
				hdidle_default_power_condition: 0,
				hdidle_default_command_type: "scsi",
				hdidle_ignore_spin_down_detection: false,
				mountoptions: [],
			}),
			{
				status: 200,
				headers: {
					"Content-Type": "application/json",
				},
			},
		);
	}),
];

/**
 * Export generated handlers
 * 
 * Note: This is a placeholder. For full auto-generation from OpenAPI spec,
 * install and configure msw-auto-mock:
 * 
 * 1. Add script to package.json: "gen:mocks": "msw-auto-mock ../../backend/docs/openapi.json -o src/mocks/generatedHandlers.ts"
 * 2. Run: npm run gen:mocks
 * 3. The file will be regenerated with all endpoints from the OpenAPI spec
 */
