[global]
   {{ if .compatibility_mode }}
   client min protocol = NT1
   server min protocol = NT1
   {{ else }}
   server min protocol = SMB2_10
   client min protocol = SMB2_10
   {{ end }}

   {{if .multi_channel }}
   server multi channel support = yes
   aio read size = 1
   aio write size = 1
   {{ end }}  

   dns proxy = yes 

   ea support = yes
   vfs objects = catia fruit streams_xattr{{ if .recyle_bin_enabled }} recycle{{ end }}  
   fruit:aapl = yes
   fruit:model = MacSamba

   fruit:resource = file
   fruit:veto_appledouble = no
   fruit:posix_rename = yes 
   fruit:wipe_intentionally_left_blank_rfork = yes
   fruit:zero_file_id = yes
   fruit:delete_empty_adfiles = yes

   # cherry pick from PR#167 to Test
   fruit:copyfile = yes
   fruit:nfs_aces = no

   # Performance Enhancements for network
   socket options = TCP_NODELAY IPTOS_LOWDELAY
   min receivefile size = 16384
   getwd cache = yes
   aio read size = 1
   aio write size = 1  
   # End PR#167

   netbios name = {{ env "HOSTNAME" }}
   workgroup = {{ .workgroup | default "NOWORKGROUP" }}
   server string = Samba NAS HomeAssistant config
   multicast dns register = {{ if or .wsdd .wsdd2 }}no{{ else }}yes{{ end }}

   security = user
   ntlm auth = yes
   idmap config * : backend = tdb
   idmap config * : range = 1000000-2000000

   load printers = no
   disable spoolss = yes

   {{ $log_level := dict "trace" "5" "debug" "4" "info" "3" "notice" "2" "warning" "1" "error" "1"  "fatal" "1" -}}
   log level = {{ .log_level | default "warning" | get $log_level }}

   bind interfaces only = {{ .bind_all_interfaces | default false | ternary "no" "yes" }}
   interfaces = 127.0.0.1 {{ .interfaces | join " " }} {{ .docker_interface | default " "}}
   hosts allow = 127.0.0.1 {{ .allow_hosts | join " " }} {{ .docker_net | default " " }}

   mangled names = no
   dos charset = CP1253
   unix charset = UTF-8   

{{ define "SHT" }}
{{- $unsupported := list "vfat"	"msdos"	"f2fs"	"fuseblk" "exfat" -}}
{{- $rosupported := list "apfs"}}
{{- $name := regexReplaceAll "[^A-Za-z0-9_/ ]" .share "_" | regexFind "[A-Za-z0-9_ ]+$" | upper -}}
{{- $dinfo := get .shares $name | default dict -}}
[{{- $name -}}]
   browseable = yes
   writeable = {{ has $dinfo.fs $rosupported | ternary "no" "yes" }}

   # cherry pick from PR#167 to Test
   create mask = 0664
   force create mode = 0664
   directory mask = 0775
   force directory mode = 0775
   # End PR#167

   path = {{- if eq .share "config" }} /homeassistant{{- else }} {{ .data.path }}{{- end }}
   valid users =_ha_mount_user_ {{ .users|default .username|join " " }} {{ .ro_users|join " " }}
   {{ if .ro_users }}
   read list = {{ .ro_users|join " " }}
   {{ end }}
   force user = root
   force group = root
   veto files = /{{ .veto_files | join "/" }}/
   delete veto files = {{ eq (len .veto_files) 0 | ternary "no" "yes" }}

# DEBUG: {{ toJson $dinfo  }}|.share={{ .share }}|$name={{ $name }}|.shares={{ .shares }}|

{{if .recyle_bin_enabled }}
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

# TM:{{ if has $dinfo.fs $unsupported }}unsupported{{else}}{{ .timemachine }}{{ end }} US:{{ .users|default .username|join "," }} {{ .ro_users|join "," }}{{- if .medialibrary.enable }}{{ if .usage }} CL:{{ .usage }}{{ end }} FS:{{ $dinfo.fs | default "native" }} {{ if .recyle_bin_enabled }}RECYCLEBIN{{ end }} {{ end }}
{{- if and .timemachine (has $dinfo.fs $unsupported | not ) }}
   vfs objects = catia fruit streams_xattr{{ if .recyle_bin_enabled }} recycle{{ end }}

   # Time Machine Settings Ref: https://github.com/markthomas93/samba.apple.templates
   fruit:time machine = yes
   #fruit:time machine max size = SIZE [K|M|G|T|P]
   fruit:metadata = stream
{{ end }}
{{- if has $dinfo.fs $unsupported }}
   vfs objects = catia{{ if .recyle_bin_enabled }} recycle{{ end }}{{/*- printf "/*%#v* /" . -*/}}
{{ end }}

{{ end }}

{{- $root := . -}}
{{- range $dd := .shares -}}
               {{- if not $dd.disabled -}}
                  {{- $root2 := deepCopy $root -}}
                  {{- $_ := set $root2 "share" $dd.name -}}
                  {{- $_ := set $root2 "data" $dd -}}
                  {{- template "SHT" $root2 -}}
               {{- end -}}  
        {{/* - end - */}}
{{- end -}}
