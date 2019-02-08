// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xerrors

import (
	"reflect"
)

// A Wrapper provides context around another error.
type Wrapper interface {
	// Unwrap returns the next error in the error chain.
	// If there is no next error, Unwrap returns nil.
	Unwrap() error
}

// Opaque returns an error with the same error formatting as err
// but that does not match err and cannot be unwrapped.
func Opaque(err error) error {
	return noWrapper{err}
}

type noWrapper struct {
	error
}

func (e noWrapper) FormatError(p Printer) (next error) {
	if f, ok := e.error.(Formatter); ok {
		return f.FormatError(p)
	}
	p.Print(e.error)
	return nil
}

// Unwrap returns the result of calling the Unwrap method on err, if err implements
// Unwrap. Otherwise, Unwrap returns nil.
func Unwrap(err error) error {
	u, ok := err.(Wrapper)
	if !ok {
		return nil
	}
	return u.Unwrap()
}

// Is reports whether any error in err's chain matches target.
//
// An error is considered to match a target if it is equal to that target or if
// it implements a method Is(error) bool such that Is(target) returns true.
func Is(err, target error) bool {
	if target == nil {
		return err == target
	}
	for {
		if err == target {
			return true
		}
		if x, ok := err.(interface{ Is(error) bool }); ok && x.Is(target) {
			return true
		}
		// TODO: consider supporing target.Is(err). This would allow
		// user-definable predicates, but also may allow for coping with sloppy
		// APIs, thereby making it easier to get away with them.
		if err = Unwrap(err); err == nil {
			return false
		}
	}
}

// IsOneOf reports whether any error in the provided slice of errors matches the target.
// IsOneOf looks at the entire error chain for each error and compares against the provided target.
//
// An error is considered to match a target if it is equal to that target or if
// it implements a method Is(error) bool such that Is(target) returns true.
func IsOneOf(errs []error, target error) bool {
	isOneOf := false

	for _, err := range errs {
		if Is(err, target) {
			isOneOf = true
			break
		}
	}

	return isOneOf
}

// As finds the first error in err's chain that matches the type to which target
// points, and if so, sets the target to its value and returns true. An error
// matches a type if it is assignable to the target type, or if it has a method
// As(interface{}) bool such that As(target) returns true. As will panic if target
// is nil or not a pointer.
//
// The As method should set the target to its value and return true if err
// matches the type to which target points.
func As(err error, target interface{}) bool {
	if target == nil {
		panic("errors: target cannot be nil")
	}
	typ := reflect.TypeOf(target)
	if typ.Kind() != reflect.Ptr {
		panic("errors: target must be a pointer")
	}
	targetType := typ.Elem()
	for {
		if reflect.TypeOf(err).AssignableTo(targetType) {
			reflect.ValueOf(target).Elem().Set(reflect.ValueOf(err))
			return true
		}
		if x, ok := err.(interface{ As(interface{}) bool }); ok && x.As(target) {
			return true
		}
		if err = Unwrap(err); err == nil {
			return false
		}
	}
}
