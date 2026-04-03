package parser

import (
	"fmt"

	def "github.com/netgarden/rrpc-def"
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

	return nil
}
