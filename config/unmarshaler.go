package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"gopkg.in/yaml.v3"
)

type jsonUnmarshaler struct{}

func (jsonUnmarshaler) Unmarshal(b []byte, tgt any) error {
	return json.Unmarshal(b, tgt)
}

type yamlUnmarshaler struct{}

func (yamlUnmarshaler) Unmarshal(b []byte, tgt any) error {
	return yaml.Unmarshal(b, tgt)
}

type textUnmarshaler struct{}

func (textUnmarshaler) Unmarshal(b []byte, tgt any) error {
	v := reflect.ValueOf(tgt)
	if v.Type().Kind() != reflect.Pointer {
		return fmt.Errorf("target should be **string kind, but got '%v'", v.Type())
	}
	if v.Type().Elem().Kind() != reflect.Pointer {
		return fmt.Errorf("target should be **string kind, but got '%v'", v.Type())
	}
	if v.Type().Elem().Elem().Kind() != reflect.String {
		return fmt.Errorf("target should be pointer of string type, but got '%v'", v.Type())
	}
	if v.IsNil() {
		return errors.New("unmarshal target is nil")
	}

	str := string(b)
	ptr := reflect.ValueOf(&str)
	v.Elem().Set(ptr)
	return nil
}
