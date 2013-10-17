// Copyright 2013 Federico Sogaro. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hwinfo

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

//remove brackets "(...)" and multiple spaces
func reformat(str string) string {
	for k := 0; k < 100; k++ {
		if i := strings.Index(str, "("); i != -1 {
			if j := strings.Index(str, ")"); j != -1 {
				str = strings.TrimSpace(str[:i] + str[j+1:])
			}
		} else {
			break
		}
	}
	return strings.Join(strings.Fields(str), " ")
}

func getKeyValue(line, sep string) (key, value string) {
	line = strings.TrimSpace(line)
	kv := strings.SplitN(line, ":", 2)
	for j := 0; j < len(kv); j++ {
		kv[j] = reformat(kv[j])
	}
	if len(kv) != 2 {
		return "", ""
	}
	return strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])
}

type exitStatus struct {
	Stdout []byte
	Stderr []byte
	Err error
}

func (es exitStatus) code() int {
	if es.Err == nil {
		return 0
	}
	if ec, ok := es.Err.(*exec.ExitError); ok {
		return ec.Sys().(syscall.WaitStatus).ExitStatus()
	}
	return -1
}

func (es exitStatus) lines() []string {
	var lines []string
	i := 0
	for {
		advance, line, _ := bufio.ScanLines(es.Stdout[i:], true)
		if advance == 0 {
			break
		}
		lines = append(lines, string(line))
		i += advance
	}
	return lines
}

func run(name string, args ...string) *exitStatus {
	cmd := exec.Command(name, args...)
	var bufout, buferr bytes.Buffer
	cmd.Stdout = &bufout
	cmd.Stderr = &buferr
	err := cmd.Run()
	return &exitStatus{bufout.Bytes(), buferr.Bytes(), err}
}

type CmdError struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
	Err      error
}

func (e CmdError) Error() string {
	return fmt.Sprintf("(%d): %s", e.ExitCode, e.Err)
}

func newCmdError(es *exitStatus) CmdError {
	return CmdError{es.Stdout, es.Stderr, es.code(), es.Err}
}

func read(filename string) *exitStatus {
	buf, err := ioutil.ReadFile(filename)
	return &exitStatus{buf, nil, err}
}

func strToInt(str string) (int, error) {
	val, err := strconv.Atoi(strings.TrimSpace(str))
	if err != nil {
		return 0, err
	}
	return val, nil
}

func strToFloat(str string) (float64, error) {
	val, err := strconv.ParseFloat(strings.TrimSpace(str), 64)
	if err != nil {
		return 0.0, err
	}
	return val, nil
}
