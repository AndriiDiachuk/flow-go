package errors

import (
	stdErrors "errors"
	"fmt"

	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/errors"
)

type CodedError interface {
	Code() ErrorCode

	Unwrap() error

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

// findImportantCodedError recursively unwraps the error to search for important
// coded error:
//  1. If err is nil, this returns (nil, false),
//  2. If err has no error code, this returns (nil, true),
//  3. If err has a failure error code, this returns
//     (<the shallowest failure coded error>, false),
//  4. If err has a non-failure error code, this returns
//     (<the deepest, aka root cause, non-failure coded error>, false)
func findImportantCodedError(err error) (CodedError, bool) {
	if err == nil {
		return nil, false
	}

	var coded CodedError
	if !As(err, &coded) {
		return nil, true
	}

	for {
		if coded.Code().IsFailure() {
			return coded, false
		}

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
	if err == nil {
		return false
	}

	coded, isUnknown := findImportantCodedError(err)
	return isUnknown || coded.Code().IsFailure()
}

// SplitErrorTypes splits the error into fatal (failures) and non-fatal errors
func SplitErrorTypes(inp error) (err CodedError, failure CodedError) {
	if inp == nil {
		return nil, nil
	}

	coded, isUnknown := findImportantCodedError(inp)
	if isUnknown {
		return nil, NewUnknownFailure(inp)
	}

	if coded.Code().IsFailure() {
		return nil, WrapCodedError(
			coded.Code(),
			inp,
			"failure caused by")
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

	// External errors are reported by the runtime but originate from the VM.
	// External errors may be fatal or non-fatal, so additional handling by SplitErrorTypes
	if externalErr, ok := errors.GetExternalError(err); ok {
		if recoveredErr, ok := externalErr.Recovered.(error); ok {
			// If the recovered value is an error, pass it to the original
			// error handler to distinguish between fatal and non-fatal errors.
			return recoveredErr
		}
		// if not recovered return
		return NewUnknownFailure(externalErr)
	}

	// All other errors are non-fatal Cadence errors.
	return NewCadenceRuntimeError(runErr)
}

// This returns true if the error or one of its nested errors matches the
// specified error code.
func HasErrorCode(err error, code ErrorCode) bool {
	return Find(err, code) != nil
}

// This recursively unwraps the error and returns first CodedError that matches
// the specified error code.
func Find(err error, code ErrorCode) CodedError {
	if err == nil {
		return nil
	}

	var coded CodedError
	if !As(err, &coded) {
		return nil
	}

	if coded.Code() == code {
		return coded
	}

	return Find(coded.Unwrap(), code)
}

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

// NewEventEncodingError construct a new CodedError which indicates
// that encoding event has failed
func NewEventEncodingError(err error) CodedError {
	return NewCodedError(
		ErrCodeEventEncodingError,
		"error while encoding emitted event: %w ", err)
}
