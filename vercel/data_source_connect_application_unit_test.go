package vercel

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestConnectApplicationLookupValidation(t *testing.T) {
	ctx := context.Background()
	resp := &datasource.SchemaResponse{}
	newConnectApplicationDataSource().Schema(ctx, datasource.SchemaRequest{}, resp)
	for _, tc := range []struct {
		name    string
		id, uid types.String
		invalid bool
	}{
		{"ID", types.StringValue("scl_oidc"), types.StringNull(), false},
		{"UID", types.StringNull(), types.StringValue("oauth/company-sso"), false},
		{"neither", types.StringNull(), types.StringNull(), true},
		{"both", types.StringValue("scl_oidc"), types.StringValue("oauth/company-sso"), true},
		{"empty UID", types.StringNull(), types.StringValue(""), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := tfsdk.Config{Schema: resp.Schema}
			objectType := resp.Schema.Type().TerraformType(ctx).(tftypes.Object)
			fields := map[string]tftypes.Value{}
			for name, fieldType := range objectType.AttributeTypes {
				fields[name] = tftypes.NewValue(fieldType, nil)
			}
			fields["id"], _ = tc.id.ToTerraformValue(ctx)
			fields["uid"], _ = tc.uid.ToTerraformValue(ctx)
			config.Raw = tftypes.NewValue(objectType, fields)
			invalid := false
			for key, value := range map[string]types.String{"id": tc.id, "uid": tc.uid} {
				for _, check := range resp.Schema.Attributes[key].(schema.StringAttribute).Validators {
					validation := &validator.StringResponse{}
					check.ValidateString(ctx, validator.StringRequest{Path: path.Root(key), Config: config, ConfigValue: value}, validation)
					invalid = invalid || validation.Diagnostics.HasError()
				}
			}
			if invalid != tc.invalid {
				t.Fatalf("invalid lookup = %v, want %v", invalid, tc.invalid)
			}
		})
	}
}
