Package templateutils provides a set of functions that are designed to
make it easier for developers to add template based scripting to their
command line tools.

Command line tools written in Go often allow users to specify a template
script to tailor the output of the tool to their specific needs. This can be
useful both when visually inspecting the data and also when invoking command
line tools in scripts. The best example of this is go list which allows users
to pass a template script to extract interesting information about Go
packages. For example,

```
go list -f '{{range .Imports}}{{println .}}{{end}}'
```

prints all the imports of the current package.

The aim of this package is to make it easier for developers to add template
scripting support to their tools and easier for users of these tools to
extract the information they need.   It does this by augmenting the
templating language provided by the standard library package text/template in
two ways:

1. It auto generates descriptions of the data structures passed as
input to a template script for use in help messages.  This ensures
that help usage information is always up to date with the source code.

2. It provides a suite of convenience functions to make it easy for
script writers to extract the data they need.  There are functions for
sorting, selecting rows and columns and generating nicely formatted
tables.

For example, if a program passed a slice of structs containing stock
data to a template script, we could use the following script to extract
the names of the 3 stocks with the highest trade volume.

```
{{table (cols (head (sort . "Volume" "dsc") 3) "Name" "Volume")}}
```

The output might look something like this:

```
Name              Volume
Happy Enterprises 6395624278
Big Company       7500000
Medium Company    300122
```

The functions head, sort, tables and col are provided by this package.
