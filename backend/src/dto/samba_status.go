package dto

import "time"

// CustomTime is a custom time.Time type that handles the specific date format from smbstatus.
type CustomTime struct {
	time.Time
}

// UnmarshalJSON implements the json.Unmarshaler interface for CustomTime.
func (ct *CustomTime) UnmarshalJSON(b []byte) (err error) {
	s := string(b)
	// Remove quotes from the string
	s = s[1 : len(s)-1]

	// Try parsing with the expected smbstatus format
	// Example: "2025-06-28T15:04:28.288225+0200"
	// Go layout: "2006-01-02T15:04:05.999999-0700"
	t, err := time.Parse("2006-01-02T15:04:05.999999-0700", s)
	if err != nil {
		// Fallback to RFC3339 if the custom format fails (e.g., for other time fields)
		t, err = time.Parse(time.RFC3339, s)
		if err != nil {
			return err
		}
	}
	ct.Time = t
	return nil
}

type SambaStatus struct {
	Timestamp CustomTime              `json:"timestamp"`
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
	ConnectTime CustomTime `json:"connect_time"`
}
