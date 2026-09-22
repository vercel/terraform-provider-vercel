package vercel

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const testOpenAPI = `{"paths":{
  "/v9/projects/{idOrName}":{"patch":{"requestBody":{"content":{"application/json":{"schema":{"properties":{"unrelated":{"enum":[1,2,3]},"framework":{"enum":[null,"nextjs","ruby","future-framework"]}}}}}}}},
  "/v1/webhooks":{"post":{"requestBody":{"content":{"application/json":{"schema":{"properties":{"events":{"items":{"enum":["deployment.created","deployment.checkrun.start","deployment.checkrun.cancel","deployment.checks.succeeded","deployment.checks.failed","future.event"]}}}}}}}}}
}}`

func TestOpenAPIValidators(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		fmt.Fprint(w, testOpenAPI)
	}))
	defer server.Close()
	source := &openAPIEnums{url: server.URL, client: server.Client()}

	for _, tt := range []struct {
		name  string
		value types.String
		bad   bool
	}{
		{"framework", types.StringValue("nextjs"), false},
		{"framework", types.StringValue("ruby"), false},
		{"framework", types.StringValue("future-framework"), false},
		{"framework", types.StringValue("missing"), true},
		{"framework", types.StringValue(""), true},
		{"framework", types.StringValue("deployment.created"), true},
		{"webhook event", types.StringValue("deployment.checkrun.start"), false},
		{"webhook event", types.StringValue("deployment.checkrun.cancel"), false},
		{"webhook event", types.StringValue("deployment.checks.succeeded"), false},
		{"webhook event", types.StringValue("deployment.checks.failed"), false},
		{"webhook event", types.StringValue("future.event"), false},
		{"webhook event", types.StringValue("nextjs"), true},
		{"webhook event", types.StringValue("missing"), true},
	} {
		t.Run(tt.name+"/"+tt.value.String(), func(t *testing.T) {
			v := validatorOpenAPIEnum{source: source, name: tt.name}
			response := &validator.StringResponse{}
			v.ValidateString(context.Background(), validator.StringRequest{
				ConfigValue: tt.value, Path: path.Root("value"),
			}, response)
			if response.Diagnostics.HasError() != tt.bad {
				t.Fatalf("diagnostics = %v, want error %t", response.Diagnostics, tt.bad)
			}
		})
	}
	if requests.Load() != 1 {
		t.Fatalf("requests = %d, want one shared fetch", requests.Load())
	}
}

func TestOpenAPIValidatorSkipsUnknownAndNull(t *testing.T) {
	for _, name := range []string{"framework", "webhook event"} {
		v := validatorOpenAPIEnum{name: name} // A fetch would panic on the nil source.
		v.Description(context.Background())
		v.MarkdownDescription(context.Background())
		for _, value := range []types.String{types.StringNull(), types.StringUnknown()} {
			response := &validator.StringResponse{}
			v.ValidateString(context.Background(), validator.StringRequest{ConfigValue: value}, response)
			if response.Diagnostics.HasError() {
				t.Fatal(response.Diagnostics)
			}
		}
	}
}

func TestOpenAPIFailuresCanRetry(t *testing.T) {
	for _, tt := range []struct {
		name, body, want string
		status           int
	}{
		{"HTTP error", "", "status code 503", http.StatusServiceUnavailable},
		{"invalid JSON", "{", "decoding", http.StatusOK},
		{"missing paths", `{}`, "decoding OpenAPI framework", http.StatusOK},
		{"missing framework", strings.Replace(testOpenAPI, `"framework"`, `"renamed"`, 1), "decoding OpenAPI framework", http.StatusOK},
		{"missing event items", strings.Replace(testOpenAPI, `"items":`, `"renamed":`, 1), "missing webhook event items", http.StatusOK},
		{"non-string framework", strings.Replace(testOpenAPI, `[null,"nextjs","ruby","future-framework"]`, `[123]`, 1), "decoding OpenAPI framework", http.StatusOK},
		{"empty events", strings.Replace(testOpenAPI, `"items":{"enum":`, `"items":{"unused":`, 1), "no webhook event enum", http.StatusOK},
		{"null-only framework", strings.Replace(testOpenAPI, `[null,"nextjs","ruby","future-framework"]`, `[null]`, 1), "no framework enum", http.StatusOK},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if requests.Add(1) == 1 {
					w.WriteHeader(tt.status)
					fmt.Fprint(w, tt.body)
					return
				}
				fmt.Fprint(w, testOpenAPI)
			}))
			defer server.Close()
			source := &openAPIEnums{url: server.URL, client: server.Client()}
			v := validatorOpenAPIEnum{source: source, name: "framework"}
			response := &validator.StringResponse{}
			v.ValidateString(context.Background(), validator.StringRequest{
				ConfigValue: types.StringValue("nextjs"), Path: path.Root("framework"),
			}, response)
			if len(response.Diagnostics) != 1 || !strings.Contains(response.Diagnostics[0].Detail(), tt.want) {
				t.Fatalf("diagnostics = %v, want %q", response.Diagnostics, tt.want)
			}
			if _, err := source.load(context.Background()); err != nil {
				t.Fatalf("retry failed: %v", err)
			}
		})
	}
}

func TestOpenAPICanceledFetchCanRetry(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, testOpenAPI)
	}))
	defer server.Close()
	source := &openAPIEnums{url: server.URL, client: server.Client()}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := source.load(ctx); err == nil {
		t.Fatal("expected canceled fetch to fail")
	}
	if _, err := source.load(context.Background()); err != nil {
		t.Fatalf("retry failed: %v", err)
	}
}

func TestOpenAPIConcurrentFetch(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		fmt.Fprint(w, testOpenAPI)
	}))
	defer server.Close()
	source := &openAPIEnums{url: server.URL, client: server.Client()}
	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			if _, err := source.load(context.Background()); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if requests.Load() != 1 {
		t.Fatalf("requests = %d, want 1", requests.Load())
	}
}
