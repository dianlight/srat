import type { MountPointData } from "./sratApi"

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
    newShareData?: MountPointData
}