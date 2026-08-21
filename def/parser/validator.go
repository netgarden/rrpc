package parser

import (
	"fmt"
	"slices"
	"strings"

	def "github.com/netgarden/rrpc/def"
)

func NewValidator() *Validator {
	return &Validator{}
}

type Validator struct{}

func (v *Validator) Validate(definition *def.Module) error {

	var err error

	for _, service := range definition.Services {
		err = v.validateService(service)
		if err != nil {
			return err
		}
	}

	return nil
}

func (v *Validator) validateService(service *def.Service) error {

	var err error

	for _, method := range service.Methods {
		err = v.validateMethod(service, method)
		if err != nil {
			return err
		}
	}

	return nil
}

func (v *Validator) validateMethod(service *def.Service, method *def.Method) error {

	if method.Streaming {
		return v.validateStreamingMethod(service, method)
	}

	var methodHasStructInput bool
	var methodHasBytesInput bool
	for i, inputDef := range method.Inputs {

		if inputDef.IsStruct {
			if i != 0 {
				return fmt.Errorf(
					"method %s.%s: struct type input can be defined only as first parameter",
					service.Name,
					method.Name,
				)
			}
			if methodHasStructInput {
				return fmt.Errorf(
					"method %s.%s: method can only have one struct type input parameter",
					service.Name,
					method.Name,
				)
			}
			methodHasStructInput = true

		} else if inputDef.Type == "bytes" {
			if methodHasBytesInput {
				return fmt.Errorf(
					"method %s.%s: method can only have one bytes type input parameter",
					service.Name,
					method.Name,
				)
			}
			methodHasBytesInput = true

		} else {
			return fmt.Errorf(
				"method %s.%s: unsupported input parametr type: %s",
				service.Name,
				method.Name,
				inputDef.Type,
			)
		}

	}

	if len(method.Outputs) > 1 {
		return fmt.Errorf(
			"method %s.%s: only one output parameter is supported",
			service.Name,
			method.Name,
		)
	}

	if method.Type == "GET" || method.Type == "HEAD" || method.Type == "DELETE" {
		for _, inputDef := range method.Inputs {
			if inputDef.IsStruct {
				if err := v.validateQueryParamStruct(service, method, inputDef); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// validateStreamingMethod enforces the shape rules for @Stream/
// @InputStream/@OutputStream methods: exactly the input/output count each
// direction implies, and every declared message type is either a struct
// (typed JSON-per-frame messages) or the "bytes" primitive (raw byte
// stream, see rrpc.RawStream/RawInStream/RawOutStream) - no other
// primitive maps to a meaningful per-frame encoding.
func (v *Validator) validateStreamingMethod(service *def.Service, method *def.Method) error {

	switch method.StreamDirection {
	case "both":
		if len(method.Inputs) != 1 || len(method.Outputs) != 1 {
			return fmt.Errorf(
				"method %s.%s: @Stream requires exactly one input and one output",
				service.Name, method.Name,
			)
		}
	case "in":
		if len(method.Inputs) != 1 || len(method.Outputs) != 0 {
			return fmt.Errorf(
				"method %s.%s: @InputStream requires exactly one input and no output",
				service.Name, method.Name,
			)
		}
	case "out":
		if len(method.Inputs) != 0 || len(method.Outputs) != 1 {
			return fmt.Errorf(
				"method %s.%s: @OutputStream requires exactly one output and no input",
				service.Name, method.Name,
			)
		}
	default:
		return fmt.Errorf(
			"method %s.%s: unknown stream direction %q",
			service.Name, method.Name, method.StreamDirection,
		)
	}

	fields := make([]*def.Field, 0, len(method.Inputs)+len(method.Outputs))
	fields = append(fields, method.Inputs...)
	fields = append(fields, method.Outputs...)
	for _, field := range fields {
		if !field.IsStruct && field.Type != "bytes" {
			return fmt.Errorf(
				"method %s.%s: streaming message type %s must be a struct or bytes, got %s",
				service.Name, method.Name, field.Name, field.Type,
			)
		}
	}

	// A bidirectional stream is either fully typed (rrpc.Stream[In, Out])
	// or fully raw (rrpc.RawStream) - there's no single generic type for
	// a mixed struct-in/bytes-out combination, so reject it rather than
	// let codegen produce something inconsistent.
	if method.StreamDirection == "both" {
		inIsBytes := method.Inputs[0].Type == "bytes"
		outIsBytes := method.Outputs[0].Type == "bytes"
		if inIsBytes != outIsBytes {
			return fmt.Errorf(
				"method %s.%s: @Stream's input and output must both be structs or both be bytes, not mixed",
				service.Name, method.Name,
			)
		}
	}

	return nil
}

func (v *Validator) validateQueryParamStruct(service *def.Service, method *def.Method, input *def.Field) error {
	structDef, ok := service.Source.Module.Structs[input.Type]
	if !ok {
		return fmt.Errorf("method %s.%s: struct type %s not found", service.Name, method.Name, input.Type)
	}
	for _, field := range structDef.Fields {
		if slices.Contains(SupportedPrimitiveTypes, field.Type) || field.Type == "bytes" {
			continue
		}
		if isQueryParamMapType(field.Type) {
			continue
		}
		return fmt.Errorf(
			"method %s.%s: struct %s field %s must be a primitive type (or a map[string]<primitive>) for %s methods, got %s",
			service.Name, method.Name, input.Type, field.Name, method.Type, field.Type,
		)
	}
	return nil
}

// isQueryParamMapType reports whether fieldType is a "map[string]V" where
// V is itself a supported primitive — the shape rrpc.DecodeQueryParams
// can reconstruct from bracket-notation query keys (e.g.
// "filters[name]=foo"). The key type is required to be "string" since
// query parameter keys always are; other map shapes aren't decodable
// from a query string and stay rejected, same as before.
func isQueryParamMapType(fieldType string) bool {
	if !strings.HasPrefix(fieldType, "map[") {
		return false
	}
	parts := strings.SplitN(fieldType[4:], "]", 2)
	if len(parts) != 2 {
		return false
	}
	keyType, valueType := parts[0], parts[1]
	return keyType == "string" && slices.Contains(SupportedPrimitiveTypes, valueType)
}
