ImplGen
======

ImoGen is a interface implementation code generation tool for the [Go programming language][golang].

Installation
------------

Once you have [installed Go][golang-install], install the `implgen` tool.

To get the latest released version use:

```bash
GO111MODULE=on go get github.com/ssoor/implgen@latest
```

If you use `implgen` in your CI pipeline, it may be more appropriate to fixate
on a specific implgen version.

Building Implemented Code
--------------

input code: 

```go
// Doc:Test Foo
type Foo interface {
  // Doc:Foo.Bar
  Bar(x int) int // Comment:Foo.Bar
}

```

output code:

```go

// Doc:Test Foo
type Foo struct {
}

// Doc:Test Foo
func NewFoo() *Foo {
    implObj := &Foo{}

    // TODO: NewFoo() Not implemented

    return implObj
}

// Doc:Foo.Bar
func (m *Foo) Bar(x int) int {  // Comment:Foo.Bar
    // TODO: Foo.Bar(x int) int Not implemented

    panic("Foo.Bar(x int) int Not implemented")
}
```


Running implgen
---------------

`implgen` has two modes of operation: source and reflect.
Source mode generates mock interfaces from a source file.
It is enabled by using the -source flag. Other flags that
may be useful in this mode are -imports and -aux_files.

Example:

```bash
implgen -source=foo.go [other options]
```

Reflect mode generates mock interfaces by building a program
that uses reflection to understand interfaces. It is enabled
by passing two non-flag arguments: an import path, and a
comma-separated list of symbols.

You can use "." to refer to the current path's package.

Example:

```bash
implgen database/sql/driver Conn,Driver

# Convenient for `go:generate`.
implgen . Conn,Driver
```

The `implgen` command is used to generate source code for a implement
class given a Go source file containing interfaces to be implemented.
It supports the following flags:

* `-source`: A file containing interfaces to be implemented.

* `-destination`: A file to which to write the resulting source code. If you
    don't set this, the code is printed to standard output.

* `-package`: The package to use for the resulting implement class
    source code. If you don't set this, the package name is `mock_` concatenated
    with the package of the input file.

* `-imports`: A list of explicit imports that should be used in the resulting
    source code, specified as a comma-separated list of elements of the form
    `foo=bar/baz`, where `bar/baz` is the package being imported and `foo` is
    the identifier to use for the package in the generated source code.

* `-aux_files`: A list of additional files that should be consulted to
    resolve e.g. embedded interfaces defined in a different file. This is
    specified as a comma-separated list of elements of the form
    `foo=bar/baz.go`, where `bar/baz.go` is the source file and `foo` is the
    package name of that file used by the -source file.

* `-build_flags`: (reflect mode only) Flags passed verbatim to `go build`.

* `-mock_names`: A list of custom names for generated implements. This is specified
    as a comma-separated list of elements of the form
    `Repository=MockSensorRepository,Endpoint=MockSensorEndpoint`, where
    `Repository` is the interface name and `MockSensorRepository` is the desired
    implement name (implement factory method and implement recorder will be named after the implement).
    If one of the interfaces has no custom name specified, then default naming
    convention will be used.
    
* `-self_package`: The full package import path for the generated code. The purpose 
    of this flag is to prevent import cycles in the generated code by trying to include 
    its own package. This can happen if the implement's package is set to one of its 
    inputs (usually the main one) and the output is stdio so implgen cannot detect the 
    final output package. Setting this flag will then tell implgen which import to exclude.

* `-copyright_file`: Copyright file used to add copyright header to the resulting source code.

For an example of the use of `implgen`, see the `sample/` directory. In simple
cases, you will need only the `-source` flag.


[golang]:          http://golang.org/
[golang-install]:  http://golang.org/doc/install.html#releases
[gomock-ref]:      http://godoc.org/github.com/golang/mock/gomock
[travis-ci-badge]: https://travis-ci.org/golang/mock.svg?branch=master
[travis-ci]:       https://travis-ci.org/golang/mock
[godoc-badge]:     https://godoc.org/github.com/golang/mock/gomock?status.svg
[godoc]:           https://godoc.org/github.com/golang/mock/gomock
