# DEBUG: {{ toJson . }}
[global]
   {{if not .local_master -}}
   local master = no
   {{- else -}}
   local master = yes
   {{- end }}
   {{if versionAtLeast .samba_version 4 23 -}}
   server smb transports = tcp{{if .smb_over_quic -}}, quic{{- end }}
   {{- end }}
   {{ if .compatibility_mode -}}
   client min protocol = NT1
   server min protocol = NT1
   {{- else -}}
   client min protocol = SMB2_10
   server min protocol = SMB2_10
   {{- end }}

   {{if not .multi_channel -}}
   server multi channel support = no
   {{- else -}}
   server multi channel support = yes
   {{- end }}

   {{if .smb_over_quic -}}
   # SMB over QUIC settings (requires Samba 4.23.0+)
   {{if versionAtLeast .samba_version 4 23 -}}
   smb3 unix extensions = yes
   tls enabled = yes 
   tls keyfile = /ssl/sambanas/server.key
   tls certfile = /ssl/sambanas/server.crt
   tls cafile = /ssl/sambanas/ca.crt
   #tls trust system cas = yes
   #tls verify peer = no_check
   {{- else -}}
   # WARNING: SMB over QUIC requires Samba 4.23.0+. Current version: {{ .samba_version }}
   # Falling back to standard SMB3 configuration
   {{- end }}
   {{- end }}

   unix extensions = no
   # --- Apple macOS/Time Machine compatibility (see vfs_fruit(8), Samba wiki) ---
   # The following fruit: options MUST be set in [global] to take effect (see vfs_fruit(8)):
   #   fruit:aapl           - Enables Apple's SMB2+ AAPL extension (performance, metadata)
   #   fruit:model          - Sets Finder icon (cosmetic, default: MacSamba)
   #   fruit:nfs_aces       - Controls NFS ACEs for UNIX mode (default: yes, set no for Mac clients)
   #   fruit:copyfile       - Enables Mac copyfile ioctl (default: no, set yes for full compatibility)
   # These are GLOBAL ONLY: setting them per-share has no effect.
   vfs objects = acl_xattr catia fruit 
   fruit:aapl = yes
   fruit:model = MacSamba
   fruit:nfs_aces = no
   fruit:copyfile = yes

   # The following fruit: options CAN be set globally or per-share (see vfs_fruit(8)):
   #   fruit:resource, fruit:metadata, fruit:veto_appledouble, fruit:wipe_intentionally_left_blank_rfork,
   #   fruit:zero_file_id, fruit:delete_empty_adfiles, fruit:posix_rename (needed for TM <4.23)
   # Best practice: set globally for consistent Mac behavior, override per-share only if needed.
   fruit:resource = file
   fruit:metadata = stream
   fruit:veto_appledouble = no
   {{if versionAtLeast .samba_version 4 22 -}}
   {{- else -}}
   fruit:posix_rename = yes
   {{- end }}
   fruit:wipe_intentionally_left_blank_rfork = yes
   fruit:zero_file_id = yes
   fruit:delete_empty_adfiles = yes

   # Spotlight backend (optional, for search integration)
   spotlight = yes

   # Performance Enhancements for network
   socket options = TCP_NODELAY IPTOS_LOWDELAY
   min receivefile size = 16384
   getwd cache = yes
   aio read size = 1
   aio write size = 1
   # End PR#167

   netbios name = {{ .hostname | default (env "HOSTNAME") | upper | trunc 15 | regexReplaceAll "[^A-Z0-9]" "-" }}
   workgroup = {{ .workgroup | default "NOWORKGROUP" | upper | trunc 15 | regexReplaceAll "[^A-Z0-9]" "-" }}
   server string = SambaNAS2 HomeAssistant
   multicast dns register = yes

   security = user
   username map = /etc/samba/smbusers

   # --- SMB signing and authentication compatibility for macOS 15+ (Tahoe) ---
   # See: https://wiki.samba.org/index.php/Configure_Samba_to_Work_Better_with_Mac_OS_X
   # server signing: required for macOS 15+ Time Machine, but only supported in Samba >= 4.0
   server signing = auto
   # ntlm auth: restrict to ntlmv2-only for security (Samba >= 4.8), but always allow for NT1 compatibility mode
   {{if .compatibility_mode -}}
   ntlm auth = yes
   {{- else -}}
   ntlm auth = ntlmv2-only
   {{- end }}
   {{if .allow_guest -}}
   guest account = nobody
   map to guest = Bad User
   {{- end }}
   idmap config * : backend = tdb
   idmap config * : range = 1000000-2000000

   load printers = no
   disable spoolss = yes

# DEBUG: Log Level: {{ .log_level }}
   debug class = yes
   {{ $log_level := dict "trace" "5" "debug" "auth_audit:2 auth:2 vfs:2" "info" "auth_audit:1 auth:1 vfs:1" "notice" "auth_audit:1 auth:0 vfs:0" "warning" "auth_audit:1 auth:0 vfs:0" "error" "auth_audit:0 auth:0 vfs:0"  "fatal" "0" -}}
   log level = {{ .log_level | default "fatal" | get $log_level }}

   bind interfaces only = {{ .bind_all_interfaces | default false | ternary "no" "yes" }}
   {{ if not .bind_all_interfaces -}}
   interfaces = {{ .interfaces | join " " }} {{ .docker_interface | default " "}}
   {{- end }}
   hosts allow = 127.0.0.1 {{ .allow_hosts | join " " }} {{ .docker_net | default " " }}

   mangled names = no
   dos charset = CP1253
   unix charset = UTF-8

{{ define "SHT" }}
{{- $unsupported := list "vfat"	"msdos"	"f2fs"	"fuseblk" "exfat" -}}
{{- $rosupported := list "apfs"}}
{{- $name := regexReplaceAll "[^A-Za-z0-9_/ ]" .data.name "_" | regexFind "[A-Za-z0-9_ ]+$" | upper -}}
[{{- $name -}}]
   browseable = yes
   writeable = {{ has .data.fs $rosupported | ternary "no" "yes" }}

   create mask = 0664
   force create mode = 0664
   directory mask = 0775
   force directory mode = 0775

   path = {{- if eq .data.name "config" }} /homeassistant{{- else }} {{ .data.path }}{{- end }}
   valid users =_ha_mount_user_ {{ .data.users|default .username|join " " }} {{ .data.ro_users|join " " }}
   {{ if .data.ro_users -}}
   read list = {{ .data.ro_users|join " " }}
   {{- end }}
   force user = root
   force group = root

   {{ if and .data.veto_files (gt (len .data.veto_files) 0) -}}
   veto files = /{{ .data.veto_files | join "/" }}/
   delete veto files = yes
   {{- end }}

   {{ if .data.GuestOk -}}
   guest ok = yes
   {{- end }}

# DEBUG: {{ toJson .data  }}|$name={{ $name }}|.shares={{ .shares }}|

{{if .data.recycle_bin_enabled }}
   recycle:repository = .recycle/%U
   recycle:keeptree = yes
   recycle:versions = yes
   recycle:touch = yes
   recycle:touch_mtime = no
   recycle:directory_mode = 0777
   #recycle:subdir_mode = 0700
   #recycle:exclude =
   #recycle:exclude_dir =
   #recycle:maxsize = 0
{{ end }}

# TM:{{ if has .data.fs $unsupported }}unsupported{{else}}{{ .data.timemachine }}{{ end }} US:{{ .data.users|default .username|join "," }} {{ .data.ro_users|join "," }}{{- if .medialibrary.enable }}{{ if .data.usage }} CL:{{ .data.usage }}{{ end }} FS:{{ .data.fs | default "native" }} {{ if .data.recycle_bin_enabled }}RECYCLEBIN{{ end }} {{ end }}
# Note:"Setting vfs objects in a share will overwrite the globally configured option, it will NOT supplement them."
{{- if and .data.timemachine (has .data.fs $unsupported | not ) }}
   vfs objects = acl_xattr catia fruit streams_xattr{{- if .data.recycle_bin_enabled -}} recycle{{- end }}

   # Time Machine Settings Ref: https://github.com/markthomas93/samba.apple.templates
   fruit:time machine = yes
   {{ if .data.TimeMachineMaxSize -}}
   fruit:time machine max size = {{ .data.TimeMachineMaxSize }}
   {{- end }}
{{ else }}
   vfs objects = acl_xattr catia fruit{{- if .data.recycle_bin_enabled }} recycle{{- end }}

{{ end }}

{{ end }}

{{- $root := . -}}
{{- range $dd := .shares -}}
               {{- if not $dd.disabled -}}
                  {{- $root2 := deepCopy $root -}}
                  {{- $_ := set $root2 "data" $dd -}}
                  {{- template "SHT" $root2 -}}
               {{- end -}}
        {{/* - end - */}}
{{- end -}}
