package dto

//go:generate go tool goenums issue_problem_type.go
type problemType int // Description string,IssueTitle string,RepositoryUrl string,Template string

const (
	problemTypeFrontendUI    problemType = iota // "frontend_ui" "Frontend UI","[UI]","https://github.com/dianlight/srat","bug_report.yaml"
	problemTypeHAIntegration                    // "ha_integration" "HA Integration","[HA Integration]","https://github.com/dianlight/srat","bug_report.yaml"
	problemTypeAddon                            // "addon" "Addon","[SambaNas2]","https://github.com/dianlight/hassio-addons","BUG-REPORT.yml"
	problemTypeSamba                            // "samba" "Samba","[Samba]","https://github.com/dianlight/srat","bug_report.yaml"
)
