package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"

	yaml "gopkg.in/yaml.v3"
)

type jqFlags struct {
	compact                     bool
	nullAsSingleInputValue      bool
	exitStatusCodeBasedOnOutput bool
	slurp                       bool
	raw                         bool
	rawString                   bool
	color                       bool
	monochrome                  bool
	sort                        bool
	tab                         bool
	arg                         string
	slurpfile                   string
	rawfile                     string
}

type yq struct {
	returnYAML    bool
	jqCmd         exec.Cmd
	jqStdout      io.ReadCloser
	jqStdinWriter io.WriteCloser
	files         []string

	jqFlags
}

func transformToYAML(reader io.Reader, writer io.Writer) error {
	dec := json.NewDecoder(reader)
	enc := yaml.NewEncoder(writer)
	enc.SetIndent(2)
	var err error
	for {
		var m interface{}
		if err := dec.Decode(&m); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if err := enc.Encode(m); err != nil {
			return err
		}
	}
	return err
}

func transformToJSON(reader io.Reader, writer io.WriteCloser) error {
	dec := yaml.NewDecoder(reader)
	enc := json.NewEncoder(writer)
	enc.SetEscapeHTML(false)
	var err error
	for {
		var m interface{}
		if err := dec.Decode(&m); err != nil {
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
		}
		if m == nil {
			continue
		}
		if err = enc.Encode(m); err != nil {
			return err
		}
	}
	return err
}

func (yq *yq) parseFlags(f *flag.FlagSet, osArgs []string) error {
	f.BoolVar(&(yq.returnYAML), "y", false, "Transcode jq JSON output back "+
		"into YAML and emit it")
	f.BoolVar(&(yq.returnYAML), "yaml-output", false, "Transcode jq JSON output back "+
		"into YAML and emit it")
	f.BoolVar(&(yq.compact), "c", false, "jq Flag: compact instead of "+
		"pretty-printed output")
	f.BoolVar(&(yq.exitStatusCodeBasedOnOutput), "e", false, "jq Flag: set the "+
		"exit status code based on the output")
	f.BoolVar(&(yq.nullAsSingleInputValue), "n", false, "jq Flag: use `null` "+
		"as the single input value")
	f.BoolVar(&(yq.raw), "r", false, "jq Flag: output raw strings, not JSON "+
		"texts")
	f.BoolVar(&(yq.slurp), "s", false, "jq Flag: read (slurp) all inputs into "+
		"an array; apply filter to it")
	f.BoolVar(&(yq.rawString), "R", false, "jq Flag: read raw strings, not "+
		"JSON texts")
	f.BoolVar(&(yq.color), "C", false, "jq Flag: colorize JSON")
	f.BoolVar(&(yq.monochrome), "M", false, "jq Flag: monochrome (don't "+
		"colorize JSON)")
	f.BoolVar(&(yq.sort), "S", false, "jq Flag: sort keys of objects on "+
		"output")
	f.BoolVar(&(yq.tab), "tab", false, "jq Flag: use tabs for indentation")
	f.StringVar(&(yq.arg), "arg", "", "jq Flag: 'a v' set variable $a to value "+
		"<v>")
	f.StringVar(&(yq.slurpfile), "slurpfile", "", "jq Flag: 'a f' set variable "+
		"$a to an array of JSON texts read from <f>")
	f.StringVar(&(yq.rawfile), "rawfile", "", "set variable $a to a string "+
		"consisting of the contents of <f>")

	if len(osArgs) == 1 {
		f.Usage()
		return errors.New("no arguments passed")
	}

	err := f.Parse(osArgs[1:])
	return err
}

func (yq *yq) appendArgs(argName string, osArgs []string) {
	yq.jqCmd.Args = append(yq.jqCmd.Args, argName)
	idx := 0
	for i, arg := range osArgs {
		if arg == argName {
			idx = i
			break
		}
	}

	yq.jqCmd.Args = append(yq.jqCmd.Args, osArgs[idx+1:idx+3]...)
}

func (yq *yq) compileJqCmd(osArgs []string, stderr io.Writer) error {
	var f flag.FlagSet

	f.SetOutput(stderr)
	f.Usage = func() {
		fmt.Fprintf(stderr, "Usage of %s:\n", osArgs[0])
		f.PrintDefaults()
	}

	if err := yq.parseFlags(&f, osArgs); err != nil {
		return errors.New("")
	}

	flagArgs := f.Args()

	skippedArgs := 1
	yq.jqCmd.Args = append(yq.jqCmd.Args, yq.jqCmd.Path)
	if yq.compact {
		yq.jqCmd.Args = append(yq.jqCmd.Args, "-c")
	}
	if yq.nullAsSingleInputValue {
		yq.jqCmd.Args = append(yq.jqCmd.Args, "-n")
	}
	if yq.exitStatusCodeBasedOnOutput {
		yq.jqCmd.Args = append(yq.jqCmd.Args, "-e")
	}
	if yq.raw {
		yq.jqCmd.Args = append(yq.jqCmd.Args, "-r")
	}
	if yq.slurp {
		yq.jqCmd.Args = append(yq.jqCmd.Args, "-s")
	}
	if yq.rawString {
		yq.jqCmd.Args = append(yq.jqCmd.Args, "-R")
	}
	if yq.color {
		yq.jqCmd.Args = append(yq.jqCmd.Args, "-C")
	}
	if yq.monochrome {
		yq.jqCmd.Args = append(yq.jqCmd.Args, "-M")
	}
	if yq.sort {
		yq.jqCmd.Args = append(yq.jqCmd.Args, "-S")
	}
	if yq.tab {
		yq.jqCmd.Args = append(yq.jqCmd.Args, "--tab")
	}
	if yq.arg != "" {
		skippedArgs = skippedArgs + 1
		yq.appendArgs("--arg", osArgs)
	}
	if yq.slurpfile != "" {
		skippedArgs = skippedArgs + 1
		yq.appendArgs("--slurpfile", osArgs)
	}
	if yq.rawfile != "" {
		skippedArgs = skippedArgs + 1
		yq.appendArgs("--rawfile", osArgs)
	}

	yq.jqCmd.Args = append(yq.jqCmd.Args, flagArgs[skippedArgs-1])

	for _, arg := range flagArgs[skippedArgs:] {
		if _, err := os.Stat(arg); err != nil {
			return err
		}

		yq.files = append(yq.files, arg)
	}

	reader, writer := io.Pipe()
	yq.jqStdinWriter = writer
	yq.jqCmd.Stdin = reader

	if yq.returnYAML {
		var stdoutPipe io.ReadCloser
		stdoutPipe, err := yq.jqCmd.StdoutPipe()

		if err != nil {
			return err
		}
		yq.jqStdout = stdoutPipe
	} else {
		yq.jqCmd.Stdout = os.Stdout
	}

	yq.jqCmd.Stderr = os.Stderr

	return nil
}

func (yq *yq) run() error {
	var err error
	if err = yq.jqCmd.Start(); err != nil {
		return err
	}

	if len(yq.files) == 0 {
		err = transformToJSON(os.Stdin, yq.jqStdinWriter)

		if err != nil {
			yq.jqStdinWriter.Close()
			return err
		}
	} else {
		for _, file := range yq.files {
			file, err := os.Open(file)
			if err != nil {
				yq.jqStdinWriter.Close()
				return err
			}

			if err = transformToJSON(file, yq.jqStdinWriter); err != nil {
				yq.jqStdinWriter.Close()
				return err
			}
		}
	}
	yq.jqStdinWriter.Close()

	if yq.returnYAML {
		if err := transformToYAML(yq.jqStdout, os.Stdout); err != nil {
			yq.jqStdinWriter.Close()
			return err
		}
	}

	yq.jqCmd.Wait()

	return nil
}

func main() {
	var y yq

	if path, err := exec.LookPath("jq"); err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	} else {
		y.jqCmd.Path = path
	}

	if err := y.compileJqCmd(os.Args, os.Stderr); err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	if err := y.run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
