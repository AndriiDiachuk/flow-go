package errors

import (
	stdErrors "errors"
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/errors"
)

type Unwrappable interface {
	error
	Unwrap() error
}

type UnwrappableErrors interface {
	error
	Unwrap() []error
}

type CodedError interface {
	Code() ErrorCode

	Unwrappable
	error
}

type CodedFailure interface {
	Code() FailureCode

	Unwrappable
	error
}

// Is is a utility function to call std error lib `Is` function for instance equality checks.
func Is(err error, target error) bool {
	return stdErrors.Is(err, target)
}

// As is a utility function to call std error lib `As` function.
// As finds the first error in err's chain that matches target,
// and if so, sets target to that error value and returns true. Otherwise, it returns false.
// The chain consists of err itself followed by the sequence of errors obtained by repeatedly calling Unwrap.
func As(err error, target interface{}) bool {
	return stdErrors.As(err, target)
}

// findRootCodedError recursively unwraps the error to search for the root (deepest) coded error:
//  1. If err is nil, this returns (nil, false),
//  2. If err has no error code, this returns (nil, true),
//  3. If err has an error code, this returns
//     (<the deepest, aka root cause, coded error>, false)
//
// Note: This assumes the caller has already checked if the error contains a CodedFailure.
func findRootCodedError(err error) (CodedError, bool) {
	if err == nil {
		return nil, false
	}

	var coded CodedError
	if !As(err, &coded) {
		return nil, true
	}

	for {
		var nextCoded CodedError
		if !As(coded.Unwrap(), &nextCoded) {
			return coded, false
		}

		coded = nextCoded
	}
}

// IsFailure returns true if the error is un-coded, or if the error contains
// a failure code.
func IsFailure(err error) bool {
	return AsFailure(err) != nil
}

func AsFailure(err error) CodedFailure {
	if err == nil {
		return nil
	}

	var failure CodedFailure
	if As(err, &failure) {
		return failure
	}

	var coded CodedError
	if !As(err, &coded) {
		return NewUnknownFailure(err)
	}

	return nil
}

// SplitErrorTypes splits the error into fatal (failures) and non-fatal errors
func SplitErrorTypes(inp error) (err CodedError, failure CodedFailure) {
	if inp == nil {
		return nil, nil
	}

	if failure = AsFailure(inp); failure != nil {
		return nil, WrapCodedFailure(
			failure.Code(),
			inp,
			"failure caused by")
	}

	coded, isUnknown := findRootCodedError(inp)
	if isUnknown {
		return nil, NewUnknownFailure(inp)
	}

	return WrapCodedError(
		coded.Code(),
		inp,
		"error caused by"), nil
}

// HandleRuntimeError handles runtime errors and separates
// errors generated by runtime from fvm errors (e.g. environment errors)
func HandleRuntimeError(err error) error {
	if err == nil {
		return nil
	}

	// if is not a runtime error return as vm error
	// this should never happen unless a bug in the code
	runErr, ok := err.(runtime.Error)
	if !ok {
		return NewUnknownFailure(err)
	}

	// All other errors are non-fatal Cadence errors.
	return NewCadenceRuntimeError(runErr)
}

// HasErrorCode returns true if the error or one of its nested errors matches the
// specified error code.
func HasErrorCode(err error, code ErrorCode) bool {
	return Find(err, code) != nil
}

// HasFailureCode returns true if the error or one of its nested errors matches the
// specified failure code.
func HasFailureCode(err error, code FailureCode) bool {
	return FindFailure(err, code) != nil
}

// Find recursively unwraps the error and returns the first CodedError that matches
// the specified error code.
func Find(originalErr error, code ErrorCode) CodedError {
	if originalErr == nil {
		return nil
	}

	// Handle non-chained errors
	var unwrappedErrs []error
	switch err := originalErr.(type) {
	case *multierror.Error:
		unwrappedErrs = err.WrappedErrors()
	case UnwrappableErrors:
		unwrappedErrs = err.Unwrap()

	// IMPORTANT: this check needs to run after *multierror.Error because multierror does implement
	// the Unwrappable interface, however its implementation only visits the base errors in the list,
	// and ignores their descendants.
	case Unwrappable:
		coded, ok := err.(CodedError)
		if ok && coded.Code() == code {
			return coded
		}
		return Find(err.Unwrap(), code)
	default:
		return nil
	}

	for _, innerErr := range unwrappedErrs {
		coded := Find(innerErr, code)
		if coded != nil {
			return coded
		}
	}

	return nil
}

// FindFailure recursively unwraps the error and returns the first CodedFailure that matches
// the specified error code.
func FindFailure(originalErr error, code FailureCode) CodedFailure {
	if originalErr == nil {
		return nil
	}

	// Handle non-chained errors
	var unwrappedErrs []error
	switch err := originalErr.(type) {
	case *multierror.Error:
		unwrappedErrs = err.WrappedErrors()
	case UnwrappableErrors:
		unwrappedErrs = err.Unwrap()

	// IMPORTANT: this check needs to run after *multierror.Error because multierror does implement
	// the Unwrappable interface, however its implementation only visits the base errors in the list,
	// and ignores their descendants.
	case Unwrappable:
		coded, ok := err.(CodedFailure)
		if ok && coded.Code() == code {
			return coded
		}
		return FindFailure(err.Unwrap(), code)
	default:
		return nil
	}

	for _, innerErr := range unwrappedErrs {
		coded := FindFailure(innerErr, code)
		if coded != nil {
			return coded
		}
	}

	return nil
}

var _ CodedError = (*codedError)(nil)

type codedError struct {
	code ErrorCode

	err error
}

func newError(
	code ErrorCode,
	rootCause error,
) codedError {
	return codedError{
		code: code,
		err:  rootCause,
	}
}

func WrapCodedError(
	code ErrorCode,
	err error,
	prefixMsgFormat string,
	formatArguments ...interface{},
) codedError {
	if prefixMsgFormat != "" {
		msg := fmt.Sprintf(prefixMsgFormat, formatArguments...)
		err = fmt.Errorf("%s: %w", msg, err)
	}
	return newError(code, err)
}

func NewCodedError(
	code ErrorCode,
	format string,
	formatArguments ...interface{},
) codedError {
	return newError(code, fmt.Errorf(format, formatArguments...))
}

func (err codedError) Unwrap() error {
	return err.err
}

func (err codedError) Error() string {
	return fmt.Sprintf("%v %v", err.code, err.err)
}

func (err codedError) Code() ErrorCode {
	return err.code
}

var _ CodedFailure = (*codedFailure)(nil)

type codedFailure struct {
	code FailureCode
	err  error
}

func newFailure(
	code FailureCode,
	rootCause error,
) codedFailure {
	return codedFailure{
		code: code,
		err:  rootCause,
	}
}

func WrapCodedFailure(
	code FailureCode,
	err error,
	prefixMsgFormat string,
	formatArguments ...interface{},
) codedFailure {
	if prefixMsgFormat != "" {
		msg := fmt.Sprintf(prefixMsgFormat, formatArguments...)
		err = fmt.Errorf("%s: %w", msg, err)
	}
	return newFailure(code, err)
}

func NewCodedFailure(
	code FailureCode,
	format string,
	formatArguments ...interface{},
) codedFailure {
	return newFailure(code, fmt.Errorf(format, formatArguments...))
}

func (err codedFailure) Unwrap() error {
	return err.err
}

func (err codedFailure) Error() string {
	return fmt.Sprintf("%v %v", err.code, err.err)
}

func (err codedFailure) Code() FailureCode {
	return err.code
}

// NewEventEncodingError construct a new CodedError which indicates
// that encoding event has failed
func NewEventEncodingError(err error) CodedError {
	return NewCodedError(
		ErrCodeEventEncodingError,
		"error while encoding emitted event: %w ", err)
}

// EVMError needs to satisfy the user error interface
// in order for Cadence to correctly handle the error
var _ errors.UserError = &(EVMError{})

type EVMError struct {
	CodedError
}

func (e EVMError) IsUserError() {}

// NewEVMError constructs a new CodedError which captures a
// collection of errors provided by (non-fatal) evm runtime.
func NewEVMError(err error) EVMError {
	return EVMError{
		WrapCodedError(
			ErrEVMExecutionError,
			err,
			"evm runtime error"),
	}
}

// IsEVMError returns true if error is an EVM error
func IsEVMError(err error) bool {
	return HasErrorCode(err, ErrEVMExecutionError)
}

// IsAnFVMError returns true if error is a coded error or a failure
func IsFVMError(err error) bool {
	var coded CodedError
	if As(err, &coded) {
		return true
	}
	return IsFailure(err)
}
