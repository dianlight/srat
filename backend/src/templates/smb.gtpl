# DEBUG: {{ toJson . }}
[global]
   {{if not .local_master -}}
   local master = no
   {{- else -}}
   local master = yes
   {{- end }}
   server smb transports = tcp{{if .smb_over_quic -}}, quic{{- end }}
   {{ if .compatibility_mode -}}
   client min protocol = NT1
   server min protocol = NT1
   {{- else -}}
   server min protocol = SMB2_10
   client min protocol = SMB2_10
   {{- end }}

   {{if not .multi_channel -}}
   server multi channel support = no
   {{- else -}}
   server multi channel support = yes
   {{- end }}

   {{if .smb_over_quic -}}
   # SMB over QUIC settings
   server smb3 encryption = mandatory
   smb3 unix extensions = yes
   tls enable = yes 
   tls keyfile = /config/server.key
   tls certfile = /config/server.cert
   #tls trust system cas = yes
   #tls verify peer = no_check
   {{- end }}

   unix extensions = no
   vfs objects = acl_xattr catia fruit streams_xattr recycle
   fruit:aapl = yes
   fruit:model = MacSamba

   fruit:resource = file
   fruit:veto_appledouble = no
   fruit:posix_rename = yes
   fruit:wipe_intentionally_left_blank_rfork = yes
   fruit:zero_file_id = yes
   fruit:delete_empty_adfiles = yes
   fruit:copyfile = yes
   fruit:nfs_aces = no
   fruit:metadata = stream
   fruit:veto_appledouble = no
   spotlight = yes

   # Performance Enhancements for network
   socket options = TCP_NODELAY IPTOS_LOWDELAY
   min receivefile size = 16384
   getwd cache = yes
   aio read size = 1
   aio write size = 1
   # End PR#167

   netbios name = {{ .hostname | default (env "HOSTNAME") }}
   workgroup = {{ .workgroup | default "NOWORKGROUP" }}
   server string = Samba NAS2 HomeAssistant %v
   multicast dns register = yes

   security = user
   ntlm auth = yes
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
   interfaces = lo {{ .interfaces | join " " }} {{ .docker_interface | default " "}}
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
{{- if and .data.timemachine (has .data.fs $unsupported | not ) }}
   vfs objects = catia fruit streams_xattr{{ if .data.recycle_bin_enabled }} recycle{{ end }}

   # Time Machine Settings Ref: https://github.com/markthomas93/samba.apple.templates
   fruit:time machine = yes
   {{ if .data.TimeMachineMaxSize -}}
   fruit:time machine max size = {{ .data.TimeMachineMaxSize }}
   {{- end }}
   fruit:metadata = stream
{{ else }}
   vfs objects = catia{{ if .data.recycle_bin_enabled }} recycle{{ end }}{{/*- printf "/*%#v* /" . -*/}}
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
