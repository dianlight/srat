import type { MountPointData } from "./sratApi";

export enum TabIDs {
	DASHBOARD = 0,
	SHARES,
	VOLUMES,
	USERS,
	SETTINGS,
	SMB_FILE_CONFIG,
	API_OPENDOC,
}

export interface LocationState {
	tabId?: TabIDs;
	shareName?: string;
	newShareData?: MountPointData;
	mountPathHashToView?: string;
}
