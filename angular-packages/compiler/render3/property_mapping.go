package render3

// ClassPropertyName is the name of a class property that backs an input or output
// declared by a directive or component.
type ClassPropertyName = string

// BindingPropertyName is the name by which an input or output of a directive or component
// is bound in an Angular template.
type BindingPropertyName = string

// InputOrOutput represents an input or output of a directive that has both a named JavaScript
// class property and an Angular template property name used for binding.
type InputOrOutput struct {
	// ClassPropertyName is the name of the JavaScript property on the component or directive
	// instance for this input or output.
	ClassPropertyName ClassPropertyName

	// BindingPropertyName is the property name used to bind this input or output in an Angular
	// template.
	BindingPropertyName BindingPropertyName

	// IsSignal indicates whether the input or output is signal based.
	IsSignal bool
}

// ClassPropertyMapping is a mapping of component property and template binding property names,
// for example containing the inputs of a particular directive or component.
//
// A single component property has exactly one input/output annotation (and therefore one binding
// property name) associated with it, but the same binding property name may be shared across many
// component property names.
//
// Allows bidirectional querying of the mapping - looking up all inputs/outputs with a given
// property name, or mapping from a specific class property to its binding property name.
type ClassPropertyMapping[T interface {
	GetClassPropertyName() ClassPropertyName
	GetBindingPropertyName() BindingPropertyName
	GetIsSignal() bool
}] struct {
	// forwardMap maps class property names to the single InputOrOutput for that class property.
	forwardMap map[ClassPropertyName]T

	// reverseMap maps binding property names to one or more InputOrOutputs which share that name.
	reverseMap map[BindingPropertyName][]T
}

// ClassPropertyMappingGeneric is a non-generic version of ClassPropertyMapping using InputOrOutput directly.
// This is the primary type used in the codebase for parity with the TypeScript ClassPropertyMapping<T = InputOrOutput>.
type ClassPropertyMappingGeneric struct {
	forwardMap map[ClassPropertyName]*InputOrOutput
	reverseMap map[BindingPropertyName][]*InputOrOutput
}

// EmptyClassPropertyMapping constructs a ClassPropertyMappingGeneric with no entries.
func EmptyClassPropertyMapping() *ClassPropertyMappingGeneric {
	return newClassPropertyMappingFromForward(make(map[ClassPropertyName]*InputOrOutput))
}

// ClassPropertyMappingFromMappedObject constructs a ClassPropertyMappingGeneric from a primitive
// map which maps class property names to either binding property names (string) or full InputOrOutput structs.
func ClassPropertyMappingFromMappedObject(obj map[string]interface{}) *ClassPropertyMappingGeneric {
	forwardMap := make(map[ClassPropertyName]*InputOrOutput)

	for classPropertyName, value := range obj {
		var inputOrOutput *InputOrOutput
		switch v := value.(type) {
		case string:
			inputOrOutput = &InputOrOutput{
				ClassPropertyName:   classPropertyName,
				BindingPropertyName: v,
				// Inputs/outputs not captured via an explicit InputOrOutput mapping
				// value are always considered non-signal. This is the string shorthand.
				IsSignal:            false,
			}
		case *InputOrOutput:
			inputOrOutput = v
		case InputOrOutput:
			vCopy := v
			inputOrOutput = &vCopy
		}
		if inputOrOutput != nil {
			forwardMap[classPropertyName] = inputOrOutput
		}
	}

	return newClassPropertyMappingFromForward(forwardMap)
}

// MergeClassPropertyMappings merges two mappings into one, with class properties from b
// taking precedence over class properties from a.
func MergeClassPropertyMappings(a, b *ClassPropertyMappingGeneric) *ClassPropertyMappingGeneric {
	forwardMap := make(map[ClassPropertyName]*InputOrOutput)
	for k, v := range a.forwardMap {
		forwardMap[k] = v
	}
	for k, v := range b.forwardMap {
		forwardMap[k] = v
	}
	return newClassPropertyMappingFromForward(forwardMap)
}

func newClassPropertyMappingFromForward(forwardMap map[ClassPropertyName]*InputOrOutput) *ClassPropertyMappingGeneric {
	return &ClassPropertyMappingGeneric{
		forwardMap: forwardMap,
		reverseMap: reverseMapFromForwardMapGeneric(forwardMap),
	}
}

// ClassPropertyNames returns all class property names mapped in this mapping.
func (m *ClassPropertyMappingGeneric) ClassPropertyNames() []ClassPropertyName {
	result := make([]ClassPropertyName, 0, len(m.forwardMap))
	for k := range m.forwardMap {
		result = append(result, k)
	}
	return result
}

// PropertyNames returns all binding property names mapped in this mapping.
func (m *ClassPropertyMappingGeneric) PropertyNames() []BindingPropertyName {
	result := make([]BindingPropertyName, 0, len(m.reverseMap))
	for k := range m.reverseMap {
		result = append(result, k)
	}
	return result
}

// HasBindingPropertyName checks whether a mapping for the given property name exists.
func (m *ClassPropertyMappingGeneric) HasBindingPropertyName(propertyName BindingPropertyName) bool {
	_, ok := m.reverseMap[propertyName]
	return ok
}

// GetByBindingPropertyName looks up all InputOrOutputs that use this propertyName.
func (m *ClassPropertyMappingGeneric) GetByBindingPropertyName(propertyName string) []*InputOrOutput {
	if v, ok := m.reverseMap[propertyName]; ok {
		return v
	}
	return nil
}

// GetByClassPropertyName looks up the InputOrOutput associated with a classPropertyName.
func (m *ClassPropertyMappingGeneric) GetByClassPropertyName(classPropertyName string) *InputOrOutput {
	if v, ok := m.forwardMap[classPropertyName]; ok {
		return v
	}
	return nil
}

// ToDirectMappedObject converts this mapping to a primitive map which maps each class property
// directly to the binding property name associated with it.
func (m *ClassPropertyMappingGeneric) ToDirectMappedObject() map[ClassPropertyName]BindingPropertyName {
	obj := make(map[ClassPropertyName]BindingPropertyName, len(m.forwardMap))
	for classPropertyName, inputOrOutput := range m.forwardMap {
		obj[classPropertyName] = inputOrOutput.BindingPropertyName
	}
	return obj
}

// ToJointMappedObject converts this mapping to a primitive map which maps each class property
// to a transformed value.
func (m *ClassPropertyMappingGeneric) ToJointMappedObject(transform func(value *InputOrOutput) interface{}) map[ClassPropertyName]interface{} {
	obj := make(map[ClassPropertyName]interface{}, len(m.forwardMap))
	for classPropertyName, inputOrOutput := range m.forwardMap {
		obj[classPropertyName] = transform(inputOrOutput)
	}
	return obj
}

// ForEach iterates over all InputOrOutput values in this mapping.
func (m *ClassPropertyMappingGeneric) ForEach(fn func(inputOrOutput *InputOrOutput)) {
	for _, v := range m.forwardMap {
		fn(v)
	}
}

func reverseMapFromForwardMapGeneric(
	forwardMap map[ClassPropertyName]*InputOrOutput,
) map[BindingPropertyName][]*InputOrOutput {
	reverseMap := make(map[BindingPropertyName][]*InputOrOutput)
	for _, inputOrOutput := range forwardMap {
		name := inputOrOutput.BindingPropertyName
		reverseMap[name] = append(reverseMap[name], inputOrOutput)
	}
	return reverseMap
}
