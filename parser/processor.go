package parser

import (
	"fmt"
	def "github.com/netgarden/rrpc-def"
	"slices"
	"strings"
)

func NewProcessor() *Processor {
	return &Processor{}
}

type Processor struct{}

func (p *Processor) Process(sources []*def.SourceOrigin) (*def.Definition, error) {

	var err error

	definition := def.NewDefinition()

	for _, source := range sources {
		err = p.processSource(definition, source)
		if err != nil {
			return nil, err
		}
	}

	return definition, nil
}

func (p *Processor) processSource(definition *def.Definition, sourceOrigin *def.SourceOrigin) error {

	source, err := p.sourceOriginToSource(sourceOrigin)
	if err != nil {
		return err
	}

	err = p.addSourceToDefinition(definition, source)
	if err != nil {
		return err
	}

	return nil
}

func (p *Processor) sourceOriginToSource(sourceOrigin *def.SourceOrigin) (*def.Source, error) {

	source := def.NewSource(sourceOrigin.Path)
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
		pathAnnotation := p.getAnnotationValue(serviceDef.Annotations, "path")
		if pathAnnotation != "" {
			serviceDef.Path = pathAnnotation
		}

		serviceDef.Source = source
		source.Services[serviceDef.Name] = serviceDef

		for _, methodDef := range serviceDef.Methods {

			methodDef.Path = methodDef.Name
			pathAnnotation := p.getAnnotationValue(methodDef.Annotations, "path")
			if pathAnnotation != "" {
				methodDef.Path = pathAnnotation
			}

			typeAnnotation := p.getAnnotationValue(methodDef.Annotations, "method")
			if typeAnnotation != "" {
				methodDef.Type = typeAnnotation
			}

			if methodDef.Type != "POST" && len(methodDef.Inputs) > 0 {
				return nil, fmt.Errorf("method %s.%s has inputs but type is not POST", serviceDef.Name, methodDef.Name)
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

func (p *Processor) addSourceToDefinition(definition *def.Definition, source *def.Source) error {

	if definition.MainSource == nil {
		definition.MainSource = source
	}

	definition.Sources[source.Name] = source

	// -- Version

	if definition.MainSource != source && source.Version != "" {
		return fmt.Errorf("version can be defined only in main source file")
	} else if definition.MainSource == source && source.Version == "" {
		return fmt.Errorf("version is not specified in main source file")
	}

	definition.Version = source.Version

	// -- Options

	if definition.MainSource != source && len(source.Options) > 0 {
		return fmt.Errorf("options can be defined only in main source file")
	}

	definition.Options = source.Options

	// -- Structs

	for _, structDef := range source.Structs {

		var structDef0 *def.Struct
		var ok bool

		if structDef0, ok = definition.Structs[structDef.Name]; !ok {
			definition.Structs[structDef.Name] = structDef
			continue
		}

		if structDef0.Source != structDef.Source {
			return fmt.Errorf(
				"struct %s from %s is already defined in %s",
				structDef.Name,
				structDef.Source.Name,
				structDef0.Source.Name,
			)
		}

	}

	// -- Services

	for _, serviceDef := range source.Services {

		var serviceDef0 *def.Service
		var ok bool

		if serviceDef0, ok = definition.Services[serviceDef.Name]; !ok {
			definition.Services[serviceDef.Name] = serviceDef
			continue
		}

		if serviceDef0.Source != serviceDef.Source {
			return fmt.Errorf(
				"service %s from %s is already defined in %s",
				serviceDef.Name,
				serviceDef.Source.Name,
				serviceDef0.Source.Name,
			)
		}

	}

	// -- Errors

	for _, errorDef := range source.Errors {

		var errorDef0 *def.Error
		var ok bool

		if errorDef0, ok = definition.Errors[errorDef.Name]; !ok {
			definition.Errors[errorDef.Name] = errorDef
			continue
		}

		if errorDef0.Source != errorDef.Source {
			return fmt.Errorf(
				"error %s from %s is already defined in %s",
				errorDef.Name,
				errorDef.Source.Name,
				errorDef0.Source.Name,
			)
		}

	}

	// -- Includes

	for _, include := range source.Includes {
		if !slices.Contains(definition.Includes, include) {
			definition.Includes = append(definition.Includes, include)
		}
	}

	return nil
}

func (p *Processor) checkDataTypes(definition *def.Definition) error {

	var structNames = make([]string, 0, len(definition.Structs))
	for _, structDef := range definition.Structs {
		structNames = append(structNames, structDef.Name)
	}

	for _, structDef := range definition.Structs {
		for _, fieldDef := range structDef.Fields {
			err := p.checkDataType(fieldDef.Type, structNames)
			if err != nil {
				return err
			}
		}
	}

	for _, serviceDef := range definition.Services {
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

func (p *Processor) computeDefintionData(definition *def.Definition) error {

	structNames := make([]string, 0, len(definition.Structs))
	for _, structDef := range definition.Structs {
		structNames = append(structNames, structDef.Name)
	}

	for _, serviceDef := range definition.Services {
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
		}
	}

	for _, structDef := range definition.Structs {
		for _, fieldDef := range structDef.Fields {
			if slices.Contains(structNames, fieldDef.Type) {
				fieldDef.IsStruct = true
			}
		}
	}

	return nil
}
