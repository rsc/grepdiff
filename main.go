// Copyright 2015 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Grepdiff greps diffs.
//
// Usage:
//
//	grepdiff regexp [file.diff ...]
//
// Grepdiff reads diffs from the named files (or standard input)
// and prints a reduced diff containing only the hunks matching
// the regular expression.
//
// The regular expression syntax is that of the Go regexp package,
// which matches RE2 and roughly matches PCRE, Perl, and most
// other languages.
// For details, see ``go doc regexp/syntax'' or https://godoc.org/regexp/syntax.
//
// The diffs are expected to be in roughly unified diff format,
// as produced by commands like ``git diff'' or ``hg diff''.
//
// Grepdiff exits with status 0 if it found any matches, 1 if it found no matches, and 2 if an error occurred.
//
// Example
//
// To diff two Git revisions and extract just the hunks mentioning foo:
//
//	git diff rev1 rev2 | grepdiff foo
//
// If you are feeling particularly adventurous, to stage those hunks:
//
//	git diff rev1 rev2 | grepdiff foo | EDITOR='bash -c "cat >$1"' git add -e
//
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: grepdiff regexp [file.diff ...]\n")
	// fmt.Fprintf(os.Stderr, "options:\n")
	// flag.PrintDefaults()
	os.Exit(2)
}

var exitStatus = 1

func main() {
	log.SetFlags(0)
	log.SetPrefix("grepdiff: ")
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() < 1 {
		usage()
	}

	re, err := regexp.Compile(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}

	files := flag.Args()[1:]
	if len(files) == 0 {
		grepDiff(re, os.Stdin)
	} else {
		for _, file := range files {
			f, err := os.Open(file)
			if err != nil {
				log.Print(err)
				exitStatus = 2
				continue
			}
			grepDiff(re, f)
			f.Close()
		}
	}

	os.Exit(exitStatus)
}

func grepDiff(re *regexp.Regexp, file *os.File) {
	data, err := ioutil.ReadAll(file)
	grepDiffData(re, data, file.Name())
	if err != nil {
		log.Print("%v", err)
		exitStatus = 2
	}
}

var (
	diffLine = []byte("\ndiff ")
	hunkLine = []byte("\n@@ ")
)

func grepDiffData(re *regexp.Regexp, data []byte, name string) {
	forEach(data, "diff ", func(diff []byte) {
		var hdr []byte
		forEach(diff, "@@ ", func(hunk []byte) {
			if hdr == nil {
				hdr = diff[:cap(diff)-cap(hunk)]
			}
			if re.Match(hunk) {
				os.Stdout.Write(hdr)
				hdr = hdr[:0] // not nil
				os.Stdout.Write(hunk)
				if exitStatus == 1 {
					exitStatus = 0
				}
			}
		})
	})
}

func forEach(x []byte, prefix string, f func([]byte)) {
	b := []byte("\n" + prefix)
	start := 0
	if !bytes.HasPrefix(x, b[1:]) {
		start = bytes.Index(x, b) + 1
		if start == 0 {
			return
		}
	}
	for start < len(x) {
		i := bytes.Index(x[start:], b) + start + 1
		if i == start {
			i = len(x)
		}
		if start >= 0 {
			f(x[start:i])
		}
		start = i
	}
}
