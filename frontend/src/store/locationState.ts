import type { Volumes } from "../pages/Volumes"

export enum TabIDs {
    SHARES = 0,
    VOLUMES,
    USERS,
    SETTINGS,
    SMB_FILE_CONFIG,
    API_OPENDOC
}


export interface LocationState {
    tabId?: TabIDs
    shareName?: string
}