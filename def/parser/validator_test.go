package parser

import (
	"testing"

	def "github.com/netgarden/rrpc/def"
)

func TestIsQueryParamMapType(t *testing.T) {
	cases := []struct {
		fieldType string
		want      bool
	}{
		{"map[string]string", true},
		{"map[string]int", true},
		{"map[string]bool", true},
		{"map[int]string", false},   // key must be string (query keys always are)
		{"map[string]bytes", false}, // bytes isn't a valid map value for query decoding
		{"string", false},
		{"[]string", false},
	}

	for _, c := range cases {
		if got := isQueryParamMapType(c.fieldType); got != c.want {
			t.Errorf("isQueryParamMapType(%q) = %v, want %v", c.fieldType, got, c.want)
		}
	}
}

func TestValidateQueryParamStruct_AllowsStringKeyedPrimitiveMap(t *testing.T) {
	v := NewValidator()
	service := &def.Service{Name: "Hosts"}
	method := &def.Method{Name: "list", Type: "GET"}
	structDef := &def.Struct{
		Name: "DatatableRequest",
		Fields: []*def.Field{
			def.NewField("page", "int"),
			def.NewField("filters", "map[string]string"),
		},
	}
	module := &def.Module{Structs: map[string]*def.Struct{"DatatableRequest": structDef}}
	service.Source = &def.Source{Module: module}
	input := def.NewField("request", "DatatableRequest")

	if err := v.validateQueryParamStruct(service, method, input); err != nil {
		t.Errorf("expected map[string]string field to be accepted for a GET method, got error: %v", err)
	}
}

func TestValidateQueryParamStruct_RejectsNonPrimitiveMapValue(t *testing.T) {
	v := NewValidator()
	service := &def.Service{Name: "Hosts"}
	method := &def.Method{Name: "list", Type: "GET"}
	structDef := &def.Struct{
		Name: "BadRequest",
		Fields: []*def.Field{
			def.NewField("nested", "map[string]SomeStruct"),
		},
	}
	module := &def.Module{Structs: map[string]*def.Struct{"BadRequest": structDef}}
	service.Source = &def.Source{Module: module}
	input := def.NewField("request", "BadRequest")

	if err := v.validateQueryParamStruct(service, method, input); err == nil {
		t.Error("expected an error for a map with a non-primitive value type on a GET method")
	}
}
