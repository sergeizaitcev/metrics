package commands

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/caarlos0/env/v10"
)

// ExecFunc определяет функцию выполнения командой.
type ExecFunc[T Config] func(ctx context.Context, c T) error

// Command определяет команду выполнения.
type Command[T Config] struct {
	fs     *flag.FlagSet
	args   []string
	exec   ExecFunc[T]
	config T
}

// New возвращает новый экземпляр Command.
func New[T Config](name string, exec ExecFunc[T]) *Command[T] {
	return &Command[T]{
		fs:     flag.NewFlagSet(name, flag.ContinueOnError),
		args:   cleanArgs(os.Args[1:]),
		exec:   exec,
		config: makeConfig[T](),
	}
}

func cleanArgs(args []string) []string {
	isTested := func(arg string) bool {
		return strings.HasPrefix(arg, "-test.") || strings.HasPrefix(arg, "--test.")
	}
	clean := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if isTested(args[i]) {
			continue
		}
		clean = append(clean, args[i])
	}
	return clean
}

// Execute запускает команду и блокируется до её завершения.
func (cmd *Command[T]) Execute(ctx context.Context) (err error) {
	if err = cmd.initConfig(); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			cmd.usage()
			return nil
		}
		return fmt.Errorf("failed to init config: %w", err)
	}
	if err = cmd.exec(ctx, cmd.config); err != nil {
		return fmt.Errorf("failed to execute: %w", err)
	}
	return nil
}

// Execute запускает команду и блокируется до её завершения.
func Execute[T Config](name string, execFunc ExecFunc[T]) {
	signals := []os.Signal{
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	}

	ctx, cancel := notifyContext(signals...)
	defer cancel()

	executeContext(ctx, name, execFunc)
}

func notifyContext(signals ...os.Signal) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	sigs := make(chan os.Signal, 1)
	done := make(chan struct{})

	signal.Notify(sigs, signals...)

	go func() {
		select {
		case <-sigs:
			// Перенос строки после символов ^C, ^D и т.д.
			fmt.Println()
			cancel()
		case <-ctx.Done():
		}
		signal.Stop(sigs)
		close(done)
	}()

	return ctx, func() {
		cancel()
		<-done
	}
}

func executeContext[T Config](ctx context.Context, name string, execFunc ExecFunc[T]) {
	cmd := New(name, execFunc)
	if err := cmd.Execute(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

// initConfig инициализирует и валидирует конфигурацию.
func (cmd *Command[T]) initConfig() (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("panic: %s", e)
		}
	}()

	cmd.init()

	if err = cmd.parseConfigPath(); err != nil {
		return fmt.Errorf("file parsing: %w", err)
	}
	if err = cmd.parseFlags(); err != nil {
		return fmt.Errorf("flag parsing: %w", err)
	}
	if err = cmd.parseEnv(); err != nil {
		return fmt.Errorf("env parsing: %w", err)
	}

	if err = cmd.config.Validate(); err != nil {
		return fmt.Errorf("config validation: %w", err)
	}

	return nil
}

// init инициализирует первичную конфигурацию.
func (cmd *Command[T]) init() {
	// Отключение автоматического срабатывания Usage
	// в случае возникновения ошибки при парсинге флагов.
	cmd.fs.Usage = func() {}

	out := cmd.fs.Output()
	cmd.config.SetFlags(cmd.fs)
	cmd.fs.SetOutput(out)
}

// usage выводит в cmd.fs.Output() формат использования команды.
func (cmd *Command[T]) usage() {
	fmt.Fprintf(cmd.fs.Output(), "Usage of %s:\n", cmd.fs.Name())
	cmd.fs.PrintDefaults()
}

// parseConfigPath парсит файл конфигурации.
func (cmd *Command[T]) parseConfigPath() error {
	reader, ok := any(cmd.config).(io.ReaderFrom)
	if !ok {
		return nil
	}

	field, tag := lookupConfigPath(cmd.config)
	if !field.IsValid() {
		return nil
	}

	value, err := cmd.lookupFlagValue(field)
	if err != nil {
		return fmt.Errorf("looking up for a flag value: %w", err)
	}
	if value != "" {
		err = readConfig(reader, value)
		if err != nil {
			return fmt.Errorf("writing a config using the flag: %w", err)
		}
	}

	value, err = lookupEnvValue(tag)
	if err != nil {
		return fmt.Errorf("lookup up for a env value: %w", err)
	}
	if value != "" {
		err = readConfig(reader, value)
		if err != nil {
			return fmt.Errorf("writing a config using the env: %w", err)
		}
	}

	return nil
}

func readConfig(r io.ReaderFrom, filename string) error {
	f, err := os.OpenFile(filename, os.O_RDONLY, 0o400)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	_, err = r.ReadFrom(bufio.NewReader(f))
	if err != nil {
		return fmt.Errorf("writing a config from a file: %w", err)
	}

	return nil
}

// lookupFlagValue выполняет поиск значения флага для поля с типом
// commands.ConfigPath и возвращает передаваемое значение аргумента командной
// строки.
func (cmd *Command[T]) lookupFlagValue(x reflect.Value) (string, error) {
	if x.Kind() != reflect.Pointer {
		if !x.CanAddr() {
			return "", nil
		}
		x = x.Addr()
	}

	var name string

	cmd.fs.VisitAll(func(f *flag.Flag) {
		// f.Value априори всегда является указателем.
		y := reflect.ValueOf(f.Value)
		if x.UnsafePointer() == y.UnsafePointer() {
			name = f.Name
		}
	})
	if name == "" {
		return "", nil
	}

	// Необходимо установить значение по умолчанию,
	// если оно было установлено.
	tempConfigPath := x.Elem().Interface().(ConfigPath)

	tempFlagSet := flag.NewFlagSet("", flag.ContinueOnError)
	tempFlagSet.Usage = func() {}
	tempFlagSet.SetOutput(io.Discard)
	tempFlagSet.Var(&tempConfigPath, name, "")

	// Игнорируются все ошибки, кроме flag.ErrHelp,
	// т.к. это не основной набор флагов.
	err := tempFlagSet.Parse(cmd.args)
	if err != nil && errors.Is(err, flag.ErrHelp) {
		return "", err
	}

	return tempConfigPath.String(), nil
}

func lookupEnvValue(tag reflect.StructTag) (string, error) {
	value := func() string {
		value := os.Getenv(tag.Get("env"))
		if value == "" {
			return os.Getenv(tag.Get("envDefault"))
		}
		return value
	}()
	if value == "" {
		return "", nil
	}

	var configPath ConfigPath

	err := configPath.Set(value)
	if err != nil {
		return "", err
	}

	return configPath.String(), nil
}

// parseFlags парсит флаги командной строки.
func (cmd *Command[T]) parseFlags() error {
	return cmd.fs.Parse(cmd.args)
}

// parseEnv парсит переменные окружения.
func (cmd *Command[T]) parseEnv() error {
	opts := env.Options{FuncMap: customParsers}
	rt := reflect.TypeOf(cmd.config)
	if rt.Kind() == reflect.Pointer {
		return env.ParseWithOptions(cmd.config, opts)
	}
	return env.ParseWithOptions(&cmd.config, opts)
}

// NOTE: во флагах и переменных окружения автотестов для интервалов передаются
// значения без единиц измерений, из-за этого стандартный time.ParseDuration
// не работает.
var durationType reflect.Type = reflect.TypeOf((*time.Duration)(nil)).Elem()

var customParsers = map[reflect.Type]env.ParserFunc{
	durationType: func(s string) (any, error) {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse int: %w", err)
		}
		return time.Duration(v) * time.Second, nil
	},
}
