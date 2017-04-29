// Copyright (c) 2017 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package main

import (
	"crypto/sha1"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This implements a test that verifies that the code in interna/tests/ is up to
// date.

const _testPackage = "go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests"

func serve() string {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				panic(err)
			}

			handle(conn)
		}
	}()

	return ln.Addr().String()
}

func handle(conn net.Conn) {
	defer conn.Close()

	stdinR, stdinW, err := os.Pipe()
	if err != nil {
		panic(err)
	}

	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		panic(err)
	}

	go io.Copy(conn, stdoutR)
	go io.Copy(stdinW, conn)

	os.Stdout = stdoutW
	os.Stdin = stdinR
	main()
}

func callback(addr string) string {
	i := strings.LastIndexByte(addr, ':')
	host := addr[:i]
	port, err := strconv.ParseInt(addr[i+1:], 10, 32)
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf(`#!/bin/bash -e

nc %v %v
`, host, port)
}

func TestCodeIsUpToDate(t *testing.T) {
	{
		outputDir, err := ioutil.TempDir("", "current-thriftrw-plugin-yarpc")
		if err != nil {
			log.Fatalf("failed to create temporary directory: %v", err)
		}
		defer os.RemoveAll(outputDir)

		path := os.Getenv("PATH")
		if err := os.Setenv("PATH", fmt.Sprintf("%v:%v", outputDir, path)); err != nil {
			log.Fatalf("failed to add %q to PATH: %v", outputDir, err)
		}

		err = ioutil.WriteFile(
			filepath.Join(outputDir, "thriftrw-plugin-yarpc"),
			[]byte(callback(serve())),
			0777,
		)
		require.NoError(t, err, "failed to create thriftrw plugin script")
	}

	thriftRoot, err := filepath.Abs("internal/tests")
	require.NoError(t, err, "could not resolve absolute path to internal/tests")

	thriftFiles, err := filepath.Glob(thriftRoot + "/*.thrift")
	require.NoError(t, err)

	outputDir, err := ioutil.TempDir("", "golden-test")
	require.NoError(t, err, "failed to create temporary directory")
	defer os.RemoveAll(outputDir)

	for _, thriftFile := range thriftFiles {
		packageName := strings.TrimSuffix(filepath.Base(thriftFile), ".thrift")
		currentPackageDir := filepath.Join("internal/tests", packageName)
		newPackageDir := filepath.Join(outputDir, packageName)

		currentHash, err := dirhash(currentPackageDir)
		require.NoError(t, err, "could not hash %q", currentPackageDir)

		err = thriftrw(
			"--no-recurse",
			"--out", outputDir,
			"--pkg-prefix", _testPackage,
			"--thrift-root", thriftRoot,
			"--plugin", "yarpc",
			thriftFile,
		)
		require.NoError(t, err, "failed to generate code for %q", thriftFile)

		newHash, err := dirhash(newPackageDir)
		require.NoError(t, err, "could not hash %q", newPackageDir)

		assert.Equal(t, currentHash, newHash,
			"Generated code for %q is out of date.", thriftFile)
	}
}

func thriftrw(args ...string) error {
	cmd := exec.Command("thriftrw", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func dirhash(dir string) (map[string]string, error) {
	fileHashes := make(map[string]string)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		fileHash, err := hash(path)
		if err != nil {
			return fmt.Errorf("failed to hash %q: %v", path, err)
		}

		// We only care about the path relative to the directory being
		// hashed.
		path, err = filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		fileHashes[path] = fileHash
		return nil
	})

	return fileHashes, err
}

func hash(name string) (string, error) {
	f, err := os.Open(name)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
