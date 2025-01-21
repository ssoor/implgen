// Copyright 2010 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// MockGen generates mock implementations of Go interfaces.
package main

// TODO: This does not support recursive embedded interfaces.
// TODO: This does not support embedding package-local interfaces in a separate file.

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/build"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/ssoor/implgen/model"
)

const (
	gomockImportPath = "github.com/golang/mock/gomock"
)

var (
	version = ""
	commit  = "none"
	date    = "unknown"
)

var (
	source          = flag.String("source", "", "接口定义文件/源文件，工具根据源文件生成输出结果")
	destination     = flag.String("destination", "", "指定输出文件路径，默认将内容输出到控制台")
	implNames       = flag.String("impl_names", "", "传参为逗号分隔的 `intefaceName=implementName` 对，用来指定接口生成的结构名。默认名会根据 `interfaceName `生成，如果 `interfaceName` 后缀为 `Interface` 则删除 `Interface` 后缀后作为名称，如果没有 `Interface` 后缀就直接使用 `interfaceName`")
	implInterfaces  = flag.String("impl_interfaces", "", "传参为逗号分隔的接口名")
	packageOut      = flag.String("package", "", "代码生成的包名（package <包名>）")
	selfPackage     = flag.String("self_package", "", "The full package import path for the generated code. The purpose of this flag is to prevent import cycles in the generated code by trying to include its own package. This can happen if the mock's package is set to one of its inputs (usually the main one) and the output is stdio so mockgen cannot detect the final output package. Setting this flag will then tell mockgen which import to exclude.")
	writePkgComment = flag.Bool("write_package_comment", false, "Writes package documentation comment (godoc) if true.")
	copyrightFile   = flag.String("copyright_file", "", "Copyright file used to add copyright header")

	debugParser = flag.Bool("debug_parser", false, "仅打印解析器解析结果")
	showVersion = flag.Bool("version", false, "Print version.")
)

func main() {
	flag.Usage = usage
	flag.Parse()

	if *showVersion {
		printVersion()
		return
	}

	var pkg *model.Package
	var err error
	var packageName string
	if *source != "" {
		pkg, err = sourceMode(*source)
	} else {
		if flag.NArg() != 2 {
			usage()
			log.Fatal("Expected exactly two arguments")
		}
		packageName = flag.Arg(0)
		if packageName == "." {
			dir, err := os.Getwd()
			if err != nil {
				log.Fatalf("Get current directory failed: %v", err)
			}
			packageName, err = packageNameOfDir(dir)
			if err != nil {
				log.Fatalf("Parse package name failed: %v", err)
			}
		}
		pkg, err = reflectMode(packageName, strings.Split(flag.Arg(1), ","))
	}
	if err != nil {
		log.Fatalf("Loading input failed: %v", err)
	}

	if *debugParser {
		pkg.Print(os.Stdout)
		return
	}

	outputPackageName := *packageOut
	if outputPackageName == "" {
		// pkg.Name in reflect mode is the base name of the import path,
		// which might have characters that are illegal to have in package names.
		outputPackageName = "impl_" + sanitize(pkg.Name)
	}

	// outputPackagePath represents the fully qualified name of the package of
	// the generated code. Its purposes are to prevent the module from importing
	// itself and to prevent qualifying type names that come from its own
	// package (i.e. if there is a type called X then we want to print "X" not
	// "package.X" since "package" is this package). This can happen if the mock
	// is output into an already existing package.
	outputPackagePath := *selfPackage
	if len(outputPackagePath) == 0 && len(*destination) > 0 {
		dst, _ := filepath.Abs(filepath.Dir(*destination))
		for _, prefix := range build.Default.SrcDirs() {
			if strings.HasPrefix(dst, prefix) {
				if rel, err := filepath.Rel(prefix, dst); err == nil {
					outputPackagePath = rel
					break
				}
			}
		}
	}

	g := &generator{
		mockNames:      make(map[string]string),
		mockInterfaces: make(map[string]bool),
	}
	if *destination != "" {
		g.dstFileName = *destination
	}
	if *source != "" {
		g.filename = *source
	} else {
		g.srcPackage = packageName
		g.srcInterfaces = flag.Arg(1)
	}

	if *implNames != "" {
		g.mockNames = parseMockNames(*implNames)
	}
	if *implInterfaces != "" {
		for _, v := range strings.Split(*implInterfaces, ",") {
			v := strings.TrimSpace(v)
			g.mockInterfaces[v] = true
		}
	}
	if *copyrightFile != "" {
		header, err := ioutil.ReadFile(*copyrightFile)
		if err != nil {
			log.Fatalf("Failed reading copyright file: %v", err)
		}

		g.copyrightHeader = string(header)
	}
	if err := g.Generate(pkg, outputPackageName, outputPackagePath); err != nil {
		log.Fatalf("Failed generating mock: %v", err)
	}

	if _, err := g.Output(); err != nil {
		log.Fatalf("Failed writing to destination: %v", err)
	}
}

func parseMockNames(names string) map[string]string {
	mocksMap := make(map[string]string)
	for _, kv := range strings.Split(names, ",") {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 || parts[1] == "" {
			log.Fatalf("bad mock names spec: %v", kv)
		}
		mocksMap[parts[0]] = parts[1]
	}
	return mocksMap
}

func usage() {
	_, _ = io.WriteString(os.Stderr, usageText)
	flag.PrintDefaults()
}

const usageText = `mockgen has two modes of operation: source and reflect.

Source mode generates mock interfaces from a source file.
It is enabled by using the -source flag. Other flags that
may be useful in this mode are -imports and -aux_files.
Example:
	mockgen -source=foo.go [other options]

Reflect mode generates mock interfaces by building a program
that uses reflection to understand interfaces. It is enabled
by passing two non-flag arguments: an import path, and a
comma-separated list of symbols.
Example:
	mockgen database/sql/driver Conn,Driver

`

func removeDot(s string) string {
	if len(s) > 0 && s[len(s)-1] == '.' {
		return s[0 : len(s)-1]
	}
	return s
}

// sanitize cleans up a string to make a suitable package name.
func sanitize(s string) string {
	t := ""
	for _, r := range s {
		if t == "" {
			if unicode.IsLetter(r) || r == '_' {
				t += string(r)
				continue
			}
		} else {
			if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
				t += string(r)
				continue
			}
		}
		t += "_"
	}
	if t == "_" {
		t = "x"
	}
	return t
}

func makeArgString(argNames, argTypes []string) string {
	args := make([]string, len(argNames))
	for i, name := range argNames {
		// specify the type only once for consecutive args of the same type
		if i+1 < len(argTypes) && argTypes[i] == argTypes[i+1] {
			args[i] = name
		} else {
			args[i] = name + " " + argTypes[i]
		}
	}
	return strings.Join(args, ", ")
}

// createPackageMap returns a map of import path to package name
// for specified importPaths.
func createPackageMap(importPaths []string) map[string]string {
	var pkg struct {
		Name       string
		ImportPath string
	}
	pkgMap := make(map[string]string)
	b := bytes.NewBuffer(nil)
	args := []string{"list", "-json"}
	args = append(args, importPaths...)
	cmd := exec.Command("go", args...)
	cmd.Stdout = b
	cmd.Run()
	dec := json.NewDecoder(b)
	for dec.More() {
		err := dec.Decode(&pkg)
		if err != nil {
			log.Printf("failed to decode 'go list' output: %v", err)
			continue
		}
		pkgMap[pkg.ImportPath] = pkg.Name
	}
	return pkgMap
}

func printVersion() {
	if version != "" {
		fmt.Printf("v%s\nCommit: %s\nDate: %s\n", version, commit, date)
	} else {
		printModuleVersion()
	}
}
