package modelreg

import (
	"strings"
	"testing"
)

// The registry we actually ship must pass its own validator, or the gate is
// theatre.
func TestValidate_ShippedRegistryIsValid(t *testing.T) {
	if err := InitModelsConfig(); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := GlobalModelsConfig.Validate(); err != nil {
		t.Fatalf("the shipped registry does not pass validation:\n%v", err)
	}
}

func TestValidate_CatchesRoleNamingAMissingModel(t *testing.T) {
	c := &ModelsConfig{
		Models: map[string]ModelConfig{"real": {APIName: "real", Provider: "openrouter"}},
		Roles:  map[string][]string{"executor": {"deleted-row"}},
	}
	err := c.Validate()
	if err == nil {
		t.Fatal("a role naming a model that does not exist must not publish")
	}
	if !strings.Contains(err.Error(), "deleted-row") {
		t.Errorf("error must name the offending entry; got %v", err)
	}
}

// The accepted cost of D1(a): pricing now reaches production without a rebuild,
// so the validator must cover it.
func TestValidate_CatchesNegativePricing(t *testing.T) {
	c := &ModelsConfig{Models: map[string]ModelConfig{
		"bad": {APIName: "bad", Provider: "openrouter", Pricing: Pricing{InputPer1K: -1}},
	}}
	err := c.Validate()
	if err == nil {
		t.Fatal("negative pricing must not publish — cost accounting reads this field")
	}
	if !strings.Contains(err.Error(), "pricing") {
		t.Errorf("error should say pricing; got %v", err)
	}
}

// Zero pricing is legitimate (local ollama rows are free). Flagging it would
// train publishers to ignore the validator.
func TestValidate_ZeroPricingIsNotAnError(t *testing.T) {
	c := &ModelsConfig{Models: map[string]ModelConfig{
		"local": {APIName: "local", Provider: "ollama"},
	}}
	if err := c.Validate(); err != nil {
		t.Errorf("zero pricing must be allowed; got %v", err)
	}
}

// Every problem at once, so one publish attempt yields one complete fix list.
func TestValidate_ReportsAllProblemsNotJustTheFirst(t *testing.T) {
	c := &ModelsConfig{
		Models: map[string]ModelConfig{"a": {Provider: "openrouter"}, "b": {APIName: "b"}},
		Roles:  map[string][]string{"executor": {"nope"}},
	}
	err := c.Validate()
	if err == nil {
		t.Fatal("expected problems")
	}
	for _, want := range []string{"no api_name", "no provider", "nope"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("missing %q in:\n%v", want, err)
		}
	}
}
