package plugin_test

import (
	_ "embed"
	"os"
	"testing"

	"github.com/Andrew-M-C/trpc-go-utils/plugin"
	"github.com/smartystreets/goconvey/convey"
	"trpc.group/trpc-go/trpc-go"
)

var (
	cv = convey.Convey
	so = convey.So
	eq = convey.ShouldEqual

	notNil = convey.ShouldNotBeNil
)

func TestMain(m *testing.M) {
	_ = m.Run()
}

func TestBind(t *testing.T) {
	const yamlTarget = "./trpc_go.yaml"
	_ = os.WriteFile(yamlTarget, testYaml, 0644)
	defer os.Remove(yamlTarget)

	cv("读取配置", t, func() {
		holder := testYamlConfig{}
		plugin.Bind("test_type", "test_name", &holder)

		s := trpc.NewServer()
		so(s, notNil)

		so(holder.String, eq, "Hello, world!")
		so(holder.Int, eq, 12345678)
	})
}

type testYamlConfig struct {
	String string `yaml:"string"`
	Int    int    `yaml:"int"`
}

//go:embed bind_test.yaml
var testYaml []byte
