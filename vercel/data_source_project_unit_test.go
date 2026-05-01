package vercel

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func TestProjectDataSourceEnvironmentValueIsSensitive(t *testing.T) {
	res := newProjectDataSource()

	resp := &datasource.SchemaResponse{}
	res.Schema(context.Background(), datasource.SchemaRequest{}, resp)

	environmentAttr, ok := resp.Schema.Attributes["environment"].(schema.SetNestedAttribute)
	if !ok {
		t.Fatalf("environment attribute has unexpected type: %T", resp.Schema.Attributes["environment"])
	}

	valueAttr, ok := environmentAttr.NestedObject.Attributes["value"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("environment.value attribute has unexpected type: %T", environmentAttr.NestedObject.Attributes["value"])
	}

	if !valueAttr.Sensitive {
		t.Fatal("environment.value should be sensitive")
	}
}
