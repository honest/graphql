package graphql

import (
	"github.com/graphql-go/graphql/gqlerrors"
	"github.com/graphql-go/graphql/language/ast"
	"github.com/graphql-go/graphql/language/parser"
	"github.com/graphql-go/graphql/language/source"
	"golang.org/x/net/context"
)

type Params struct {
	// The GraphQL type system to use when validating and executing a query.
	Schema Schema

	// A GraphQL language formatted string representing the requested operation.
	RequestString string

	// The value provided as the first argument to resolver functions on the top
	// level type (e.g. the query object type).
	RootObject map[string]interface{}

	// A mapping of variable name to runtime value to use for all variables
	// defined in the requestString.
	VariableValues map[string]interface{}

	// The name of the operation to use if requestString contains multiple
	// possible operations. Can be omitted if requestString contains only
	// one operation.
	OperationName string

	// Context may be provided to pass application-specific per-request
	// information to resolve functions.
	Context context.Context

	// SkipValidation allows skipping validation of the GraphQL request.
	// Should only be used with trusted clients for the performance benefits.
	SkipValidation bool

	// CustomValidationRules are extra validators to run after the AST has been parsed.  These follow a visitor pattern that can be upon entry or exit from each node.
	CustomValidationRules []func(context *ValidationContext) *ValidationRuleInstance

	// ErrorHandlerFn expects an error handler
	ErrorHandlerFn func(err error)
}

var cachedASTs map[string]*ast.Document = make(map[string]*ast.Document)

func Do(p Params) *Result {
	var AST *ast.Document
	var ok bool
	var err error
	if AST, ok = cachedASTs[p.RequestString]; !ok {
		source := source.NewSource(&source.Source{
			Body: p.RequestString,
			Name: "GraphQL request",
		})
		AST, err = parser.Parse(parser.ParseParams{Source: source})
		if err != nil {
			return &Result{
				Errors: gqlerrors.FormatErrors(err),
			}
		}
		cachedASTs[p.RequestString] = AST
	}

	if !p.SkipValidation {
		rules := SpecifiedRules
		if p.CustomValidationRules != nil {
			for _, rule := range p.CustomValidationRules {
				rules = append(rules, rule)
			}
		}
		validationResult := ValidateDocument(&p.Schema, AST, rules)

		if !validationResult.IsValid {
			return &Result{
				Errors: validationResult.Errors,
			}
		}
	}

	return Execute(ExecuteParams{
		Schema:         p.Schema,
		Root:           p.RootObject,
		AST:            AST,
		OperationName:  p.OperationName,
		Args:           p.VariableValues,
		Context:        p.Context,
		ErrorHandlerFn: p.ErrorHandlerFn,
	})
}
