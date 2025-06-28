
export interface ProcessStatus {
    name: string;
    pid: number | null;
    status: 'Running' | 'Stopped';
    cpu: number | null;
    connections: number | null;
    memory: number | null;
}

export interface AddonStatsData {
    blk_read?: number | null;
    blk_write?: number | null;
    cpu_percent?: number | null;
    memory_limit?: number | null;
    memory_percent?: number | null;
    memory_usage?: number | null;
    network_rx?: number | null;
    network_tx?: number | null;
}
