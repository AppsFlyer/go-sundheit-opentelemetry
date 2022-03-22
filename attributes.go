package sundheitotel

import (
	"encoding/json"

	"github.com/mitchellh/mapstructure"
	"go.opentelemetry.io/otel/attribute"
)

type Attribute struct {
	Key   string
	Value AttributeValue
}

type AttributeValue struct {
	Type  string
	Value interface{}
}

func serializeAttributes(tags []attribute.KeyValue) (string, error) {
	set := attribute.NewSet(tags...)
	serialized, err := set.MarshalJSON()
	if err != nil {
		return "", err
	}
	return string(serialized), nil
}

func deserializeAttributes(tags string) ([]attribute.KeyValue, error) {
	var result []attribute.KeyValue
	var attributes []interface{}
	if err := json.Unmarshal([]byte(tags), &attributes); err != nil {
		return result, nil
	}
	for _, attr := range attributes {
		var a Attribute
		if err := mapstructure.Decode(attr, &a); err != nil {
			return nil, err
		}
		result = append(result, mapAttributeType(a))
	}
	return result, nil
}

func mapAttributeType(a Attribute) attribute.KeyValue {
	switch a.Value.Type {
	case attribute.BOOL.String():
		return attribute.Bool(a.Key, a.Value.Value.(bool))
	case attribute.STRING.String():
		return attribute.String(a.Key, a.Value.Value.(string))
	}
	return attribute.KeyValue{}
}
