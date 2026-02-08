package dto

//go:generate go tool goenums issue_problem_type.go
type problemType int // Description string,IssueTitle string,RepositoryUrl string,Template string

const (
	problemTypeFrontendUI         problemType = iota // "frontend_ui" "Frontend UI","[UI]","https://github.com/dianlight/srat","bug_report.yaml"
	problemTypeHAIntegration                         // "ha_integration" "HA Integration","[HA Integration]","https://github.com/dianlight/srat","bug_report.yaml"
	problemTypeAddonCrash                            // "addon_crash" "Addon Crash","[SambaNas2]","https://github.com/dianlight/hassio-addons","BUG-REPORT.yml"
	problemTypeAddonUpdate                           // "addon_update" "Addon Update","[SambaNas2]","https://github.com/dianlight/hassio-addons","BUG-REPORT.yml"
	problemTypeAddonStartup                          // "addon_startup" "Addon Startup","[SambaNas2]","https://github.com/dianlight/hassio-addons","BUG-REPORT.yml"
	problemTypeAddonFunctionality                    // "addon_functionality" "Addon Functionality","[SambaNas2]","https://github.com/dianlight/hassio-addons","BUG-REPORT.yml"
	problemTypeInstallation                          // "installation" "Installation","[Installation]","https://github.com/dianlight/srat","bug_report.yaml"
	problemTypePerformance                           // "performance" "Performance","[Performance]","https://github.com/dianlight/srat","bug_report.yaml"
	problemTypeSecurity                              // "security" "Security","[Security]","https://github.com/dianlight/srat","bug_report.yaml"
	problemTypeIntegration                           // "integration" "Integration","[Integration]","https://github.com/dianlight/srat","bug_report.yaml"
	problemTypeDocumentation                         // "documentation" "Documentation","[Documentation]","https://github.com/dianlight/srat","bug_report.yaml"
	problemTypeSamba                                 // "samba" "Samba","[Samba]","https://github.com/dianlight/srat","bug_report.yaml"
	problemTypeOther                                 // "other" "Other","[Other]","https://github.com/dianlight/srat","bug_report.yaml"
)
