package client

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

// ProtectionInventoryEntry describes access without retaining bearer credentials.
type ProtectionInventoryEntry struct {
	ApprovalKey      string
	Scope            string
	TargetType       string
	TargetID         string
	Alias            *string
	Subject          *string
	Status           string
	CreatedAt        *int64
	CreatedBy        *string
	LastUpdatedAt    *int64
	LastUpdatedBy    *string
	ExpiresAt        *int64
	IsEnvVar         *bool
	Note             *string
	IntegrationID    *string
	ConfigurationID  *string
	PathPrefix       *string
	TrustedProjectID *string
	CustomAllow      []TrustedSourcesAccessRule
	Issuer           *string
	Claims           TrustedSourcesClaims
	To               *TrustedSourcesEnvMatcher
	Implicit         bool
}

// ProjectProtectionInventory contains the access entries observed through public APIs.
type ProjectProtectionInventory struct {
	ProjectID         string
	TeamID            string
	Entries           []ProtectionInventoryEntry
	IncompleteReasons []string
}

type inventoryTrustedSources struct {
	Projects                     map[string]TrustedSourcesProject        `json:"projects"`
	OIDCProviders                map[string][]TrustedSourcesOIDCProvider `json:"oidcProviders"`
	EnableVercelCiSameRepository *bool                                   `json:"enableVercelCiSameRepository"`
}

type inventoryProject struct {
	ID               string                     `json:"id"`
	AccountID        string                     `json:"accountId"`
	ProtectionBypass map[string]json.RawMessage `json:"protectionBypass"`
	OptionsAllowlist json.RawMessage            `json:"optionsAllowlist"`
	TrustedSources   json.RawMessage            `json:"trustedSources"`
}

type inventoryAlias struct {
	UID              string                     `json:"uid"`
	Alias            string                     `json:"alias"`
	ProjectID        string                     `json:"projectId"`
	ProtectionBypass map[string]json.RawMessage `json:"protectionBypass"`
}

// GetProjectProtectionInventory reads project settings and all pages of project aliases.
// Deployment-owned access without an alias record is not exposed by these APIs.
func (c *Client) GetProjectProtectionInventory(ctx context.Context, projectID, teamID string) (ProjectProtectionInventory, error) {
	teamID = c.TeamID(teamID)
	query := url.Values{}
	if teamID != "" {
		query.Set("teamId", teamID)
	}
	var project inventoryProject
	err := c.doRequest(clientRequest{
		ctx: ctx, method: "GET", sensitiveResponse: true,
		url: urlWithQuery(c.baseURL+"/v9/projects/"+url.PathEscape(projectID), query),
	}, &project)
	if err != nil {
		return ProjectProtectionInventory{}, fmt.Errorf("unable to read project protection inventory: %w", err)
	}
	if project.ID != projectID || (teamID != "" && project.AccountID != teamID) {
		return ProjectProtectionInventory{}, fmt.Errorf("project protection inventory ownership did not match the requested project and team")
	}
	if strings.HasPrefix(project.AccountID, "team_") {
		teamID = project.AccountID
		query.Set("teamId", teamID)
	}
	result := ProjectProtectionInventory{
		ProjectID: project.ID, TeamID: teamID,
		IncompleteReasons: []string{
			"Public APIs do not establish coverage of deployment-owned access without alias records.",
			"Public APIs do not guarantee that all access entries are visible to the API token.",
		},
	}
	now := time.Now().Unix()
	if err := result.addBypasses("project", project.ID, nil, project.ProtectionBypass, now); err != nil {
		return ProjectProtectionInventory{}, err
	}
	if err := result.addProjectSettings(project); err != nil {
		return ProjectProtectionInventory{}, err
	}
	query.Set("projectId", project.ID)
	var until *int64
	seenCursors := map[int64]bool{}
	seenAliases := map[string]bool{}
	for {
		var page struct {
			Aliases    []inventoryAlias `json:"aliases"`
			Pagination *PageInfo        `json:"pagination"`
		}
		err := c.doRequest(clientRequest{
			ctx: ctx, method: "GET", sensitiveResponse: true,
			url: urlWithQuery(c.baseURL+"/v4/aliases", paginationQuery(query, 100, until, nil)),
		}, &page)
		if err != nil {
			return ProjectProtectionInventory{}, fmt.Errorf("unable to list project protection aliases: %w", err)
		}
		if page.Pagination == nil || page.Aliases == nil {
			return ProjectProtectionInventory{}, fmt.Errorf("alias inventory response omitted aliases or pagination")
		}
		for _, alias := range page.Aliases {
			if alias.ProjectID != project.ID || alias.UID == "" || alias.Alias == "" {
				return ProjectProtectionInventory{}, fmt.Errorf("alias inventory returned an alias without the requested project ownership or identity")
			}
			if seenAliases[alias.UID] {
				return ProjectProtectionInventory{}, fmt.Errorf("alias inventory changed during pagination; retry the read")
			}
			seenAliases[alias.UID] = true
			if err := result.addBypasses("alias", alias.UID, &alias.Alias, alias.ProtectionBypass, now); err != nil {
				return ProjectProtectionInventory{}, err
			}
		}
		if page.Pagination.Next == nil {
			break
		}
		if seenCursors[*page.Pagination.Next] {
			return ProjectProtectionInventory{}, fmt.Errorf("alias inventory pagination cursor did not advance")
		}
		seenCursors[*page.Pagination.Next] = true
		until = page.Pagination.Next
	}
	sort.Slice(result.Entries, func(i, j int) bool { return result.Entries[i].ApprovalKey < result.Entries[j].ApprovalKey })
	sort.Strings(result.IncompleteReasons)
	return result, nil
}

func (r *ProjectProtectionInventory) addBypasses(targetType, targetID string, alias *string, bypasses map[string]json.RawMessage, now int64) error {
	if bypasses == nil {
		r.IncompleteReasons = append(r.IncompleteReasons, fmt.Sprintf("The %s %s response omitted a readable protectionBypass map.", targetType, targetID))
	}
	for key, raw := range bypasses {
		var bypass struct {
			Scope           string  `json:"scope"`
			Access          string  `json:"access"`
			CreatedAt       *int64  `json:"createdAt"`
			CreatedBy       *string `json:"createdBy"`
			LastUpdatedAt   *int64  `json:"lastUpdatedAt"`
			LastUpdatedBy   *string `json:"lastUpdatedBy"`
			Expires         *int64  `json:"expires"`
			IsEnvVar        *bool   `json:"isEnvVar"`
			Note            *string `json:"note"`
			IntegrationID   *string `json:"integrationId"`
			ConfigurationID *string `json:"configurationId"`
		}
		if err := json.Unmarshal(raw, &bypass); err != nil || bypass.Scope == "" {
			return fmt.Errorf("unable to decode protection bypass metadata; response details omitted to protect secrets")
		}
		entry := ProtectionInventoryEntry{
			Scope: bypass.Scope, TargetType: targetType, TargetID: targetID, Alias: alias,
			CreatedAt: bypass.CreatedAt, CreatedBy: bypass.CreatedBy, LastUpdatedAt: bypass.LastUpdatedAt, LastUpdatedBy: bypass.LastUpdatedBy,
			ExpiresAt: bypass.Expires, IsEnvVar: bypass.IsEnvVar, Note: bypass.Note,
			IntegrationID: bypass.IntegrationID, ConfigurationID: bypass.ConfigurationID, Status: "active",
		}
		switch bypass.Scope {
		case "automation-bypass":
			if entry.IsEnvVar == nil {
				value := true // Legacy automation bypasses predate the explicit default selection.
				entry.IsEnvVar = &value
			}
		case "integration-automation-bypass", "alias-protection-override", "shareable-link":
		case "user":
			entry.Subject = &key
			entry.Status = bypass.Access
			if bypass.Access != "granted" && bypass.Access != "requested" {
				entry.Status = "unknown"
				r.IncompleteReasons = append(r.IncompleteReasons, "A user bypass has an unrecognized access status.")
			}
		case "email_invite":
			entry.Subject = &key
			entry.Status = "invited"
		default:
			entry.Status = "unknown"
			r.IncompleteReasons = append(r.IncompleteReasons, "An unrecognized protection bypass scope was returned.")
		}
		if bypass.Expires != nil && *bypass.Expires <= now {
			entry.Status = "expired"
		}
		// Hash the API map key before it can reach state. Keys for bearer bypasses are credentials.
		var access map[string]any
		_ = json.Unmarshal(raw, &access)
		for _, field := range []string{"createdAt", "createdBy", "lastUpdatedAt", "lastUpdatedBy", "note"} {
			delete(access, field)
		}
		if bypass.Scope == "automation-bypass" {
			access["isEnvVar"] = *entry.IsEnvVar
		}
		entry.ApprovalKey = inventoryApprovalKey(r.TeamID, r.ProjectID, targetType, targetID, alias, key, access)
		r.Entries = append(r.Entries, entry)
	}
	return nil
}

func (r *ProjectProtectionInventory) addProjectSettings(project inventoryProject) error {
	if len(project.OptionsAllowlist) == 0 {
		r.IncompleteReasons = append(r.IncompleteReasons, "The project response omitted optionsAllowlist.")
	} else {
		var options *OptionsAllowlist
		if err := decodeInventorySettings(project.OptionsAllowlist, &options); err != nil {
			return fmt.Errorf("unable to decode OPTIONS allowlist metadata")
		}
		if options != nil {
			for _, path := range options.Paths {
				entry := ProtectionInventoryEntry{Scope: "options-allowlist", TargetType: "project", TargetID: r.ProjectID, Status: "active", PathPrefix: &path.Value}
				entry.ApprovalKey = inventoryApprovalKey(r.TeamID, r.ProjectID, entry.Scope, path.Value)
				r.Entries = append(r.Entries, entry)
			}
		}
	}
	if len(project.TrustedSources) == 0 {
		r.IncompleteReasons = append(r.IncompleteReasons, "The project response omitted trustedSources; implicit access could not be determined.")
		return nil
	}
	var trusted inventoryTrustedSources
	if err := decodeInventorySettings(project.TrustedSources, &trusted); err != nil {
		return fmt.Errorf("unable to decode Trusted Sources metadata")
	}
	if trusted.Projects == nil {
		trusted.Projects = map[string]TrustedSourcesProject{}
	}
	if _, ok := trusted.Projects[r.ProjectID]; !ok {
		trusted.Projects[r.ProjectID] = TrustedSourcesProject{}
	}
	for id, source := range trusted.Projects {
		entry := ProtectionInventoryEntry{
			Scope: "trusted-project", TargetType: "project", TargetID: r.ProjectID, Status: "active",
			TrustedProjectID: &id, CustomAllow: source.CustomAllow, Note: source.Label,
			Implicit: id == r.ProjectID && source.CustomAllow == nil,
		}
		if source.CustomAllow != nil && len(source.CustomAllow) == 0 {
			entry.Status = "disabled"
		}
		entry.ApprovalKey = inventoryApprovalKey(r.TeamID, r.ProjectID, entry.Scope, id, source.CustomAllow)
		r.Entries = append(r.Entries, entry)
	}
	for issuer, sources := range trusted.OIDCProviders {
		for _, source := range sources {
			entry := ProtectionInventoryEntry{
				Scope: "trusted-oidc", TargetType: "project", TargetID: r.ProjectID, Status: "active",
				Issuer: &issuer, Claims: source.Claims, To: &source.To, Note: source.Label,
			}
			entry.ApprovalKey = inventoryApprovalKey(r.TeamID, r.ProjectID, entry.Scope, issuer, source.Claims, source.To)
			r.Entries = append(r.Entries, entry)
		}
	}
	if trusted.EnableVercelCiSameRepository == nil || *trusted.EnableVercelCiSameRepository {
		entry := ProtectionInventoryEntry{
			Scope: "vercel-ci-same-repository", TargetType: "project", TargetID: r.ProjectID, Status: "active",
			Implicit: trusted.EnableVercelCiSameRepository == nil,
		}
		entry.ApprovalKey = inventoryApprovalKey(r.TeamID, r.ProjectID, entry.Scope)
		r.Entries = append(r.Entries, entry)
	}
	return nil
}

func decodeInventorySettings(raw json.RawMessage, value any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	// New access-bearing settings must not be silently dropped from the inventory.
	decoder.DisallowUnknownFields()
	return decoder.Decode(value)
}

func inventoryApprovalKey(values ...any) string {
	// Sort rule arrays as sets, but retain null versus [] (default rules versus denial).
	var normalized any
	_ = json.Unmarshal(mustMarshal(values), &normalized)
	var normalize func(any)
	normalize = func(value any) {
		switch value := value.(type) {
		case map[string]any:
			for _, child := range value {
				normalize(child)
			}
		case []any:
			for _, child := range value {
				normalize(child)
			}
			sort.Slice(value, func(i, j int) bool { return string(mustMarshal(value[i])) < string(mustMarshal(value[j])) })
		}
	}
	// The outer tuple has positional meaning; only its children are sets/maps.
	for _, value := range normalized.([]any) {
		normalize(value)
	}
	digest := sha256.Sum256(mustMarshal(normalized))
	return "v1:" + hex.EncodeToString(digest[:])
}
