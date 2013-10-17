// Copyright 2013 Federico Sogaro. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package byteunit

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type Size int64

var units = []string{"byte", "kB", "MB", "GB", "TB"}

type Unit int

const (
	UnitByte Unit = iota
	UnitKB
	UnitMB
	UnitGB
	UnitTB
)

func (s Size) String() string {
	if s == 0 {
		return "0"
	}
	for i := len(units) - 1; i > 0; i-- {
		q := int64(1 << uint(10 * i))
		if int64(s) % q == 0 {
			return fmt.Sprintf("%d %s", int64(s)/q, units[i])
		}
	}
	if s == 1 {
		return "1 byte"
	} else {
		return fmt.Sprintf("%d bytes", s)
	}
}

func (s Size) Format(unit Unit) string {
	q := 2 << uint(10 * unit)
	return fmt.Sprintf("%.4g %s", float64(s)/float64(q), units[unit])
}

var re = regexp.MustCompile("^([0-9]*(?:[.][0-9]+)?) *([A-Za-z]+)$")

func Parse(str string) (Size, error) {
	str = strings.TrimSpace(str)
	if str == "0" {
		return 0, nil
	}
	submatches := re.FindStringSubmatch(str)
	if len(submatches) != 3 {
		return 0, fmt.Errorf("unable to parse string: %s", str)
	}
	value, err := strconv.ParseFloat(submatches[1], 64)
	if err != nil {
		panic("unreachable (bug in regexp!)")
	}
	switch submatches[2] {
	case "byte", "bytes", "byte(s)":
	case "Kb", "kb":
		value *= 1<<10
	case "MB", "mb":
		value *= 1<<20
	case "GB", "gb":
		value *= 1<<30
	case "TB", "tb":
		value *= 1<<40
	default:
		return 0, fmt.Errorf("invalid byteunit: %s", str)
	}
	return Size(value), nil
}

