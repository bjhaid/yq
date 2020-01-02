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
	argjson                     string
	slurpfile                   string
	rawfile                     string
	args                        string
	jsonargs                    string
}

type yq struct {
	returnYAML bool
	jqCmd      exec.Cmd
	jqStdin    io.WriteCloser
	jqStdout   io.ReadCloser
	files      []string

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

func transformToJSON(reader io.Reader, fn func(byteArray []byte) error) error {
	dec := yaml.NewDecoder(reader)
	var b []byte
	var err error
	for {
		var m interface{}
		if err := dec.Decode(&m); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if m == nil {
			continue
		}
		if b, err = json.Marshal(m); err != nil {
			return err
		}
		if err := fn(b); err != nil {
			return err
		}
	}
	return nil
}

func (yq *yq) parseFlags() error {
	flag.BoolVar(&(yq.returnYAML), "y", false, "Transcode jq JSON output back "+
		"into YAML and emit it")
	flag.BoolVar(&(yq.returnYAML), "yaml-output", false, "Transcode jq JSON output back "+
		"into YAML and emit it")
	flag.BoolVar(&(yq.compact), "c", false, "jq Flag: compact instead of "+
		"pretty-printed output")
	flag.BoolVar(&(yq.nullAsSingleInputValue), "n", false, "jq Flag: use `null` "+
		"as the single input value")
	flag.BoolVar(&(yq.exitStatusCodeBasedOnOutput), "e", false, "jq Flag: set the "+
		"exit status code based on the output")
	flag.BoolVar(&(yq.slurp), "s", false, "jq Flag: read (slurp) all inputs into "+
		"an array; apply filter to it")
	flag.BoolVar(&(yq.raw), "r", false, "jq Flag: output raw strings, not JSON "+
		"texts")
	flag.BoolVar(&(yq.rawString), "R", false, "jq Flag: read raw strings, not "+
		"JSON texts")
	flag.BoolVar(&(yq.color), "C", false, "jq Flag: colorize JSON")
	flag.BoolVar(&(yq.monochrome), "M", false, "jq Flag: monochrome (don't "+
		"colorize JSON)")
	flag.BoolVar(&(yq.sort), "S", false, "jq Flag: sort keys of objects on "+
		"output")
	flag.BoolVar(&(yq.tab), "tab", false, "jq Flag: use tabs for indentation")
	flag.StringVar(&(yq.arg), "arg", "", "jq Flag: 'a v' set variable $a to value "+
		"<v>")
	flag.StringVar(&(yq.argjson), "argjson", "", "jq Flag: 'a v' set variable $a "+
		"to JSON value <v>")
	flag.StringVar(&(yq.slurpfile), "slurpfile", "", "jq Flag: 'a f' set variable "+
		"$a to an array of JSON texts read from <f>")
	flag.StringVar(&(yq.rawfile), "rawfile", "", "set variable $a to a string "+
		"consisting of the contents of <f>")
	flag.StringVar(&(yq.args), "args", "", "remaining arguments are string "+
		"arguments, not files")
	flag.StringVar(&(yq.jsonargs), "jsonargs", "", "remaining arguments are JSON "+
		"arguments, not files")

	if len(os.Args) == 1 {
		flag.Usage()
		return errors.New("no arguments passed")
	}

	flag.Parse()
	return nil
}

func (yq *yq) compileJqCmd() error {
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
	if yq.slurp {
		yq.jqCmd.Args = append(yq.jqCmd.Args, "-s")
	}
	if yq.raw {
		yq.jqCmd.Args = append(yq.jqCmd.Args, "-r")
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
		yq.jqCmd.Args = append(yq.jqCmd.Args, "--arg")
		idx := 0
		for i, arg := range os.Args {
			if arg == "--arg" {
				idx = i
				break
			}
		}

		yq.jqCmd.Args = append(yq.jqCmd.Args, os.Args[idx+1:idx+3]...)
	}
	if yq.argjson != "" {
		yq.jqCmd.Args = append(yq.jqCmd.Args, "--argjson")
		skippedArgs = skippedArgs + 1
		idx := 0
		for i, arg := range os.Args {
			if arg == "--argjson" {
				idx = i
				break
			}
		}

		yq.jqCmd.Args = append(yq.jqCmd.Args, os.Args[idx+1:idx+3]...)
	}
	if yq.slurpfile != "" {
		yq.jqCmd.Args = append(yq.jqCmd.Args, "--slurpfile")
		skippedArgs = skippedArgs + 1
		idx := 0
		for i, arg := range os.Args {
			if arg == "--slurpfile" {
				idx = i
				break
			}
		}

		yq.jqCmd.Args = append(yq.jqCmd.Args, os.Args[idx+1:idx+3]...)
	}
	if yq.rawfile != "" {
		yq.jqCmd.Args = append(yq.jqCmd.Args, "--rawfile")
		skippedArgs = skippedArgs + 1
		idx := 0
		for i, arg := range os.Args {
			if arg == "--rawfile" {
				idx = i
				break
			}
		}

		yq.jqCmd.Args = append(yq.jqCmd.Args, os.Args[idx+1:idx+3]...)
	}
	if yq.jsonargs != "" {
		yq.jqCmd.Args = append(yq.jqCmd.Args, "--jsonargs")
		idx := 0
		for i, arg := range os.Args {
			if arg == "--argjson" {
				idx = i
				break
			}
		}

		skippedArgs = len(os.Args) - 1

		yq.jqCmd.Args = append(yq.jqCmd.Args, os.Args[idx+1:]...)
	}

	yq.jqCmd.Args = append(yq.jqCmd.Args, flag.Args()[skippedArgs-1])

	for _, arg := range flag.Args()[skippedArgs:] {
		if _, err := os.Stat(arg); err != nil {
			return err
		}

		yq.files = append(yq.files, arg)
	}

	var stdinPipe io.WriteCloser
	stdinPipe, err := yq.jqCmd.StdinPipe()
	if err != nil {
		return err
	}
	yq.jqStdin = stdinPipe

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
		yq.jqStdin.Close()
		return err
	}

	if len(yq.files) == 0 {
		err = transformToJSON(os.Stdin, func(b []byte) error {
			_, err = yq.jqStdin.Write(b)
			if err != nil {
				yq.jqStdin.Close()
			}
			return err

		})

		if err != nil {
			yq.jqStdin.Close()
			return err
		}
	} else {
		for _, file := range yq.files {
			file, _ := os.Open(file)
			if err != nil {
				yq.jqStdin.Close()
				return err
			}

			err = transformToJSON(file, func(b []byte) error {
				_, err = yq.jqStdin.Write(b)
				_, err = yq.jqStdin.Write([]byte("\r\n"))
				if err != nil {
					yq.jqStdin.Close()
				}
				return err
			})

			if err != nil {
				yq.jqStdin.Close()
				return err
			}
		}
	}

	yq.jqStdin.Close()

	if yq.returnYAML {
		err := transformToYAML(yq.jqStdout, os.Stdout)
		if err != nil {
			return err
		}
	}

	yq.jqCmd.Wait()

	return nil
}

func main() {
	y := yq{}

	if path, err := exec.LookPath("jq"); err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	} else {
		y.jqCmd.Path = path
	}

	if err := y.parseFlags(); err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	if err := y.compileJqCmd(); err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	if err := y.run(); err != nil {
		fmt.Println(err)
	}
}
