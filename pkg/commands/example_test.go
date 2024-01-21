package commands_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"

	"github.com/sergeizaitcev/metrics/pkg/commands"
)

type Config struct {
	commands.UnimplementedConfig

	Field string `env:"FIELD" json:"field"`

	Struct struct {
		Field2 int `env:"FIELD_2" json:"field_2"`
	} `json:"struct"`
}

func (c *Config) SetFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.Field, "f1", "1", "usage 1")
	fs.IntVar(&c.Struct.Field2, "f2", 2, "usage 2")
}

func ExampleCommand() {
	run := func(_ context.Context, c *Config) error {
		b, err := json.MarshalIndent(c, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		return nil
	}
	commands.Execute("test", run)

	// Output:
	//
	// {
	//   "field": "1",
	//   "struct": {
	//     "field_2": 2
	//   }
	// }
}
