package tfjson

import (
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/zclconf/go-cty/cty"
)

type Resource struct {
	Version            int                `json:"version"`
	Schema             map[string]*Schema `json:"schema"`
	Description        string             `json:"description"`
	DeprecationMessage string             `json:"deprecation_message"`
}

type Schema struct {
	Required           bool        `json:"required"`
	Description        string      `json:"description"`
	Deprecated         bool        `json:"deprecated"`
	DeprecationMessage string      `json:"deprecation_message"`
	Observation        bool        `json:"observation"`
	Sensitive          bool        `json:"sensitive"`
	MinItems           int         `json:"min_items"`
	MaxItems           int         `json:"max_items"`
	Type               ValueType   `json:"type"`
	Elem               interface{} `json:"elem"`
	ConfigMode         ConfigMode  `json:"config_mode"`
	Default            string      `json:"default"`
}

type ValueType string

type ConfigMode string

const (
	TypeString  ValueType = "string"
	TypeFloat   ValueType = "float"
	TypeBool    ValueType = "bool"
	TypeList    ValueType = "list"
	TypeSet     ValueType = "set"
	TypeMap     ValueType = "map"
	TypeObject  ValueType = "object"
	TypeInvalid ValueType = "invalid"

	ConfigModeAuto ConfigMode = "auto"
	ConfigModeAttr ConfigMode = "attr"

	TimeoutsConfigKey string = "timeouts"
)

func GetResourceMap(resourceSchemas map[string]*tfjson.Schema) map[string]*Resource {
	resourceMap := make(map[string]*Resource, len(resourceSchemas))
	for k, v := range resourceSchemas {
		resourceMap[k] = resourceFromTFJSONSchema(v)
	}
	return resourceMap
}

func resourceFromTFJSONSchema(s *tfjson.Schema) *Resource {
	r := &Resource{Version: int(s.Version)} //nolint:gosec
	if s.Block == nil {
		return r
	}

	toSchemaMap := make(map[string]*Schema, len(s.Block.Attributes)+len(s.Block.NestedBlocks))

	for k, v := range s.Block.Attributes {
		if v.AttributeNestedType != nil {
			toSchemaMap[k] = tfJSONNestedAttributeTypeToSchema(v)
		} else {
			toSchemaMap[k] = tfJSONAttributeToSchema(v)
		}
	}
	for k, v := range s.Block.NestedBlocks {
		// CRUD timeouts are not part of the generated MR API,
		// they cannot be dynamically configured and they are determined by either
		// the underlying Terraform resource configuration or the upjet resource
		// configuration. Please also see config.Resource.OperationTimeouts.
		if k == TimeoutsConfigKey {
			continue
		}
		toSchemaMap[k] = tfJSONBlockTypeToSchema(v)
	}
	r.Schema = toSchemaMap
	r.Description = s.Block.Description
	r.DeprecationMessage = deprecatedMessage(s.Block.Deprecated)
	return r
}

func tfJSONAttributeToSchema(attr *tfjson.SchemaAttribute) *Schema {
	sch := &Schema{
		Observation:        isObservation(attr.Computed, attr.Optional),
		Required:           attr.Required,
		Description:        attr.Description,
		DeprecationMessage: deprecatedMessage(attr.Deprecated),
		Sensitive:          attr.Sensitive,
	}
	if err := schemaTypeFromCtyType(attr.AttributeType, sch); err != nil {
		panic(err)
	}
	return sch
}

func tfJSONNestedAttributeTypeToSchema(nestedAttr *tfjson.SchemaAttribute) *Schema {
	na := nestedAttr.AttributeNestedType
	sch := &Schema{
		MinItems: int(na.MinItems),
		MaxItems: int(na.MaxItems),
		Required: nestedAttr.Required,
	}
	switch na.NestingMode { //nolint:exhaustive
	case tfjson.SchemaNestingModeSet:
		sch.Type = TypeSet
	case tfjson.SchemaNestingModeList:
		sch.Type = TypeList
	case tfjson.SchemaNestingModeMap:
		sch.Type = TypeMap
	case tfjson.SchemaNestingModeSingle, tfjson.SchemaNestingModeGroup:
		sch.Type = TypeObject
	default:
		panic("unhandled nesting mode: " + na.NestingMode)
	}

	res := &Resource{}
	res.Schema = make(map[string]*Schema, len(na.Attributes))
	for key, attr := range na.Attributes {
		if attr.AttributeNestedType != nil {
			res.Schema[key] = tfJSONNestedAttributeTypeToSchema(attr)
		} else {
			res.Schema[key] = tfJSONAttributeToSchema(attr)
		}
	}
	sch.Elem = res
	return sch
}

func tfJSONBlockTypeToSchema(nb *tfjson.SchemaBlockType) *Schema {
	sch := &Schema{
		MinItems: int(nb.MinItems), //nolint:gosec
		MaxItems: int(nb.MaxItems), //nolint:gosec
	}
	// Note(turkenh): Schema representation returned by the cli for block types
	// does not have optional or computed fields. So, we are trying to infer
	// those fields by doing the opposite of what is done here:
	// https://github.com/hashicorp/terraform-plugin-sdk/blob/6461ac6e9044a44157c4e2c8aec0f1ab7efc2055/helper/schema/core_schema.go#L204
	sch.Required = true
	sch.Observation = false
	if nb.MinItems == 0 {
		sch.Required = false
	}
	if nb.MinItems == 0 && nb.MaxItems == 0 && sch.Required {
		sch.Observation = true
	}

	switch nb.NestingMode { //nolint:exhaustive
	case tfjson.SchemaNestingModeSet:
		sch.Type = TypeSet
	case tfjson.SchemaNestingModeList:
		sch.Type = TypeList
	case tfjson.SchemaNestingModeMap:
		sch.Type = TypeMap
	case tfjson.SchemaNestingModeSingle:
		sch.Type = TypeList
		sch.MinItems = 0
		sch.Required = hasBlockRequiredChild(nb)
		if sch.Required {
			sch.MinItems = 1
		}
		sch.MaxItems = 1
	default:
		panic("unhandled nesting mode: " + nb.NestingMode)
	}

	if nb.Block == nil {
		return sch
	}

	sch.Description = nb.Block.Description
	sch.DeprecationMessage = deprecatedMessage(nb.Block.Deprecated)

	res := &Resource{}
	res.Schema = make(map[string]*Schema, len(nb.Block.Attributes)+len(nb.Block.NestedBlocks))
	for key, attr := range nb.Block.Attributes {
		res.Schema[key] = tfJSONAttributeToSchema(attr)
	}
	for key, block := range nb.Block.NestedBlocks {
		// Please note that unlike the resource-level CRUD timeout configuration
		// blocks (as mentioned above), we will generate the timeouts parameters
		// for any nested configuration blocks, *if they exist*.
		// We can prevent them here, but they are different than the resource's
		// top-level CRUD timeouts, so we have opted to generate them.
		res.Schema[key] = tfJSONBlockTypeToSchema(block)
	}
	sch.Elem = res
	return sch
}

func schemaTypeFromCtyType(typ cty.Type, sch *Schema) error {
	configMode := ConfigModeAuto

	switch {
	case typ.IsPrimitiveType():
		sch.Type = primitiveToSchemaType(typ)
	case typ.IsCollectionType():
		var elemType any
		et := typ.ElementType()
		switch {
		case et.IsPrimitiveType():
			elemType = &Schema{
				Type:        primitiveToSchemaType(et),
				Required:    sch.Required,
				Observation: sch.Observation,
			}
		case et.IsCollectionType():
			elemType = &Schema{
				Type:        collectionToSchemaType(et),
				Required:    sch.Required,
				Observation: sch.Observation,
			}
			if err := schemaTypeFromCtyType(et, elemType.(*Schema)); err != nil {
				return err
			}
		case et.IsObjectType():
			configMode = ConfigModeAttr
			res := &Resource{}
			res.Schema = make(map[string]*Schema, len(et.AttributeTypes()))
			for key, attrTyp := range et.AttributeTypes() {
				s := &Schema{
					Required:    sch.Required,
					Observation: sch.Observation,
				}
				if et.AttributeOptional(key) {
					s.Required = false
				}

				if err := schemaTypeFromCtyType(attrTyp, s); err != nil {
					return err
				}
				res.Schema[key] = s
			}
			elemType = res
		default:
			return errors.Errorf("unexpected cty.Type %s", typ.GoString())
		}
		sch.ConfigMode = configMode
		sch.Type = collectionToSchemaType(typ)
		sch.Elem = elemType
	case typ.IsTupleType():
		return errors.New("cannot convert cty TupleType to schema v2 type")
	case typ.IsObjectType():
		typ.AttributeTypes()
		res := &Resource{}
		res.Schema = make(map[string]*Schema, len(typ.AttributeTypes()))
		for key, attrTyp := range typ.AttributeTypes() {
			s := &Schema{
				Required:    sch.Required,
				Observation: sch.Observation,
			}
			if err := schemaTypeFromCtyType(attrTyp, s); err != nil {
				return err
			}
			res.Schema[key] = s
		}
		sch.ConfigMode = configMode
		sch.Type = TypeObject
		sch.Elem = res
	case typ.Equals(cty.DynamicPseudoType):
		return errors.New("cannot convert cty DynamicPseudoType to schema v2 type")
	}

	return nil
}

func primitiveToSchemaType(typ cty.Type) ValueType {
	switch {
	case typ.Equals(cty.String):
		return TypeString
	case typ.Equals(cty.Number):
		// TODO(turkenh): Figure out handling floats with IntOrString on type
		//  builder side
		return TypeFloat
	case typ.Equals(cty.Bool):
		return TypeBool
	}
	return TypeInvalid
}

func collectionToSchemaType(typ cty.Type) ValueType {
	switch {
	case typ.IsSetType():
		return TypeSet
	case typ.IsListType():
		return TypeList
	case typ.IsMapType():
		return TypeMap
	}
	return TypeInvalid
}

func isObservation(computed, optional bool) bool {
	// NOTE(muvaf): If a field is not optional but computed, then it's
	// definitely an observation field.
	// If it's optional but also computed, then it means the field has a server
	// side default but user can change it, so it needs to go to parameters.
	return computed && !optional
}

func (e ValueType) String() string {
	return string(e)
}
