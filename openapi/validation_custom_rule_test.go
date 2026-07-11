package openapi_test

import (
	"testing"

	validatorv10 "github.com/go-playground/validator/v10"

	"github.com/Casara/arnon/openapi"
	"github.com/Casara/arnon/validation"
)

func TestSchemaGenerator_AppliesCustomRuleSchemaEffect(t *testing.T) {
	t.Parallel()

	const tag = "cpf"

	err := validation.RegisterCustomRule(validation.CustomRule{
		Tag: tag,

		Func: func(validatorv10.FieldLevel) bool { return true },

		Schema: &validation.SchemaEffect{
			Format:  "cpf",
			Pattern: `^\d{11}$`,
		},
	})
	if err != nil {
		t.Fatalf("RegisterCustomRule failed: %v", err)
	}

	type request struct {
		Document string `json:"document" validate:"cpf"`
	}

	schema := openapi.NewSchemaGenerator().GenerateSchema(request{})

	property, ok := schema.Properties["document"]
	if !ok {
		t.Fatalf("expected schema property %q, got %+v", "document", schema.Properties)
	}

	if property.Format != "cpf" {
		t.Errorf("expected format %q, got %q", "cpf", property.Format)
	}

	if property.Pattern != `^\d{11}$` {
		t.Errorf("expected pattern %q, got %q", `^\d{11}$`, property.Pattern)
	}
}

func TestSchemaGenerator_UnregisteredTagLeavesSchemaUntouched(t *testing.T) {
	t.Parallel()

	type request struct {
		Nickname string `json:"nickname" validate:"not_a_registered_rule"`
	}

	schema := openapi.NewSchemaGenerator().GenerateSchema(request{})

	property, ok := schema.Properties["nickname"]
	if !ok {
		t.Fatalf("expected schema property %q, got %+v", "nickname", schema.Properties)
	}

	if property.Format != "" {
		t.Errorf("expected empty format, got %q", property.Format)
	}

	if property.Pattern != "" {
		t.Errorf("expected empty pattern, got %q", property.Pattern)
	}
}
