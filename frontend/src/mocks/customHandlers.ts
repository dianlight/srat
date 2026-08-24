/**
 * Custom MSW request handlers
 *
 * These handlers are manually defined to mock specific API endpoints
 * with custom logic or data that may not be covered by auto-generated handlers.
 *
 * @see https://www.npmjs.com/package/msw-auto-mock
 */

import { http, HttpResponse, type RequestHandler } from "msw";
import { Status2, type LabFeature } from "../store/sratApi";

const filesystemSupportOverrides = new Map<string, Record<string, unknown>>();

export function setFilesystemSupportOverride(
	fsType: string,
	response: Record<string, unknown>,
) {
	filesystemSupportOverrides.set(fsType.toLowerCase(), response);
}

export function clearFilesystemSupportOverrides() {
	filesystemSupportOverrides.clear();
}

/**
 * In-memory store backing the rclone cloud-sync mock endpoints. Tests can
 * seed a link before rendering to simulate the "linked" state.
 */
export const rcloneMockState = {
	link: null as Record<string, unknown> | null,
	/** Body of the last POST …/sync request, for asserting dry_run etc. */
	lastSyncBody: null as Record<string, unknown> | null,
};

export function resetRcloneMockState() {
	rcloneMockState.link = null;
	rcloneMockState.lastSyncBody = null;
}

const rcloneDropboxProvider = {
	name: "dropbox",
	display_name: "Dropbox",
	config_fields: [
		{
			name: "client_id",
			label: "App key",
			description: "Dropbox OAuth app key",
			required: true,
			secret: false,
		},
		{
			name: "client_secret",
			label: "App secret",
			description: undefined,
			required: true,
			secret: true,
		},
	],
};

function jsonResponse(body: unknown, status = 200) {
	return new Response(JSON.stringify(body), {
		status,
		headers: { "Content-Type": "application/json" },
	});
}

/**

 * 
 * These handlers will automatically mock all REST endpoints defined in the OpenAPI spec
 * with realistic data that matches the schema definitions.
 */

// Placeholder for generated handlers
// To generate, run: npm run gen:mocks
export const customHandlers: RequestHandler[] = [
	// Deterministic filesystem state endpoint mock used by volume detail tests
	http.get(/.*\/api\/filesystem\/state(?:\?.*)?$/, ({ request }) => {
		const url = new URL(request.url);
		const partitionId = url.searchParams.get("partition_id");

		if (!partitionId) {
			return HttpResponse.json(
				{ message: "partition_id is required" },
				{ status: 400 },
			);
		}

		return HttpResponse.json({
			isClean: true,
			hasErrors: false,
			isMounted: true,
			stateDescription: "Filesystem is clean",
			additionalInfo: {
				"Last check": "2026-02-10",
			},
		});
	}),

	// Deterministic filesystem support endpoint mock used by volume dialog tests.
	http.get(/.*\/api\/filesystem\/support(?:\?.*)?$/, ({ request }) => {
		const url = new URL(request.url);
		const fsType = (url.searchParams.get("fstype") ?? "").toLowerCase();
		const override = filesystemSupportOverrides.get(fsType);

		if (override) {
			return HttpResponse.json(override);
		}

		const supportByFsType: Record<string, Record<string, unknown>> = {
			ext4: {
				canMount: true,
				canFormat: true,
				canCheck: true,
				canSetLabel: true,
				canGetState: true,
				isExportable: false,
				isFormatReportProgress: false,
				isCheckReportProgress: false,
				labelRule: "^[^\\x00/]{1,16}$",
				alpinePackage: "e2fsprogs",
				missingTools: [],
			},
			vfat: {
				canMount: true,
				canFormat: true,
				canCheck: true,
				canSetLabel: true,
				canGetState: true,
				isExportable: false,
				isFormatReportProgress: false,
				isCheckReportProgress: false,
				labelRule: "^[A-Za-z0-9 ]{1,11}$",
				alpinePackage: "dosfstools",
				missingTools: [],
			},
			f2fs: {
				canMount: true,
				canFormat: true,
				canCheck: true,
				canSetLabel: false,
				canGetState: true,
				isExportable: false,
				isFormatReportProgress: false,
				isCheckReportProgress: false,
				labelRule: "^[^\\x00/]{1,512}$",
				alpinePackage: "f2fs-tools",
				missingTools: [],
			},
			zfs: {
				canMount: true,
				canFormat: false,
				canCheck: false,
				canSetLabel: false,
				canGetState: true,
				isExportable: false,
				isFormatReportProgress: false,
				isCheckReportProgress: false,
				labelRule: "",
				alpinePackage: "",
				missingTools: [],
			},
		};

		const fallback = {
			canMount: true,
			canFormat: true,
			canCheck: true,
			canSetLabel: true,
			canGetState: true,
			isExportable: false,
			isFormatReportProgress: false,
			isCheckReportProgress: false,
			labelRule: "",
			alpinePackage: "",
			missingTools: [],
		};

		return HttpResponse.json(supportByFsType[fsType] ?? fallback);
	}),

	// Deterministic filesystem label update endpoint used by volume relabel tests.
	http.put(/.*\/api\/filesystem\/label(?:\?.*)?$/, async ({ request }) => {
		const body = (await request.json().catch(() => ({}))) as {
			partitionId?: string;
			label?: string;
		};

		return HttpResponse.json({
			success: true,
			partitionId: body.partitionId ?? "",
			label: body.label ?? "",
		});
	}),
	// Deterministic GitHub discussions endpoint used by dashboard news widgets.
	http.get("https://api.github.com/repos/:owner/:repo/discussions", ({ request, params }) => {
		const url = new URL(request.url);
		const category = url.searchParams.get("category");
		if (category !== "announcements") {
			return HttpResponse.json([]);
		}

		return HttpResponse.json([
			{
				id: 1,
				number: 1,
				title: `${String(params.repo)} announcement`,
				html_url: `https://github.com/${String(params.owner)}/${String(params.repo)}/discussions/1`,
				created_at: "2026-09-01T12:00:00.000Z",
				body: "Mock announcement abstract for dashboard news widget.",
			},
		]);
	}),

	// Example: Health endpoint mock
	http.get(/.*\/api\/health(?:\?.*)?$/, () => {
		return HttpResponse.json({
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
		});
	}),

	// Example: Shares list endpoint mock
	http.get(/.*\/api\/shares(?:\?.*)?$/, () => {
		return HttpResponse.json([
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
		]);
	}),

	// Example: Volumes list endpoint mock
	http.get(/.*\/api\/volumes(?:\?.*)?$/, () => {
		return HttpResponse.json([]);
	}),

	// Lab feature registry mock — mirrors the settings mock above
	// (experimental_lab_mode: true), so every beta feature is available and
	// the alpha feature is present (tests run in a non-production env).
	http.get(/.*\/api\/lab_features(?:\?.*)?$/, () => {
		const labFeatures: LabFeature[] = [
			{ key: "hdidle", name: "HDIdle per-disk control", description: "Per-disk spin-down control.", status: Status2.Beta, available: true },
			{ key: "smb_conf", name: "smb.conf view", description: "Read-only view of the generated smb.conf.", status: Status2.Beta, available: true },
			{ key: "ha_use_nfs", name: "Use NFS for Home Assistant", description: "Mount Home Assistant shares with NFS.", status: Status2.Beta, available: true },
			{ key: "ha_custom_component", name: "Home Assistant custom component tools", description: "Install, upgrade and uninstall the SRAT custom component.", status: Status2.Alpha, available: true },
			{ key: "smb_over_quic", name: "SMB over QUIC", description: "Expose SMB shares over the QUIC transport.", status: Status2.Beta, available: true },
			{ key: "addon_mdns", name: "Add-on side mDNS registration", description: "Zeroconf mDNS registration of the Samba service.", status: Status2.Beta, available: true },
		];
		return HttpResponse.json(labFeatures);
	}),

	// Deterministic volume mount endpoint used by mount dialog/hook tests.
	http.post(/.*\/api\/volume\/mount(?:\?.*)?$/, async ({ request }) => {
		const body = (await request.json().catch(() => ({}))) as Record<
			string,
			unknown
		>;

		return HttpResponse.json({
			path:
				(typeof body.path === "string" && body.path) || "/mnt/test-volume",
			type: "HOST",
			is_mounted: true,
			is_write_supported: true,
			fstype:
				(typeof body.fstype === "string" && body.fstype) || "ext4",
			is_to_mount_at_startup: false,
		});
	}),

	// Deterministic SMART status endpoint used by SmartStatusPanel tests.
	http.get(/.*\/api\/disk\/.+\/smart\/status(?:\?.*)?$/, () => {
		return HttpResponse.json({
			enabled: true,
			in_standby: false,
			is_in_danger: false,
			is_in_warning: false,
			is_test_passed: true,
			is_test_running: false,
			others: {},
			power_cycle_count: { value: 500 },
			power_on_hours: { value: 10000 },
			temperature: { value: 35, min: 20, max: 45 },
		});
	}),

	// Deterministic SMART self-test status endpoint used by SmartStatusPanel tests.
	http.get(/.*\/api\/disk\/.+\/smart\/test(?:\?.*)?$/, () => {
		return HttpResponse.json({
			disk_id: "mock-disk",
			lba_of_first_error: "",
			percent_complete: 0,
			running: false,
			status: "idle",
			test_type: "none",
		});
	}),

	// Deterministic SMART operation endpoints (start/abort/enable/disable).
	http.post(/.*\/api\/disk\/.+\/smart\/test\/start(?:\?.*)?$/, () => {
		return HttpResponse.json("started");
	}),
	http.post(/.*\/api\/disk\/.+\/smart\/test\/abort(?:\?.*)?$/, () => {
		return HttpResponse.json("aborted");
	}),
	http.post(/.*\/api\/disk\/.+\/smart\/enable(?:\?.*)?$/, () => {
		return HttpResponse.json("enabled");
	}),
	http.post(/.*\/api\/disk\/.+\/smart\/disable(?:\?.*)?$/, () => {
		return HttpResponse.json("disabled");
	}),

	// Example: Settings endpoint mock
	http.get(/.*\/api\/settings(?:\?.*)?$/, () => {
		return HttpResponse.json({
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
			experimental_lab_mode: true,
			mountoptions: [],
		});
	}),

	// Deterministic problem upsert endpoint used by dashboard ignore tests.
	// Echoes the request body so PUT /api/problems/:problemKey is observable
	// in tests (issue 1302: IGNORE must reach the backend).
	http.put(/.*\/api\/problems\/.+(?:\?.*)?$/, async ({ request }) => {
		const body = (await request.json().catch(() => ({}))) as Record<
			string,
			unknown
		>;
		return HttpResponse.json(body);
	}),

	// OAuth broker mocks – validate SRAT-Signature (Ed25519) instead of Bearer
	// Client registration is public (self-certifying), other endpoints require SRAT-Signature
	http.post(/.*\/v1\/clients(?:\/.*)?$/, async ({ request }) => {
		const body = (await request.json().catch(() => ({}))) as { client_id?: string; public_key?: string };
		if (!body.client_id || !body.public_key) {
			return HttpResponse.json({ error: "client_id and public_key required" }, { status: 400 });
		}
		// In mock we accept any client_id/public_key that matches 43-char/32B format – real broker recomputes SHA256
		return HttpResponse.json(
			{ client_id: body.client_id, public_key: body.public_key, created_at: new Date().toISOString() },
			{ status: 201, headers: { "Cache-Control": "no-store" } },
		);
	}),
	http.post(/.*\/v1\/instances\/register(?:\?.*)?$/, ({ request }) => {
		const auth = request.headers.get("authorization") || "";
		if (!auth.startsWith("SRAT-Signature ")) {
			return HttpResponse.json({ error: "missing or malformed SRAT-Signature" }, { status: 401 });
		}
		return HttpResponse.json(
			{ instance_id: "mock-instance", redirect_url: "https://srat.example/cb", client_id: "mock-client", expires_at: new Date(Date.now() + 3600000).toISOString(), ttl_seconds: 3600 },
			{ status: 200, headers: { "Cache-Control": "no-store" } },
		);
	}),
	http.post(/.*\/v1\/start(?:\?.*)?$/, ({ request }) => {
		const auth = request.headers.get("authorization") || "";
		if (!auth.startsWith("SRAT-Signature ")) {
			return HttpResponse.json({ error: "missing SRAT-Signature" }, { status: 401 });
		}
		return HttpResponse.json(
			{ auth_url: "https://example.com/auth?code_challenge=mock", session_id: "mock-session-id" },
			{ status: 200, headers: { "Cache-Control": "no-store" } },
		);
	}),
	http.get(/.*\/v1\/session\/.*/, ({ request }) => {
		const auth = request.headers.get("authorization") || "";
		if (!auth.startsWith("SRAT-Signature ")) {
			return HttpResponse.json({ error: "missing SRAT-Signature" }, { status: 401 });
		}
		return HttpResponse.json(
			{ token_json: `{"access_token":"mock-at"}`, account_label: "mock", client_id: "dropbox-id", client_secret: "dropbox-secret" },
			{ status: 200, headers: { "Cache-Control": "no-store" } },
		);
	}),

	// Rclone cloud-sync (lab feature) endpoint mocks
	http.get(/.*\/api\/rclone\/providers(?:\?.*)?$/, () => {
		return jsonResponse({
			library_available: true,
			providers: [rcloneDropboxProvider],
		});
	}),

	http.get(/.*\/api\/rclone\/link\/([^/]+)\/(.+)$/, () => {
		if (!rcloneMockState.link) {
			return jsonResponse({ detail: "rclone link not found" }, 404);
		}
		return jsonResponse(rcloneMockState.link);
	}),

	http.put(/.*\/api\/rclone\/link\/([^/]+)\/(.+)$/, async ({ request }) => {
		const body = (await request.json()) as Record<string, unknown>;
		rcloneMockState.link = {
			target_kind: "volume",
			target_id: "/mnt/usb",
			status: "unlinked",
			auto_sync: false,
			schedule_minutes: 0,
			...body,
		};
		return jsonResponse(rcloneMockState.link);
	}),

	http.delete(/.*\/api\/rclone\/link\/([^/]+)\/(.+)$/, () => {
		rcloneMockState.link = null;
		return new Response(null, { status: 204 });
	}),

	http.post(/.*\/api\/rclone\/link\/([^/]+)\/(.+)\/auth\/start$/, () => {
		return jsonResponse({
			auth_url: "https://www.dropbox.com/oauth2/authorize?mock=1",
			redirect_uri: "http://localhost/api/rclone/oauth/callback",
			state: "mock-state",
		});
	}),

	http.post(/.*\/api\/rclone\/link\/([^/]+)\/(.+)\/diff$/, () => {
		return jsonResponse({
			local_only: 1,
			remote_only: 1,
			changed: 1,
			entries: [
				{
					path: "photos/",
					diff_type: "local_only",
					local_size: 1288490188,
				},
				{ path: "backup.zip", diff_type: "remote_only", remote_size: 92274688 },
				{
					path: "notes.txt",
					diff_type: "changed",
					local_size: 4096,
					remote_size: 2048,
				},
			],
		});
	}),

	http.post(/.*\/api\/rclone\/link\/([^/]+)\/(.+)\/sync$/, async ({ request }) => {
		rcloneMockState.lastSyncBody = (await request.json()) as Record<string, unknown>;
		return jsonResponse({});
	}),

	http.post(/.*\/api\/rclone\/link\/([^/]+)\/(.+)\/abort$/, () => {
		return jsonResponse({});
	}),
];
