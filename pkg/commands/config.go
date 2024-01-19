package commands

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"reflect"
)

// Config описывает конфигурацию приложения.
//
// Значения Config имеют следующий приоритет (от высшего к низшему)
//
//  1. Переменные окружения
//  2. Флаги командной строки
//  3. Файл конфигурации, переданный через переменные окружения
//  4. Файл конфигурации, переданный через флаги командной строки
//  5. Значения по умолчанию
//
// Реализация Config обязательно должна быть структурой и обязательно должна
// встраивать в себя commands.UnimplementedConfig.
//
// Если Config предусматривает файл конфигурации, то он должен реализовывать
// интерфейс io.ReaderFrom, а путь к файлу должен содержаться в поле с типом
// commands.ConfigPath, например:
//
//	type Config struct {
//		commands.UnimplementedConfig
//
//		ConfigPath commands.ConfigPath `env:"CONFIG"   json:"-"`
//		MyField	   string              `env:"MY_FIELD" json:"my_field"`
//		...
//	}
//
//	func (c *Config) ReadFrom(r io.Reader) (int64, error) {
//		dec := json.NewDecoder(r)
//		err := dec.Decode(c)
//		if err != nil {
//			return 0, err
//		}
//		return dec.InputOffset(), nil
//	}
//
//	func (c *Config) SetFlags(fs *flag.FlagSet) {
//		// c.ConfigPath = "/default/path/to/config"
//		fs.Var(&c.ConfigPath, "c", "path to config")
//		fs.StringVar(&c.MyField, "f", "field", "my field")
//	}
type Config interface {
	// SetFlags устанавливает флаги командной строки.
	SetFlags(fs *flag.FlagSet)

	// Validate возвращает ошибку, если конфиг не валиден.
	Validate() error

	mustEmbedding()
}

var _ Config = (*UnimplementedConfig)(nil)

type UnimplementedConfig struct{}

func (UnimplementedConfig) SetFlags(*flag.FlagSet) {}
func (UnimplementedConfig) Validate() error        { return nil }
func (UnimplementedConfig) mustEmbedding()         {}

var _ flag.Value = (*ConfigPath)(nil)

// ConfigPath определяет путь к файлу конфигурации.
type ConfigPath string

// String возвращает путь к файлу конфигурации.
func (c *ConfigPath) String() string { return string(*c) }

// Set устанавливает путь к файлу конфигурации.
func (c *ConfigPath) Set(value string) error {
	info, err := os.Stat(value)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			err = fmt.Errorf("no such file: %s", value)
		}
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("%s must be a file", value)
	}
	*c = ConfigPath(value)
	return nil
}

// makeConfig создает и возвращает новый экземпляр T.
func makeConfig[T Config]() T {
	var zero T
	makeStruct(reflect.ValueOf(&zero).Elem())
	return zero
}

// makeStruct рекурсивно инициализирует структуру и все её экспортируемые поля,
// которые имеют тип reflect.Struct.
func makeStruct(rv reflect.Value) {
	rv = makeValue(rv)
	if !rv.IsValid() || rv.Kind() != reflect.Struct {
		return
	}
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		field := reflectType(rt.Field(i).Type)
		if field.Kind() == reflect.Struct {
			makeStruct(rv.Field(i))
		}
	}
}

// makeValue инициализирует пустое значение и возвращает его.
func makeValue(rv reflect.Value) reflect.Value {
	for rv.Kind() == reflect.Pointer {
		if !rv.CanSet() {
			return reflect.Value{}
		}
		if rv.IsNil() {
			rv.Set(reflect.New(rv.Type().Elem()))
		}
		rv = rv.Elem()
	}
	return rv
}

// lookupConfigPath ищет в T поле с типом commands.ConfigPath и возвращает его
// вместе с тегом; если поле является нулевым указателем, то оно будет
// проинициализировано.
func lookupConfigPath[T Config](c T) (reflect.Value, reflect.StructTag) {
	target := reflect.TypeOf((*ConfigPath)(nil)).Elem()
	return lookupField(reflect.ValueOf(&c).Elem(), target)
}

// lookupField рекурсивно ищет в структуре поле с типом target и возвращает
// его вместе с тегом; если поле является нулевым указателем, то оно будет
// проинициализировано.
func lookupField(rv reflect.Value, target reflect.Type) (reflect.Value, reflect.StructTag) {
	rv = reflectValue(rv)
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		fieldType := reflectType(field.Type)
		if fieldType == target {
			rv = makeValue(rv.Field(i))
			return rv, field.Tag
		}
		if fieldType.Kind() == reflect.Struct {
			rv, tag := lookupField(rv.Field(i), target)
			if rv.IsValid() {
				return rv, tag
			}
		}
	}
	return reflect.Value{}, reflect.StructTag("")
}

// reflectValue разыменовывает reflect.Value и возвращает его.
func reflectValue(rv reflect.Value) reflect.Value {
	for rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	return rv
}

// reflectType разыменовывает reflect.Type и возвращает его.
func reflectType(rt reflect.Type) reflect.Type {
	for rt.Kind() == reflect.Pointer {
		rt = rt.Elem()
	}
	return rt
}
