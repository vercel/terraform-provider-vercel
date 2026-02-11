# Agent Guidelines for Vercel Terraform Provider

## Build/Test Commands
- Build: `task build` (uses goreleaser)
- Test all: `task test` (runs acceptance tests with TF_ACC=true)
- Test specific: `task test -- -run 'TestAcc_Project*'` (regex pattern matching)
- Lint: `task lint` (runs staticcheck, tfproviderlint, gofmt, go vet)
- Docs: `task docs` (generates terraform docs)

## Code Style
- Package naming: `package vercel` for main code, `package vercel_test` for tests
- Imports: Group stdlib, third-party, then local imports with blank lines between groups
- Types: Use terraform-plugin-framework types (types.String, types.Bool, etc.)
- Naming: CamelCase for exported, camelCase for unexported, snake_case for terraform attributes
- Error handling: Return errors with context, use fmt.Errorf for wrapping
- Comments: Only add comments for exported functions/types or complex logic

## Testing Patterns
- Test functions: `func TestAcc_ResourceName(t *testing.T)`
- Use `resource.Test()` with `ProtoV6ProviderFactories: testAccProtoV6ProviderFactories`
- Helper functions: `testAccResourceExists()`, `testAccResourceDestroy()`
- Environment variables required: VERCEL_API_TOKEN, VERCEL_TERRAFORM_TESTING_TEAM
- For write-only env var support, model schema as exactly one of `value`/`value_wo`, and keep `value` null in state when `value_wo` is used so ephemeral values are not persisted.
- Do not reuse a resource state/config struct in data sources when schemas differ; Terraform framework decoding fails on extra struct fields (for example resource-only `value_wo`).

## File Structure
- Resources: `vercel/resource_*.go` and `vercel/resource_*_test.go`
- Data sources: `vercel/data_source_*.go` and `vercel/data_source_*_test.go`
- Client: `client/*.go` for API interactions
