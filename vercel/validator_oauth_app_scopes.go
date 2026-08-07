package vercel

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// oauthAppScopesIncludeOpenID validates that an explicitly configured scopes
// set contains "openid". The API force-includes it server-side, so omitting it
// from the config would otherwise cause a perpetual diff.
func oauthAppScopesIncludeOpenID() validator.Set {
	return &oauthAppScopesIncludeOpenIDValidator{}
}

type oauthAppScopesIncludeOpenIDValidator struct{}

func (v *oauthAppScopesIncludeOpenIDValidator) Description(ctx context.Context) string {
	return "Scopes must include \"openid\""
}

func (v *oauthAppScopesIncludeOpenIDValidator) MarkdownDescription(ctx context.Context) string {
	return "Scopes must include `openid`"
}

func (v *oauthAppScopesIncludeOpenIDValidator) ValidateSet(ctx context.Context, req validator.SetRequest, resp *validator.SetResponse) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}
	for _, element := range req.ConfigValue.Elements() {
		if element.Equal(types.StringValue("openid")) {
			return
		}
	}
	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid scopes",
		"The \"openid\" scope is always required and is force-included by the API. Add \"openid\" to the scopes set.",
	)
}
