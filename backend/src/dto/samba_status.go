package dto

import "time"

type SambaStatus struct {
	Timestamp time.Time              `json:"timestamp"`
	Version   string                 `json:"version"`
	SmbConf   string                 `json:"smb_conf"`
	Sessions  map[string]SambaSession `json:"sessions"`
	Tcons     map[string]SambaTcon    `json:"tcons"`
}

type SambaSession struct {
	SessionID    uint64 `json:"session_id"`
	UserID       uint64 `json:"uid"`
	GroupID      uint64 `json:"gid"`
	Username     string `json:"username"`
	Groupname    string `json:"groupname"`
	RemoteMachine string `json:"remote_machine"`
	Hostname     string `json:"hostname"`
	SessionDialect string `json:"session_dialect"`
	IsEncrypted  bool   `json:"is_encrypted"`
	Signing      string `json:"signing"`
}

type SambaTcon struct {
	TconID      uint64    `json:"tcon_id"`
	SessionID   uint64    `json:"session_id"`
	Share       string    `json:"share"`
	Device      string    `json:"device"`
	ConnectTime time.Time `json:"connect_time"`
}
