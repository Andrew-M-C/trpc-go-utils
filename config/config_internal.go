package config

var internal = struct {
	unmarshalerByName map[Encoding]unmarshaler
}{
	unmarshalerByName: map[Encoding]unmarshaler{
		JSON: jsonUnmarshaler{},
		YAML: yamlUnmarshaler{},
	},
}

type unmarshaler interface {
	Unmarshal([]byte, any) error
}
