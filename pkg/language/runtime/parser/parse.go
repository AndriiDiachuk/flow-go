package parser

import (
	"fmt"
	goRuntime "runtime"

	"github.com/antlr/antlr4/runtime/Go/antlr"

	"github.com/dapperlabs/bamboo-node/pkg/language/runtime/ast"
)

type errorListener struct {
	*antlr.DefaultErrorListener
	syntaxErrors []*SyntaxError
}

func (l *errorListener) SyntaxError(
	recognizer antlr.Recognizer,
	offendingSymbol interface{},
	line, column int,
	message string,
	e antlr.RecognitionException,
) {
	position := ast.PositionFromToken(offendingSymbol.(antlr.Token))

	l.syntaxErrors = append(l.syntaxErrors, &SyntaxError{
		Pos:     position,
		Message: message,
	})
}

func Parse(code string) (program *ast.Program, errors []error) {
	input := antlr.NewInputStream(code)
	lexer := NewStrictusLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, 0)
	parser := NewStrictusParser(stream)
	// diagnostics, for debugging only:
	// parser.AddErrorListener(antlr.NewDiagnosticErrorListener(true))
	listener := new(errorListener)
	// remove the default console error listener
	parser.RemoveErrorListeners()
	parser.AddErrorListener(listener)

	appendSyntaxErrors := func() {
		for _, syntaxError := range listener.syntaxErrors {
			errors = append(errors, syntaxError)
		}
	}

	// recover internal panics and return them as an error
	defer func() {
		if r := recover(); r != nil {
			var ok bool
			var err error
			// don't recover Go errors
			err, ok = r.(goRuntime.Error)
			if ok {
				panic(err)
			}
			err, ok = r.(error)
			if !ok {
				err = fmt.Errorf("%v", r)
			}
			appendSyntaxErrors()
			errors = append(errors, err)
			program = nil
		}
	}()

	parsed := parser.Program()

	appendSyntaxErrors()

	if len(errors) > 0 {
		return nil, errors
	}

	result := parsed.Accept(&ProgramVisitor{})

	program, ok := result.(*ast.Program)
	if !ok {
		return nil, errors
	}

	return program, errors
}
