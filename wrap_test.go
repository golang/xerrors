// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xerrors_test

import (
	"fmt"
	"os"
	"testing"

	"golang.org/x/xerrors"
)

func TestIs(t *testing.T) {
	err1 := xerrors.New("1")
	erra := xerrors.Errorf("wrap 2: %w", err1)
	errb := xerrors.Errorf("wrap 3: %w", erra)
	erro := xerrors.Opaque(err1)
	errco := xerrors.Errorf("opaque: %w", erro)
	err3 := xerrors.New("3")

	poser := &poser{"either 1 or 3", func(err error) bool {
		return err == err1 || err == err3
	}}

	testCases := []struct {
		err    error
		target error
		match  bool
	}{
		{nil, nil, true},
		{err1, nil, false},
		{err1, err1, true},
		{erra, err1, true},
		{errb, err1, true},
		{errco, erro, true},
		{errco, err1, false},
		{erro, erro, true},
		{err1, err3, false},
		{erra, err3, false},
		{errb, err3, false},
		{poser, err1, true},
		{poser, err3, true},
		{poser, erra, false},
		{poser, errb, false},
		{poser, erro, false},
		{poser, errco, false},
	}
	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			if got := xerrors.Is(tc.err, tc.target); got != tc.match {
				t.Errorf("Is(%v, %v) = %v, want %v", tc.err, tc.target, got, tc.match)
			}
		})
	}
}

func TestIsOneOf(t *testing.T) {
	err1 := xerrors.New("1")
	erra := xerrors.Errorf("wrap 2: %w", err1)
	erro := xerrors.Opaque(err1)

	poser := &poser{"either 1 or a", func(err error) bool {
		return err == err1 || err == erra
	}}

	testCases := []struct {
		errs   []error
		target error
		match  bool
	}{
		{[]error{nil, err1}, nil, true},
		{[]error{err1, erro}, nil, false},
		{[]error{erra, erro, err1, nil}, err1, true},
		{[]error{poser, erra}, err1, true},
	}
	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			if got := xerrors.IsOneOf(tc.errs, tc.target); got != tc.match {
				t.Errorf("Is(%v, %v) = %v, want %v", tc.errs, tc.target, got, tc.match)
			}
		})
	}
}

type poser struct {
	msg string
	f   func(error) bool
}

func (p *poser) Error() string     { return p.msg }
func (p *poser) Is(err error) bool { return p.f(err) }
func (p *poser) As(err interface{}) bool {
	switch x := err.(type) {
	case **poser:
		*x = p
	case *errorT:
		*x = errorT{}
	case **os.PathError:
		*x = &os.PathError{}
	default:
		return false
	}
	return true
}

func TestAs(t *testing.T) {
	var errT errorT
	var errP *os.PathError
	var timeout interface{ Timeout() bool }
	var p *poser
	_, errF := os.Open("non-existing")

	testCases := []struct {
		err    error
		target interface{}
		match  bool
	}{{
		xerrors.Errorf("pittied the fool: %w", errorT{}),
		&errT,
		true,
	}, {
		errF,
		&errP,
		true,
	}, {
		xerrors.Opaque(errT),
		&errT,
		false,
	}, {
		errorT{},
		&errP,
		false,
	}, {
		errWrap{nil},
		&errT,
		false,
	}, {
		&poser{"error", nil},
		&errT,
		true,
	}, {
		&poser{"path", nil},
		&errP,
		true,
	}, {
		&poser{"oh no", nil},
		&p,
		true,
	}, {
		xerrors.New("err"),
		&timeout,
		false,
	}, {
		errF,
		&timeout,
		true,
	}, {
		xerrors.Errorf("path error: %w", errF),
		&timeout,
		true,
	}}
	for i, tc := range testCases {
		name := fmt.Sprintf("%d:As(Errorf(..., %v), %v)", i, tc.err, tc.target)
		t.Run(name, func(t *testing.T) {
			match := xerrors.As(tc.err, tc.target)
			if match != tc.match {
				t.Fatalf("xerrors.As(%T, %T): got %v; want %v", tc.err, tc.target, match, tc.match)
			}
			if !match {
				return
			}
			if tc.target == nil {
				t.Fatalf("non-nil result after match")
			}
		})
	}
}

func TestUnwrap(t *testing.T) {
	err1 := xerrors.New("1")
	erra := xerrors.Errorf("wrap 2: %w", err1)
	erro := xerrors.Opaque(err1)

	testCases := []struct {
		err  error
		want error
	}{
		{nil, nil},
		{errWrap{nil}, nil},
		{err1, nil},
		{erra, err1},
		{xerrors.Errorf("wrap 3: %w", erra), erra},

		{erro, nil},
		{xerrors.Errorf("opaque: %w", erro), erro},
	}
	for _, tc := range testCases {
		if got := xerrors.Unwrap(tc.err); got != tc.want {
			t.Errorf("Unwrap(%v) = %v, want %v", tc.err, got, tc.want)
		}
	}
}

func TestOpaque(t *testing.T) {
	got := fmt.Sprintf("%v", xerrors.Errorf("foo: %v", xerrors.Opaque(errorT{})))
	want := "foo: errorT"
	if got != want {
		t.Errorf("error without Format: got %v; want %v", got, want)
	}

	got = fmt.Sprintf("%v", xerrors.Errorf("foo: %v", xerrors.Opaque(errorD{})))
	want = "foo: errorD"
	if got != want {
		t.Errorf("error with Format: got %v; want %v", got, want)
	}
}

type errorT struct{}

func (errorT) Error() string { return "errorT" }

type errorD struct{}

func (errorD) Error() string { return "errorD" }

func (errorD) FormatError(p xerrors.Printer) error {
	p.Print("errorD")
	p.Detail()
	p.Print("detail")
	return nil
}

type errWrap struct{ error }

func (errWrap) Error() string { return "wrapped" }

func (errWrap) Unwrap() error { return nil }
