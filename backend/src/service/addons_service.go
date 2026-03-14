package service

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/apps"
	"github.com/dianlight/tlog"
	gocache "github.com/patrickmn/go-cache"
	"gitlab.com/tozd/go/errors"
	"go.uber.org/fx"
)

// AddonsServiceInterface defines the contract for addon-related operations.
type AddonsServiceInterface interface {
	// GetStats retrieves the resource usage statistics for the current app.
	// It returns an AppStatsData object on success.
	GetStats() (*apps.AppStatsData, errors.E)
	GetLatestLogs(ctx context.Context) (string, errors.E)
	GetInfo(ctx context.Context) (*apps.AppInfoData, errors.E)
	GetAppConfig(ctx context.Context) (*dto.AppConfigData, errors.E)
	GetAppConfigSchema(ctx context.Context) (*dto.AppConfigSchema, errors.E)
	SetAppConfig(ctx context.Context, options map[string]any) errors.E
}

// AddonsService provides methods to interact with Home Assistant apps.
type AddonsService struct {
	ctx          context.Context
	apictx       *dto.ContextState // Context state for the API, can be used for logging or passing additional information.
	addonsClient apps.ClientWithResponsesInterface
	haWsService  HaWsServiceInterface
	statsCache   *gocache.Cache
	statsMutex   sync.Mutex
}

// AddonsServiceParams holds the dependencies for AddonsService.
type AddonsServiceParams struct {
	fx.In
	Ctx          context.Context
	Apictx       *dto.ContextState
	AddonsClient apps.ClientWithResponsesInterface `optional:"true"`
	HaWsService  HaWsServiceInterface
}

const (
	statsCacheKey     = "addonStats"
	statsCacheExpiry  = 30 * time.Second
	statsCacheCleanup = 1 * time.Minute
)

// NewAddonsService creates a new instance of AddonsService.
func NewAddonsService(lc fx.Lifecycle, params AddonsServiceParams) AddonsServiceInterface {
	if params.AddonsClient == nil {
		tlog.DebugContext(params.Ctx, "AddonsClient is not available for AddonsService. Operations requiring it will fail.")
	}
	p := &AddonsService{
		ctx:          params.Ctx,
		apictx:       params.Apictx,
		addonsClient: params.AddonsClient,
		statsCache:   gocache.New(statsCacheExpiry, statsCacheCleanup),
		haWsService:  params.HaWsService,
	}
	return p
}

// GetStats implements the AddonsServiceInterface.
func (s *AddonsService) GetStats() (*apps.AppStatsData, errors.E) {
	// Try to get from cache first
	if cachedStats, found := s.statsCache.Get(statsCacheKey); found {
		if stats, ok := cachedStats.(*apps.AppStatsData); ok {
			return stats, nil
		}
	}

	// If not in cache, acquire lock to fetch
	s.statsMutex.Lock()
	defer s.statsMutex.Unlock()

	// Re-check cache after acquiring lock
	if cachedStats, found := s.statsCache.Get(statsCacheKey); found {
		if stats, ok := cachedStats.(*apps.AppStatsData); ok {
			return stats, nil
		}
	}

	if s.addonsClient == nil {
		return nil, errors.New("addons client is not initialized")
	}

	resp, err := s.addonsClient.GetSelfAppStatsWithResponse(s.ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get addon stats")
	}

	if resp.StatusCode() != http.StatusOK {
		if resp != nil && resp.Body != nil && strings.Contains(string(resp.Body), "System is not ready with state: shutdown") {
			return nil, errors.WithDetails(dto.ErrorInvalidStateForOperation, string(resp.Body))
		}
		return nil, errors.Errorf("failed to get addon stats: status %d, body: %s", resp.StatusCode(), string(resp.Body))
	}

	if resp.JSON200 == nil {
		return nil, errors.New("addon stats not available or data incomplete")
	}

	stats := &resp.JSON200.Data
	s.statsCache.Set(statsCacheKey, stats, gocache.DefaultExpiration)

	return stats, nil
}

func (s *AddonsService) GetLatestLogs(ctx context.Context) (string, errors.E) {
	if s.addonsClient == nil {
		return "", errors.New("addons client is not initialized")
	}

	resp, err := s.addonsClient.GetAppLogsLatestWithResponse(ctx, "self", &apps.GetAppLogsLatestParams{
		Lines:  new(1000),
		Accept: apps.TextxLog,
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to get addon logs")
	}

	if resp.StatusCode() != http.StatusOK {
		return "", errors.Errorf("failed to get addon logs: status %d, body: %s", resp.StatusCode(), string(resp.Body))
	}

	if resp.Body == nil {
		return "", errors.New("addon logs not available or data incomplete")
	}

	return string(resp.Body), nil
}

func (s *AddonsService) GetInfo(ctx context.Context) (*apps.AppInfoData, errors.E) {
	if s.addonsClient == nil {
		return nil, errors.New("addons client is not initialized")
	}

	resp, err := s.addonsClient.GetAppInfoWithResponse(ctx, "self")
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get addon info: %s", err.Error())
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("failed to get addon info: status %d, body: %s", resp.StatusCode(), string(resp.Body))
	}

	if resp.JSON200 == nil {
		return nil, errors.New("addon info not available or data incomplete")
	}

	return &resp.JSON200.Data, nil
}

func (s *AddonsService) GetAppConfig(ctx context.Context) (*dto.AppConfigData, errors.E) {
	if s.addonsClient == nil {
		return nil, errors.New("addons client is not initialized")
	}

	infoResp, err := s.addonsClient.GetAppInfoWithResponse(ctx, "self")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get addon info for app config")
	}
	if infoResp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("failed to get addon info for app config: status %d, body: %s", infoResp.StatusCode(), string(infoResp.Body))
	}
	if infoResp.JSON200 == nil {
		return nil, errors.New("addon info for app config not available")
	}

	configResp, err := s.addonsClient.GetAppOptionsConfigWithResponse(ctx, "self")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get addon runtime config")
	}
	if configResp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("failed to get addon runtime config: status %d, body: %s", configResp.StatusCode(), string(configResp.Body))
	}
	if configResp.JSON200 == nil {
		return nil, errors.New("addon runtime config not available")
	}

	options := make(map[string]any)
	if infoResp.JSON200.Data.Options != nil {
		options = *infoResp.JSON200.Data.Options
	}

	runtimeConfig := make(map[string]any)
	for key, value := range configResp.JSON200.Data {
		runtimeConfig[key] = value
	}

	return &dto.AppConfigData{
		Options:         options,
		RuntimeConfig:   runtimeConfig,
		RequiresRestart: true,
	}, nil
}

func (s *AddonsService) GetAppConfigSchema(ctx context.Context) (*dto.AppConfigSchema, errors.E) {
	if s.addonsClient == nil {
		return nil, errors.New("addons client is not initialized")
	}

	infoResp, err := s.addonsClient.GetAppInfoWithResponse(ctx, "self")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get addon info for app config schema")
	}
	if infoResp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("failed to get addon info for app config schema: status %d, body: %s", infoResp.StatusCode(), string(infoResp.Body))
	}
	if infoResp.JSON200 == nil {
		return nil, errors.New("addon info for app config schema not available")
	}

	fields := extractAppSchemaFields(infoResp.JSON200.Data.Schema, infoResp.JSON200.Data.Translations)

	description := ""
	if infoResp.JSON200.Data.Description != nil {
		description = *infoResp.JSON200.Data.Description
	}

	longDescription := ""
	if infoResp.JSON200.Data.LongDescription != nil {
		longDescription = *infoResp.JSON200.Data.LongDescription
	}

	return &dto.AppConfigSchema{
		Description:     description,
		LongDescription: longDescription,
		RequiresRestart: true,
		Fields:          fields,
	}, nil
}

func extractAppSchemaFields(rawSchema *[]*map[string]interface{}, translations *map[string]interface{}) []dto.AppConfigSchemaField {
	if rawSchema == nil {
		return []dto.AppConfigSchemaField{}
	}

	optionNames := make([]string, 0)
	for _, schemaItem := range *rawSchema {
		if schemaItem == nil {
			continue
		}

		item := *schemaItem
		if len(item) == 0 {
			continue
		}

		if rawName, hasName := item["name"]; hasName {
			if name, ok := rawName.(string); ok && strings.TrimSpace(name) != "" {
				optionNames = append(optionNames, name)
				continue
			}
		}

		for key := range item {
			optionNames = append(optionNames, key)
		}
	}

	optionDescriptions := extractAppOptionDescriptions(translations, optionNames)

	fields := make([]dto.AppConfigSchemaField, 0)
	for _, schemaItem := range *rawSchema {
		if schemaItem == nil {
			continue
		}

		item := *schemaItem
		if len(item) == 0 {
			continue
		}

		if rawName, hasName := item["name"]; hasName {
			if name, ok := rawName.(string); ok && strings.TrimSpace(name) != "" {
				fields = append(fields, extractSchemaFieldFromNamedItem(name, item, optionDescriptions[name]))
				continue
			}
		}

		for key, value := range item {
			fields = append(fields, extractSchemaFieldFromValue(key, value, optionDescriptions[key]))
		}
	}

	sort.Slice(fields, func(i, j int) bool {
		return fields[i].Name < fields[j].Name
	})

	return fields
}

func extractSchemaFieldFromNamedItem(name string, item map[string]interface{}, translatedDescription string) dto.AppConfigSchemaField {
	normalized := make(map[string]any)
	for key, value := range item {
		if key == "name" {
			continue
		}
		normalized[key] = value
	}

	field := extractSchemaFieldFromValue(name, normalized, translatedDescription)
	if constraint, ok := normalized["type"].(string); ok && field.Constraint == "" {
		field.Constraint = strings.TrimSpace(constraint)
	}

	return field
}

func extractSchemaFieldFromValue(name string, value any, translatedDescription string) dto.AppConfigSchemaField {
	field := dto.AppConfigSchemaField{
		Name:        name,
		Description: strings.TrimSpace(translatedDescription),
	}

	switch rawValue := value.(type) {
	case string:
		field.Constraint = strings.TrimSpace(rawValue)
		if field.Constraint == "" {
			field.Constraint = "str"
		}
		return field
	case map[string]any:
		if rawType, ok := rawValue["type"].(string); ok && strings.TrimSpace(rawType) != "" {
			field.Constraint = strings.TrimSpace(rawType)
		} else if rawSchema, ok := rawValue["schema"].(string); ok && strings.TrimSpace(rawSchema) != "" {
			field.Constraint = strings.TrimSpace(rawSchema)
		} else if rawConstraint, ok := rawValue["constraint"].(string); ok && strings.TrimSpace(rawConstraint) != "" {
			field.Constraint = strings.TrimSpace(rawConstraint)
		}

		if field.Constraint == "" {
			field.Constraint = "str"
		}

		if field.Description == "" {
			field.Description = extractLocalizedString(rawValue["description"])
		}

		field.Optional = extractOptionalFlag(rawValue)
		field.Options = extractSchemaOptions(rawValue)
		return field
	default:
		field.Constraint = "str"
		return field
	}
}

func extractAppOptionDescriptions(translations *map[string]interface{}, optionNames []string) map[string]string {
	if translations == nil || len(*translations) == 0 || len(optionNames) == 0 {
		return map[string]string{}
	}

	descriptions := make(map[string]string)
	languageMaps := extractTranslationLanguageMaps(*translations)
	for _, key := range optionNames {
		if description, found := findOptionDescriptionInLanguageConfigurations(languageMaps, key); found {
			descriptions[key] = description
			continue
		}

		if description, found := findOptionDescriptionInTranslations(*translations, key); found {
			descriptions[key] = description
		}
	}

	return descriptions
}

func extractOptionalFlag(definition map[string]any) bool {
	if optional, ok := definition["optional"].(bool); ok {
		return optional
	}

	if required, ok := definition["required"].(bool); ok {
		return !required
	}

	return false
}

func extractSchemaOptions(definition map[string]any) []string {
	keys := []string{"options", "enum", "values", "valid_values"}
	for _, key := range keys {
		if values, ok := definition[key]; ok {
			if options := toStringSlice(values); len(options) > 0 {
				return options
			}
		}
	}

	return nil
}

func toStringSlice(values any) []string {
	list, ok := values.([]any)
	if !ok || len(list) == 0 {
		return nil
	}

	result := make([]string, 0, len(list))
	for _, value := range list {
		stringValue := strings.TrimSpace(toString(value))
		if stringValue == "" {
			continue
		}
		result = append(result, stringValue)
	}

	if len(result) == 0 {
		return nil
	}

	return result
}

func toString(value any) string {
	switch converted := value.(type) {
	case string:
		return converted
	case float64:
		return strings.TrimSpace(strings.TrimRight(strings.TrimRight(fmtFloat(converted), "0"), "."))
	case bool:
		if converted {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}

func fmtFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func extractTranslationLanguageMaps(translations map[string]any) []map[string]any {
	if len(translations) == 0 {
		return nil
	}

	languagePriority := []string{"en", "en_US"}
	languageMaps := make([]map[string]any, 0, len(translations))
	seen := make(map[string]struct{})

	for _, language := range languagePriority {
		node, ok := translations[language]
		if !ok {
			continue
		}
		languageMap, ok := node.(map[string]any)
		if !ok {
			continue
		}
		languageMaps = append(languageMaps, languageMap)
		seen[language] = struct{}{}
	}

	for language, node := range translations {
		if _, ok := seen[language]; ok {
			continue
		}
		languageMap, ok := node.(map[string]any)
		if !ok {
			continue
		}
		languageMaps = append(languageMaps, languageMap)
	}

	return languageMaps
}

func findOptionDescriptionInLanguageConfigurations(languageMaps []map[string]any, optionName string) (string, bool) {
	for _, languageMap := range languageMaps {
		configuration, ok := languageMap["configuration"].(map[string]any)
		if !ok {
			continue
		}

		if direct, ok := configuration[optionName]; ok {
			if description := extractLocalizedString(direct); description != "" {
				return description, true
			}
		}

		if direct, ok := configuration[optionName+"_description"]; ok {
			if description := extractLocalizedString(direct); description != "" {
				return description, true
			}
		}
	}

	return "", false
}

func findOptionDescriptionInTranslations(node any, optionName string) (string, bool) {
	switch value := node.(type) {
	case map[string]any:
		if direct, ok := value[optionName]; ok {
			if description := extractLocalizedString(direct); description != "" {
				return description, true
			}
		}

		if direct, ok := value[optionName+"_description"]; ok {
			if description := extractLocalizedString(direct); description != "" {
				return description, true
			}
		}

		for _, nested := range value {
			if description, found := findOptionDescriptionInTranslations(nested, optionName); found {
				return description, true
			}
		}
	case []any:
		for _, nested := range value {
			if description, found := findOptionDescriptionInTranslations(nested, optionName); found {
				return description, true
			}
		}
	}

	return "", false
}

func extractLocalizedString(value any) string {
	switch data := value.(type) {
	case string:
		return strings.TrimSpace(data)
	case map[string]any:
		for _, key := range []string{"description", "label", "name", "title", "en", "en_US"} {
			if localized, ok := data[key]; ok {
				if localizedString, ok := localized.(string); ok {
					localizedString = strings.TrimSpace(localizedString)
					if localizedString != "" {
						return localizedString
					}
					continue
				}

				if nested := extractLocalizedString(localized); nested != "" {
					return nested
				}
			}
		}
	}

	return ""
}

func (s *AddonsService) SetAppConfig(ctx context.Context, options map[string]any) errors.E {
	if s.addonsClient == nil {
		return errors.New("addons client is not initialized")
	}

	request := apps.SetAppOptionsJSONRequestBody{
		Options: &options,
	}

	resp, err := s.addonsClient.SetAppOptionsWithResponse(ctx, "self", request)
	if err != nil {
		return errors.Wrap(err, "failed to set addon options")
	}

	if resp.StatusCode() != http.StatusOK {
		return errors.Errorf("failed to set addon options: status %d, body: %s", resp.StatusCode(), string(resp.Body))
	}

	return nil
}
