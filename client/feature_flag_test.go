package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestCreateFeatureFlag(t *testing.T) {
	t.Parallel()

	request := CreateFeatureFlagRequest{
		ProjectID:   "prj_123",
		TeamID:      "team_123",
		Slug:        "checkout-banner",
		Kind:        "string",
		Description: "Controls the checkout banner copy",
		State:       "active",
		Variants: []FeatureFlagVariant{
			{ID: "control", Value: "control"},
			{ID: "variant-a", Label: "Variant A", Value: "variant-a"},
		},
		Environments: map[string]FeatureFlagEnvironment{
			"production": {
				Active: true,
				Rules:  []json.RawMessage{},
				Fallthrough: FeatureFlagOutcome{
					Type:      "variant",
					VariantID: "variant-a",
				},
				PausedOutcome: FeatureFlagOutcome{
					Type:      "variant",
					VariantID: "control",
				},
			},
		},
	}

	client := newFeatureFlagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "PUT", "/v1/projects/prj_123/feature-flags/flags", "team_123", request)
		_, _ = w.Write([]byte(`{
			"id":"flag_123",
			"slug":"checkout-banner",
			"kind":"string",
			"description":"Controls the checkout banner copy",
			"state":"active",
			"projectId":"prj_123",
			"ownerId":"team_123",
			"typeName":"flag",
			"createdBy":"user_123",
			"seed":42,
			"revision":3,
			"variants":[
				{"id":"control","value":"control"},
				{"id":"variant-a","label":"Variant A","value":"variant-a"}
			],
			"environments":{
				"production":{
					"active":true,
					"rules":[],
					"fallthrough":{"type":"variant","variantId":"variant-a"},
					"pausedOutcome":{"type":"variant","variantId":"control"}
				}
			}
		}`))
	})

	flag, err := client.CreateFeatureFlag(context.Background(), request)
	if err != nil {
		t.Fatalf("CreateFeatureFlag returned error: %v", err)
	}

	if flag.ID != "flag_123" {
		t.Fatalf("expected ID flag_123, got %q", flag.ID)
	}
	if flag.Revision != 3 {
		t.Fatalf("expected revision 3, got %d", flag.Revision)
	}
	if flag.Environments["production"].Fallthrough.VariantID != "variant-a" {
		t.Fatalf("expected production fallthrough variant-a, got %q", flag.Environments["production"].Fallthrough.VariantID)
	}
}

func TestCreateFeatureFlagOmitsEnvironmentsWhenUnset(t *testing.T) {
	t.Parallel()

	request := CreateFeatureFlagRequest{
		ProjectID:   "prj_123",
		TeamID:      "team_123",
		Slug:        "checkout-banner",
		Kind:        "string",
		Description: "Controls the checkout banner copy",
		State:       "active",
		Variants: []FeatureFlagVariant{
			{ID: "control", Value: "control"},
			{ID: "variant-a", Value: "variant-a"},
		},
	}

	client := newFeatureFlagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "PUT", "/v1/projects/prj_123/feature-flags/flags", "team_123", map[string]any{
			"slug":        "checkout-banner",
			"kind":        "string",
			"description": "Controls the checkout banner copy",
			"state":       "active",
			"variants": []map[string]any{
				{"id": "control", "value": "control"},
				{"id": "variant-a", "value": "variant-a"},
			},
		})
		_, _ = w.Write([]byte(`{
			"id":"flag_123",
			"slug":"checkout-banner",
			"kind":"string",
			"description":"Controls the checkout banner copy",
			"state":"active",
			"projectId":"prj_123",
			"ownerId":"team_123",
			"typeName":"flag",
			"createdBy":"user_123",
			"seed":42,
			"revision":1,
			"variants":[
				{"id":"control","value":"control"},
				{"id":"variant-a","value":"variant-a"}
			],
			"environments":{}
		}`))
	})

	_, err := client.CreateFeatureFlag(context.Background(), request)
	if err != nil {
		t.Fatalf("CreateFeatureFlag returned error: %v", err)
	}
}

func TestGetFeatureFlagUsesConfiguredTeam(t *testing.T) {
	t.Parallel()

	client := newFeatureFlagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/v1/projects/prj_123/feature-flags/flags/flag_123", "team_default", nil)
		_, _ = w.Write([]byte(`{
			"id":"flag_123",
			"slug":"new-homepage",
			"kind":"boolean",
			"description":"Homepage experiment",
			"state":"active",
			"projectId":"prj_123",
			"ownerId":"team_default",
			"typeName":"flag",
			"createdBy":"user_123",
			"seed":7,
			"revision":1,
			"variants":[
				{"id":"off","value":false},
				{"id":"on","value":true}
			],
			"environments":{
				"preview":{
					"active":false,
					"rules":[],
					"fallthrough":{"type":"variant","variantId":"off"},
					"pausedOutcome":{"type":"variant","variantId":"off"}
				}
			}
		}`))
	}).WithTeam(Team{ID: "team_default"})

	flag, err := client.GetFeatureFlag(context.Background(), GetFeatureFlagRequest{
		ProjectID:    "prj_123",
		FlagIDOrSlug: "flag_123",
	})
	if err != nil {
		t.Fatalf("GetFeatureFlag returned error: %v", err)
	}

	value, ok := flag.Variants[1].Value.(bool)
	if !ok {
		t.Fatalf("expected boolean variant value, got %T", flag.Variants[1].Value)
	}
	if !value {
		t.Fatalf("expected variant value true")
	}
}

func TestCreateFeatureFlagSupportsLegacyKeyField(t *testing.T) {
	t.Parallel()

	request := CreateFeatureFlagRequest{
		ProjectID: "prj_123",
		TeamID:    "team_123",
		Key:       "legacy-key",
		Kind:      "boolean",
		Variants: []FeatureFlagVariant{
			{ID: "off", Value: false},
			{ID: "on", Value: true},
		},
		Environments: map[string]FeatureFlagEnvironment{
			"production": {
				Active: true,
				Rules:  []json.RawMessage{},
				Fallthrough: FeatureFlagOutcome{
					Type:      "variant",
					VariantID: "on",
				},
				PausedOutcome: FeatureFlagOutcome{
					Type:      "variant",
					VariantID: "off",
				},
			},
		},
	}

	client := newFeatureFlagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "PUT", "/v1/projects/prj_123/feature-flags/flags", "team_123", map[string]any{
			"slug":        "legacy-key",
			"kind":        "boolean",
			"description": "",
			"variants": []map[string]any{
				{"id": "off", "value": false},
				{"id": "on", "value": true},
			},
			"environments": map[string]any{
				"production": map[string]any{
					"active": true,
					"rules":  []any{},
					"fallthrough": map[string]any{
						"type":      "variant",
						"variantId": "on",
					},
					"pausedOutcome": map[string]any{
						"type":      "variant",
						"variantId": "off",
					},
				},
			},
		})
		_, _ = w.Write([]byte(`{
			"id":"flag_123",
			"slug":"legacy-key",
			"kind":"boolean",
			"state":"active",
			"projectId":"prj_123",
			"ownerId":"team_123",
			"typeName":"flag",
			"createdBy":"user_123",
			"seed":1,
			"revision":1,
			"variants":[
				{"id":"off","value":false},
				{"id":"on","value":true}
			],
			"environments":{
				"production":{
					"active":true,
					"rules":[],
					"fallthrough":{"type":"variant","variantId":"on"},
					"pausedOutcome":{"type":"variant","variantId":"off"}
				}
			}
		}`))
	})

	flag, err := client.CreateFeatureFlag(context.Background(), request)
	if err != nil {
		t.Fatalf("CreateFeatureFlag returned error: %v", err)
	}

	if flag.Slug != "legacy-key" {
		t.Fatalf("expected slug legacy-key, got %q", flag.Slug)
	}
}

func TestUpdateFeatureFlag(t *testing.T) {
	t.Parallel()

	request := UpdateFeatureFlagRequest{
		ProjectID:    "prj_123",
		FlagIDOrSlug: "flag_123",
		TeamID:       "team_123",
		Message:      "pause rollout",
		State:        "archived",
		Environments: map[string]FeatureFlagEnvironment{
			"production": {
				Active: false,
				Rules:  []json.RawMessage{},
				Fallthrough: FeatureFlagOutcome{
					Type:      "variant",
					VariantID: "control",
				},
				PausedOutcome: FeatureFlagOutcome{
					Type:      "variant",
					VariantID: "control",
				},
			},
		},
	}

	client := newFeatureFlagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "PATCH", "/v1/projects/prj_123/feature-flags/flags/flag_123", "team_123", request)
		_, _ = w.Write([]byte(`{
			"id":"flag_123",
			"slug":"checkout-banner",
			"kind":"string",
			"description":"Controls the checkout banner copy",
			"state":"archived",
			"projectId":"prj_123",
			"ownerId":"team_123",
			"typeName":"flag",
			"createdBy":"user_123",
			"seed":42,
			"revision":4,
			"variants":[{"id":"control","value":"control"}],
			"environments":{
				"production":{
					"active":false,
					"rules":[],
					"fallthrough":{"type":"variant","variantId":"control"},
					"pausedOutcome":{"type":"variant","variantId":"control"}
				}
			}
		}`))
	})

	flag, err := client.UpdateFeatureFlag(context.Background(), request)
	if err != nil {
		t.Fatalf("UpdateFeatureFlag returned error: %v", err)
	}

	if flag.State != "archived" {
		t.Fatalf("expected archived state, got %q", flag.State)
	}
	if flag.Revision != 4 {
		t.Fatalf("expected revision 4, got %d", flag.Revision)
	}
}

func TestDeleteFeatureFlag(t *testing.T) {
	t.Parallel()

	client := newFeatureFlagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "DELETE", "/v1/projects/prj_123/feature-flags/flags/flag_123", "team_123", nil)
		w.WriteHeader(http.StatusNoContent)
	})

	err := client.DeleteFeatureFlag(context.Background(), DeleteFeatureFlagRequest{
		ProjectID:    "prj_123",
		FlagIDOrSlug: "flag_123",
		TeamID:       "team_123",
	})
	if err != nil {
		t.Fatalf("DeleteFeatureFlag returned error: %v", err)
	}
}

func TestFeatureFlagRequestAliasesUseLegacyFlagID(t *testing.T) {
	t.Parallel()

	t.Run("get", func(t *testing.T) {
		t.Parallel()

		client := newFeatureFlagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			assertRequest(t, r, "GET", "/v1/projects/prj_123/feature-flags/flags/flag_legacy", "team_123", nil)
			_, _ = w.Write([]byte(`{
				"id":"flag_legacy",
				"slug":"legacy-key",
				"kind":"boolean",
				"state":"active",
				"projectId":"prj_123",
				"ownerId":"team_123",
				"typeName":"flag",
				"createdBy":"user_123",
				"seed":1,
				"revision":1,
				"variants":[{"id":"on","value":true}],
				"environments":{"production":{"active":true,"rules":[],"fallthrough":{"type":"variant","variantId":"on"},"pausedOutcome":{"type":"variant","variantId":"on"}}}
			}`))
		})

		_, err := client.GetFeatureFlag(context.Background(), GetFeatureFlagRequest{
			ProjectID: "prj_123",
			TeamID:    "team_123",
			FlagID:    "flag_legacy",
		})
		if err != nil {
			t.Fatalf("GetFeatureFlag returned error: %v", err)
		}
	})

	t.Run("update", func(t *testing.T) {
		t.Parallel()

		client := newFeatureFlagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			assertRequest(t, r, "PATCH", "/v1/projects/prj_123/feature-flags/flags/flag_legacy", "team_123", map[string]any{
				"slug":  "legacy-key",
				"state": "active",
			})
			_, _ = w.Write([]byte(`{
				"id":"flag_legacy",
				"slug":"legacy-key",
				"kind":"boolean",
				"state":"active",
				"projectId":"prj_123",
				"ownerId":"team_123",
				"typeName":"flag",
				"createdBy":"user_123",
				"seed":1,
				"revision":2,
				"variants":[{"id":"on","value":true}],
				"environments":{"production":{"active":true,"rules":[],"fallthrough":{"type":"variant","variantId":"on"},"pausedOutcome":{"type":"variant","variantId":"on"}}}
			}`))
		})

		_, err := client.UpdateFeatureFlag(context.Background(), UpdateFeatureFlagRequest{
			ProjectID: "prj_123",
			TeamID:    "team_123",
			FlagID:    "flag_legacy",
			Key:       "legacy-key",
			State:     "active",
		})
		if err != nil {
			t.Fatalf("UpdateFeatureFlag returned error: %v", err)
		}
	})

	t.Run("delete", func(t *testing.T) {
		t.Parallel()

		client := newFeatureFlagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			assertRequest(t, r, "DELETE", "/v1/projects/prj_123/feature-flags/flags/flag_legacy", "team_123", nil)
			w.WriteHeader(http.StatusNoContent)
		})

		err := client.DeleteFeatureFlag(context.Background(), DeleteFeatureFlagRequest{
			ProjectID: "prj_123",
			TeamID:    "team_123",
			FlagID:    "flag_legacy",
		})
		if err != nil {
			t.Fatalf("DeleteFeatureFlag returned error: %v", err)
		}
	})
}

func TestCreateFeatureFlagSegment(t *testing.T) {
	t.Parallel()

	request := CreateFeatureFlagSegmentRequest{
		ProjectID:   "prj_123",
		TeamID:      "team_123",
		Slug:        "internal-users",
		Label:       "Internal Users",
		Description: "Matches employees",
		Hint:        "user-email",
		Data: FeatureFlagSegmentData{
			Include: map[string]map[string][]FeatureFlagSegmentValue{
				"user": {
					"email": {
						{Value: "alice@example.com"},
					},
				},
			},
			Rules: []FeatureFlagSegmentRule{
				{
					ID: "rule-1",
					Conditions: []FeatureFlagSegmentCondition{
						{
							LHS: FeatureFlagSegmentConditionLHS{
								Type:      "entity",
								Kind:      "user",
								Attribute: "email",
							},
							CMP: "endsWith",
							RHS: "@example.com",
						},
					},
					Outcome: FeatureFlagSegmentOutcome{Type: "all"},
				},
			},
		},
	}

	client := newFeatureFlagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "PUT", "/v1/projects/prj_123/feature-flags/segments", "team_123", request)
		_, _ = w.Write([]byte(`{
			"id":"segment_123",
			"slug":"internal-users",
			"label":"Internal Users",
			"description":"Matches employees",
			"projectId":"prj_123",
			"typeName":"segment",
			"hint":"user-email",
			"createdAt":1,
			"updatedAt":2,
			"data":{
				"include":{"user":{"email":[{"value":"alice@example.com"}]}},
				"exclude":{},
				"rules":[
					{
						"id":"rule-1",
						"conditions":[
							{
								"lhs":{"type":"entity","kind":"user","attribute":"email"},
								"cmp":"endsWith",
								"rhs":"@example.com"
							}
						],
						"outcome":{"type":"all"}
					}
				]
			}
		}`))
	})

	segment, err := client.CreateFeatureFlagSegment(context.Background(), request)
	if err != nil {
		t.Fatalf("CreateFeatureFlagSegment returned error: %v", err)
	}

	if segment.ID != "segment_123" {
		t.Fatalf("expected ID segment_123, got %q", segment.ID)
	}
	if segment.Data.Include["user"]["email"][0].Value != "alice@example.com" {
		t.Fatalf("expected included email alice@example.com, got %q", segment.Data.Include["user"]["email"][0].Value)
	}
}

func TestGetFeatureFlagSegment(t *testing.T) {
	t.Parallel()

	client := newFeatureFlagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/v1/projects/prj_123/feature-flags/segments/segment_123", "team_123", nil)
		_, _ = w.Write([]byte(`{
			"id":"segment_123",
			"slug":"beta-users",
			"label":"Beta Users",
			"projectId":"prj_123",
			"typeName":"segment",
			"hint":"user-id",
			"createdAt":1,
			"updatedAt":2,
			"data":{
				"include":{},
				"exclude":{"user":{"id":[{"value":"user_123","note":"manual"}]}},
				"rules":[]
			}
		}`))
	})

	segment, err := client.GetFeatureFlagSegment(context.Background(), GetFeatureFlagSegmentRequest{
		ProjectID:       "prj_123",
		SegmentIDOrSlug: "segment_123",
		TeamID:          "team_123",
	})
	if err != nil {
		t.Fatalf("GetFeatureFlagSegment returned error: %v", err)
	}

	if segment.Data.Exclude["user"]["id"][0].Note != "manual" {
		t.Fatalf("expected exclusion note manual, got %q", segment.Data.Exclude["user"]["id"][0].Note)
	}
}

func TestUpdateFeatureFlagSegment(t *testing.T) {
	t.Parallel()

	request := UpdateFeatureFlagSegmentRequest{
		ProjectID:       "prj_123",
		SegmentIDOrSlug: "segment_123",
		TeamID:          "team_123",
		Label:           "Beta Users",
		Hint:            "user-id",
		Data: FeatureFlagSegmentData{
			Rules: []FeatureFlagSegmentRule{
				{
					ID: "rule-1",
					Conditions: []FeatureFlagSegmentCondition{
						{
							LHS: FeatureFlagSegmentConditionLHS{
								Type:      "entity",
								Kind:      "user",
								Attribute: "plan",
							},
							CMP: "eq",
							RHS: "beta",
						},
					},
					Outcome: FeatureFlagSegmentOutcome{
						Type: "split",
						Base: &FeatureFlagSegmentConditionLHS{
							Type:      "entity",
							Kind:      "user",
							Attribute: "id",
						},
						PassPromille: floatPtr(500),
					},
				},
			},
		},
	}

	client := newFeatureFlagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "PATCH", "/v1/projects/prj_123/feature-flags/segments/segment_123", "team_123", request)
		_, _ = w.Write([]byte(`{
			"id":"segment_123",
			"slug":"beta-users",
			"label":"Beta Users",
			"projectId":"prj_123",
			"typeName":"segment",
			"hint":"user-id",
			"createdAt":1,
			"updatedAt":2,
			"data":{
				"include":{},
				"exclude":{},
				"rules":[
					{
						"id":"rule-1",
						"conditions":[
							{
								"lhs":{"type":"entity","kind":"user","attribute":"plan"},
								"cmp":"eq",
								"rhs":"beta"
							}
						],
						"outcome":{
							"type":"split",
							"base":{"type":"entity","kind":"user","attribute":"id"},
							"passPromille":500
						}
					}
				]
			}
		}`))
	})

	segment, err := client.UpdateFeatureFlagSegment(context.Background(), request)
	if err != nil {
		t.Fatalf("UpdateFeatureFlagSegment returned error: %v", err)
	}

	if segment.Data.Rules[0].Outcome.PassPromille == nil || *segment.Data.Rules[0].Outcome.PassPromille != 500 {
		t.Fatalf("expected passPromille 500, got %#v", segment.Data.Rules[0].Outcome.PassPromille)
	}
}

func TestDeleteFeatureFlagSegment(t *testing.T) {
	t.Parallel()

	client := newFeatureFlagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "DELETE", "/v1/projects/prj_123/feature-flags/segments/segment_123", "team_123", nil)
		w.WriteHeader(http.StatusNoContent)
	})

	err := client.DeleteFeatureFlagSegment(context.Background(), DeleteFeatureFlagSegmentRequest{
		ProjectID:       "prj_123",
		SegmentIDOrSlug: "segment_123",
		TeamID:          "team_123",
	})
	if err != nil {
		t.Fatalf("DeleteFeatureFlagSegment returned error: %v", err)
	}
}

func TestFeatureFlagSegmentRequestAliasesUseLegacySegmentID(t *testing.T) {
	t.Parallel()

	t.Run("get", func(t *testing.T) {
		t.Parallel()

		client := newFeatureFlagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			assertRequest(t, r, "GET", "/v1/projects/prj_123/feature-flags/segments/segment_legacy", "team_123", nil)
			_, _ = w.Write([]byte(`{
				"id":"segment_legacy",
				"slug":"legacy-segment",
				"label":"Legacy Segment",
				"projectId":"prj_123",
				"typeName":"segment",
				"hint":"user-id",
				"createdAt":1,
				"updatedAt":2,
				"data":{"include":{},"exclude":{},"rules":[]}
			}`))
		})

		_, err := client.GetFeatureFlagSegment(context.Background(), GetFeatureFlagSegmentRequest{
			ProjectID: "prj_123",
			TeamID:    "team_123",
			SegmentID: "segment_legacy",
		})
		if err != nil {
			t.Fatalf("GetFeatureFlagSegment returned error: %v", err)
		}
	})

	t.Run("update", func(t *testing.T) {
		t.Parallel()

		client := newFeatureFlagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			assertRequest(t, r, "PATCH", "/v1/projects/prj_123/feature-flags/segments/segment_legacy", "team_123", map[string]any{
				"label": "Legacy Segment",
				"hint":  "user-id",
				"data": map[string]any{
					"include": map[string]any{},
					"exclude": map[string]any{},
					"rules":   []any{},
				},
			})
			_, _ = w.Write([]byte(`{
				"id":"segment_legacy",
				"slug":"legacy-segment",
				"label":"Legacy Segment",
				"projectId":"prj_123",
				"typeName":"segment",
				"hint":"user-id",
				"createdAt":1,
				"updatedAt":2,
				"data":{"include":{},"exclude":{},"rules":[]}
			}`))
		})

		_, err := client.UpdateFeatureFlagSegment(context.Background(), UpdateFeatureFlagSegmentRequest{
			ProjectID: "prj_123",
			TeamID:    "team_123",
			SegmentID: "segment_legacy",
			Label:     "Legacy Segment",
			Hint:      "user-id",
			Data: FeatureFlagSegmentData{
				Rules:   []FeatureFlagSegmentRule{},
				Include: map[string]map[string][]FeatureFlagSegmentValue{},
				Exclude: map[string]map[string][]FeatureFlagSegmentValue{},
			},
		})
		if err != nil {
			t.Fatalf("UpdateFeatureFlagSegment returned error: %v", err)
		}
	})

	t.Run("delete", func(t *testing.T) {
		t.Parallel()

		client := newFeatureFlagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			assertRequest(t, r, "DELETE", "/v1/projects/prj_123/feature-flags/segments/segment_legacy", "team_123", nil)
			w.WriteHeader(http.StatusNoContent)
		})

		err := client.DeleteFeatureFlagSegment(context.Background(), DeleteFeatureFlagSegmentRequest{
			ProjectID: "prj_123",
			TeamID:    "team_123",
			SegmentID: "segment_legacy",
		})
		if err != nil {
			t.Fatalf("DeleteFeatureFlagSegment returned error: %v", err)
		}
	})
}

func TestCreateFeatureFlagSDKKey(t *testing.T) {
	t.Parallel()

	request := CreateFeatureFlagSDKKeyRequest{
		ProjectID:   "prj_123",
		TeamID:      "team_123",
		Type:        "server",
		Environment: "production",
		Label:       "Primary production key",
	}

	client := newFeatureFlagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "PUT", "/v1/projects/prj_123/feature-flags/sdk-keys", "team_123", request)
		_, _ = w.Write([]byte(`{
			"hashKey":"sdk_123",
			"projectId":"prj_123",
			"type":"server",
			"environment":"production",
			"createdBy":"user_123",
			"createdAt":1,
			"updatedAt":2,
			"label":"Primary production key",
			"keyValue":"flags_sk_live_123",
			"tokenValue":"edge_token_123",
			"connectionString":"https://flags.example.com"
		}`))
	})

	key, err := client.CreateFeatureFlagSDKKey(context.Background(), request)
	if err != nil {
		t.Fatalf("CreateFeatureFlagSDKKey returned error: %v", err)
	}

	if key.HashKey != "sdk_123" {
		t.Fatalf("expected hash key sdk_123, got %q", key.HashKey)
	}
	if key.KeyValue != "flags_sk_live_123" {
		t.Fatalf("expected key value flags_sk_live_123, got %q", key.KeyValue)
	}
}

func TestListFeatureFlagSDKKeys(t *testing.T) {
	t.Parallel()

	client := newFeatureFlagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/v1/projects/prj_123/feature-flags/sdk-keys", "team_123", nil)
		_, _ = w.Write([]byte(`{
			"data":[
				{
					"hashKey":"sdk_123",
					"projectId":"prj_123",
					"type":"server",
					"environment":"production",
					"createdBy":"user_123",
					"createdAt":1,
					"updatedAt":2,
					"label":"Primary production key"
				},
				{
					"hashKey":"sdk_456",
					"projectId":"prj_123",
					"type":"client",
					"environment":"preview",
					"createdBy":"user_456",
					"createdAt":3,
					"updatedAt":4,
					"label":"Preview client key"
				}
			]
		}`))
	})

	keys, err := client.ListFeatureFlagSDKKeys(context.Background(), ListFeatureFlagSDKKeysRequest{
		ProjectID: "prj_123",
		TeamID:    "team_123",
	})
	if err != nil {
		t.Fatalf("ListFeatureFlagSDKKeys returned error: %v", err)
	}

	if len(keys) != 2 {
		t.Fatalf("expected 2 sdk keys, got %d", len(keys))
	}
	if keys[1].Label != "Preview client key" {
		t.Fatalf("expected second label Preview client key, got %q", keys[1].Label)
	}
}

func TestDeleteFeatureFlagSDKKey(t *testing.T) {
	t.Parallel()

	client := newFeatureFlagTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "DELETE", "/v1/projects/prj_123/feature-flags/sdk-keys/sdk_123", "team_123", nil)
		w.WriteHeader(http.StatusNoContent)
	})

	err := client.DeleteFeatureFlagSDKKey(context.Background(), DeleteFeatureFlagSDKKeyRequest{
		ProjectID: "prj_123",
		HashKey:   "sdk_123",
		TeamID:    "team_123",
	})
	if err != nil {
		t.Fatalf("DeleteFeatureFlagSDKKey returned error: %v", err)
	}
}

func newFeatureFlagTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client := New("test-token")
	client.baseURL = server.URL
	return client
}

func assertRequest(t *testing.T, r *http.Request, method, path, teamID string, expectedBody any) {
	t.Helper()

	if r.Method != method {
		t.Fatalf("expected method %s, got %s", method, r.Method)
	}
	if r.URL.Path != path {
		t.Fatalf("expected path %s, got %s", path, r.URL.Path)
	}
	if teamID != "" {
		if value := r.URL.Query().Get("teamId"); value != teamID {
			t.Fatalf("expected teamId %q, got %q", teamID, value)
		}
	}
	if expectedBody == nil {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("reading request body: %v", err)
		}
		if len(body) != 0 {
			t.Fatalf("expected empty body, got %s", string(body))
		}
		return
	}

	assertJSONEqual(t, expectedBody, r.Body)
}

func assertJSONEqual(t *testing.T, expected any, body io.Reader) {
	t.Helper()

	rawBody, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("reading request body: %v", err)
	}

	var got any
	if err := json.Unmarshal(rawBody, &got); err != nil {
		t.Fatalf("unmarshaling request body: %v", err)
	}

	var want any
	if err := json.Unmarshal(mustMarshal(expected), &want); err != nil {
		t.Fatalf("unmarshaling expected body: %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected request body:\nwant: %#v\ngot:  %#v", want, got)
	}
}

func floatPtr(value float64) *float64 {
	return &value
}
