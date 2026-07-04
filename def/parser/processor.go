package parser

import (
	"fmt"
	"slices"
	"strings"

	def "github.com/netgarden/rrpc/def"
)

func NewProcessor() *Processor {
	return &Processor{}
}

type Processor struct{}

func (p *Processor) Process(sources []*def.SourceOrigin) (*def.Module, error) {

	var err error

	module := def.NewModule()

	for _, source := range sources {
		err = p.processSource(module, source)
		if err != nil {
			return nil, err
		}
	}

	err = p.computeModuleData(module)
	if err != nil {
		return nil, err
	}

	return module, nil
}

func (p *Processor) processSource(module *def.Module, sourceOrigin *def.SourceOrigin) error {

	source, err := p.sourceOriginToSource(module, sourceOrigin)
	if err != nil {
		return err
	}

	err = p.addSourceToModule(module, source)
	if err != nil {
		return err
	}

	return nil
}

func (p *Processor) sourceOriginToSource(module *def.Module, sourceOrigin *def.SourceOrigin) (*def.Source, error) {

	source := def.NewSource(module, sourceOrigin.SourceName, sourceOrigin.SourcePath)
	source.Name = sourceOrigin.Name
	source.Version = sourceOrigin.Version
	source.Includes = sourceOrigin.Includes

	for _, optionDef := range sourceOrigin.Options {
		source.Options[optionDef.Name] = optionDef.Value
	}

	for _, structDef := range sourceOrigin.Structs {
		structDef.Source = source
		source.Structs[structDef.Name] = structDef
	}

	for _, serviceDef := range sourceOrigin.Services {

		serviceDef.Path = serviceDef.Name
		if p.hasAnnotation(serviceDef.Annotations, "Path") {
			serviceDef.Path = p.getAnnotationValue(serviceDef.Annotations, "Path")
		}

		serviceDef.Source = source
		source.Services[serviceDef.Name] = serviceDef

		for _, methodDef := range serviceDef.Methods {

			methodDef.Path = methodDef.Name
			if p.hasAnnotation(methodDef.Annotations, "Path") {
				methodDef.Path = p.getAnnotationValue(methodDef.Annotations, "Path")
			}

			typeAnnotation := p.getAnnotationValue(methodDef.Annotations, "method")
			if typeAnnotation != "" {
				methodDef.Type = typeAnnotation
			}

			if methodDef.Type == "GET" || methodDef.Type == "HEAD" || methodDef.Type == "DELETE" {
				for _, input := range methodDef.Inputs {
					if input.Type == "bytes" {
						return nil, fmt.Errorf(
							"method %s.%s: bytes input is not allowed for %s methods",
							serviceDef.Name,
							methodDef.Name,
							methodDef.Type,
						)
					}
				}
			} else if methodDef.Type != "POST" && methodDef.Type != "PUT" && len(methodDef.Inputs) > 0 {
				return nil, fmt.Errorf(
					"method %s.%s has inputs but type is not POST or PUT",
					serviceDef.Name,
					methodDef.Name,
				)
			}


		}

	}

	for _, errorDef := range sourceOrigin.Errors {
		errorDef.Source = source
		source.Errors[errorDef.Name] = errorDef
	}

	return source, nil
}

func (p *Processor) getAnnotationValue(annotations []*def.Annotation, name string) string {
	for _, annotation := range annotations {
		if annotation.Name == name {
			return annotation.Value
		}
	}
	return ""
}

func (p *Processor) hasAnnotation(annotations []*def.Annotation, name string) bool {
	for _, annotation := range annotations {
		if annotation.Name == name {
			return true
		}
	}
	return false
}

func (p *Processor) addSourceToModule(module *def.Module, source *def.Source) error {

	if module.MainSource == nil {
		module.MainSource = source
	}

	module.Sources[source.SourceName] = source
	source.Module = module

	// -- Name

	if module.MainSource != source && source.Name != "" {
		return fmt.Errorf("name can be defined only in main source file")
	} else if module.MainSource == source && source.Name == "" {
		return fmt.Errorf("name is not specified in main source file")
	}

	if module.MainSource == source {
		module.Name = source.Name
	}

	// -- Version

	if module.MainSource != source && source.Version != "" {
		return fmt.Errorf("version can be defined only in main source file")
	} else if module.MainSource == source && source.Version == "" {
		return fmt.Errorf("version is not specified in main source file")
	}

	if module.MainSource == source {
		module.Version = source.Version
	}

	// -- Options

	if module.MainSource != source && len(source.Options) > 0 {
		return fmt.Errorf("options can be defined only in main source file")
	}

	if module.MainSource == source {
		module.Options = source.Options
	}

	// -- Structs

	for _, structDef := range source.Structs {

		var structDef0 *def.Struct
		var ok bool

		if structDef0, ok = module.Structs[structDef.Name]; !ok {
			module.Structs[structDef.Name] = structDef
			continue
		}

		if structDef0.Source != structDef.Source {
			return fmt.Errorf(
				"struct %s from %s is already defined in %s",
				structDef.Name,
				structDef.Source.SourceName,
				structDef0.Source.SourceName,
			)
		}

	}

	// -- Services

	for _, serviceDef := range source.Services {

		var serviceDef0 *def.Service
		var ok bool

		if serviceDef0, ok = module.Services[serviceDef.Name]; !ok {
			module.Services[serviceDef.Name] = serviceDef
			continue
		}

		if serviceDef0.Source != serviceDef.Source {
			return fmt.Errorf(
				"service %s from %s is already defined in %s",
				serviceDef.Name,
				serviceDef.Source.SourceName,
				serviceDef0.Source.SourceName,
			)
		}

	}

	// -- Errors

	for _, errorDef := range source.Errors {

		var errorDef0 *def.Error
		var ok bool

		if errorDef0, ok = module.Errors[errorDef.Name]; !ok {
			module.Errors[errorDef.Name] = errorDef
			continue
		}

		if errorDef0.Source != errorDef.Source {
			return fmt.Errorf(
				"error %s from %s is already defined in %s",
				errorDef.Name,
				errorDef.Source.SourceName,
				errorDef0.Source.SourceName,
			)
		}

	}

	// -- Includes

	for _, include := range source.Includes {
		if !slices.Contains(module.Includes, include) {
			module.Includes = append(module.Includes, include)
		}
	}

	return nil
}

func (p *Processor) checkDataTypes(module *def.Module) error {

	var structNames = make([]string, 0, len(module.Structs))
	for _, structDef := range module.Structs {
		structNames = append(structNames, structDef.Name)
	}

	for _, structDef := range module.Structs {
		for _, fieldDef := range structDef.Fields {
			err := p.checkDataType(fieldDef.Type, structNames)
			if err != nil {
				return err
			}
		}
	}

	for _, serviceDef := range module.Services {
		for _, methodDef := range serviceDef.Methods {
			for _, inputDef := range methodDef.Inputs {
				err := p.checkDataType(inputDef.Type, structNames)
				if err != nil {
					return err
				}
			}
			for _, outputDef := range methodDef.Outputs {
				err := p.checkDataType(outputDef.Type, structNames)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (p *Processor) checkDataType(dataType string, structNames []string) error {

	if strings.HasPrefix(dataType, "[]") {
		return p._checkDataType(dataType[2:], structNames)
	}

	if strings.HasPrefix(dataType, "map[") {
		parts := strings.SplitN(dataType[4:], "]", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid map type %s", dataType)
		}
		err := p._checkDataType(parts[0], structNames)
		if err != nil {
			return err
		}
		return p._checkDataType(parts[1], structNames)
	}

	return p._checkDataType(dataType, structNames)
}

func (p *Processor) _checkDataType(dataType string, structNames []string) error {

	found := false

	if slices.Contains(structNames, dataType) {
		found = true
	}

	if slices.Contains(SupportedPrimitiveTypes, dataType) {
		found = true
	}

	if slices.Contains(SupportedComplexTypes, dataType) {
		found = true
	}

	if !found {
		return fmt.Errorf("unknown or unsupported type %s", dataType)
	}

	return nil
}

func (p *Processor) computeModuleData(module *def.Module) error {

	structNames := make([]string, 0, len(module.Structs))
	for _, structDef := range module.Structs {
		structNames = append(structNames, structDef.Name)
	}

	for _, serviceDef := range module.Services {
		for _, methodDef := range serviceDef.Methods {
			for _, inputDef := range methodDef.Inputs {
				if slices.Contains(structNames, inputDef.Type) {
					inputDef.IsStruct = true
				}
			}
			for _, outputDef := range methodDef.Outputs {
				if slices.Contains(structNames, outputDef.Type) {
					outputDef.IsStruct = true
				}
			}

			var hasStructInput, hasBytesInput bool
			for _, inputDef := range methodDef.Inputs {
				if inputDef.IsStruct {
					hasStructInput = true
				} else if inputDef.Type == "bytes" {
					hasBytesInput = true
				}
			}
			if hasStructInput && hasBytesInput {
				methodDef.ContentType = "multipart/form-data"
			} else if hasStructInput {
				methodDef.ContentType = "application/json"
			} else if hasBytesInput {
				methodDef.ContentType = "application/octet-stream"
			}
		}
	}

	for _, structDef := range module.Structs {
		for _, fieldDef := range structDef.Fields {
			if slices.Contains(structNames, fieldDef.Type) {
				fieldDef.IsStruct = true
			}
		}
	}

	return nil
}
