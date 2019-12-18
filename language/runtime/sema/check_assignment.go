package sema

import (
	"github.com/dapperlabs/flow-go/language/runtime/ast"
	"github.com/dapperlabs/flow-go/language/runtime/common"
	"github.com/dapperlabs/flow-go/language/runtime/errors"
)

func (checker *Checker) VisitAssignmentStatement(assignment *ast.AssignmentStatement) ast.Repr {
	targetType, valueType := checker.checkAssignment(
		assignment.Target,
		assignment.Value,
		assignment.Transfer,
		false,
	)

	checker.Elaboration.AssignmentStatementValueTypes[assignment] = valueType
	checker.Elaboration.AssignmentStatementTargetTypes[assignment] = targetType

	return nil
}

func (checker *Checker) checkAssignment(
	target, value ast.Expression,
	transfer *ast.Transfer,
	isResourceAssignment bool,
) (targetType, valueType Type) {
	valueType = value.Accept(checker).(Type)

	targetType = checker.visitAssignmentValueType(target, value, valueType)

	// NOTE: `visitAssignmentValueType` checked compatibility between value and target types.
	// Check for the *target* type, so that assignment using non-resource typed value (e.g. `nil`)
	// is possible

	checker.checkTransfer(transfer, targetType)

	// An assignment to a resource is invalid, as it would result in a loss

	if targetType.IsResourceType() {

		// However, an assignment to a `self` field in the initializer is allowed.
		// In that case the value that is assigned must be invalidated.
		//
		// The check for a repeated assignment of a constant field after initialization
		// is not part of this logic here, see `visitMemberExpressionAssignment`

		accessedSelfMember := checker.accessedSelfMember(target)

		if (accessedSelfMember == nil || checker.functionActivations.Current().InitializationInfo == nil) && !isResourceAssignment {
			checker.report(
				&InvalidResourceAssignmentError{
					Range: ast.NewRangeFromPositioned(target),
				},
			)
		}
	}

	checker.checkVariableMove(value)

	if valueType.IsResourceType() {
		checker.recordResourceInvalidation(
			value,
			valueType,
			ResourceInvalidationKindMove,
		)
	}

	return
}

func (checker *Checker) accessedSelfMember(expression ast.Expression) *Member {
	memberExpression, isMemberExpression := expression.(*ast.MemberExpression)
	if !isMemberExpression {
		return nil
	}

	identifierExpression, isIdentifierExpression := memberExpression.Expression.(*ast.IdentifierExpression)
	if !isIdentifierExpression {
		return nil
	}

	variable := checker.valueActivations.Find(identifierExpression.Identifier.Identifier)
	if variable == nil ||
		variable.DeclarationKind != common.DeclarationKindSelf {

		return nil
	}

	var members map[string]*Member
	switch containerType := variable.Type.(type) {
	case *CompositeType:
		members = containerType.Members
	case *InterfaceType:
		members = containerType.Members
	case *TransactionType:
		members = containerType.Members
	default:
		panic(errors.NewUnreachableError())
	}

	fieldName := memberExpression.Identifier.Identifier

	return members[fieldName]
}

func (checker *Checker) visitAssignmentValueType(
	targetExpression ast.Expression,
	valueExpression ast.Expression,
	valueType Type,
) (targetType Type) {

	inAssignment := checker.inAssignment
	checker.inAssignment = true
	defer func() {
		checker.inAssignment = inAssignment
	}()

	switch target := targetExpression.(type) {
	case *ast.IdentifierExpression:
		return checker.visitIdentifierExpressionAssignment(valueExpression, target, valueType)

	case *ast.IndexExpression:
		return checker.visitIndexExpressionAssignment(valueExpression, target, valueType)

	case *ast.MemberExpression:
		return checker.visitMemberExpressionAssignment(valueExpression, target, valueType)

	default:
		panic(&unsupportedAssignmentTargetExpression{
			target: target,
		})
	}
}

func (checker *Checker) visitIdentifierExpressionAssignment(
	valueExpression ast.Expression,
	target *ast.IdentifierExpression,
	valueType Type,
) (targetType Type) {
	identifier := target.Identifier.Identifier

	// check identifier was declared before
	variable := checker.findAndCheckVariable(target.Identifier, true)
	if variable == nil {
		return &InvalidType{}
	}

	// check identifier is not a constant
	if variable.IsConstant {
		checker.report(
			&AssignmentToConstantError{
				Name:  identifier,
				Range: ast.NewRangeFromPositioned(target),
			},
		)
	}

	// check value type is subtype of variable type
	if !valueType.IsInvalidType() &&
		!variable.Type.IsInvalidType() &&
		!checker.IsTypeCompatible(valueExpression, valueType, variable.Type) {

		checker.report(
			&TypeMismatchError{
				ExpectedType: variable.Type,
				ActualType:   valueType,
				Range:        ast.NewRangeFromPositioned(valueExpression),
			},
		)
	}

	return variable.Type
}

func (checker *Checker) visitIndexExpressionAssignment(
	valueExpression ast.Expression,
	target *ast.IndexExpression,
	valueType Type,
) (elementType Type) {

	elementType, _ = checker.visitIndexExpression(target, true)

	if elementType == nil {
		return &InvalidType{}
	}

	if !valueType.IsInvalidType() &&
		!elementType.IsInvalidType() &&
		!checker.IsTypeCompatible(valueExpression, valueType, elementType) {

		checker.report(
			&TypeMismatchError{
				ExpectedType: elementType,
				ActualType:   valueType,
				Range:        ast.NewRangeFromPositioned(valueExpression),
			},
		)
	}

	return elementType
}

func (checker *Checker) visitMemberExpressionAssignment(
	valueExpression ast.Expression,
	target *ast.MemberExpression,
	valueType Type,
) (memberType Type) {

	member, isOptional := checker.visitMember(target)

	if member == nil {
		return &InvalidType{}
	}

	if isOptional {
		checker.report(
			&UnsupportedOptionalChainingAssignmentError{
				Range: ast.NewRangeFromPositioned(target),
			},
		)
	}

	// If the value type is valid, check that the value can be assigned to the member type

	if !valueType.IsInvalidType() &&
		!member.TypeAnnotation.Type.IsInvalidType() &&
		!checker.IsTypeCompatible(valueExpression, valueType, member.TypeAnnotation.Type) {

		checker.report(
			&TypeMismatchError{
				ExpectedType: member.TypeAnnotation.Type,
				ActualType:   valueType,
				Range:        ast.NewRangeFromPositioned(valueExpression),
			},
		)
	}

	if !checker.isWriteableMember(member) {
		checker.report(
			&InvalidAssignmentAccessError{
				Name:              member.Identifier.Identifier,
				RestrictingAccess: member.Access,
				DeclarationKind:   member.DeclarationKind,
				Range:             ast.NewRangeFromPositioned(member.Identifier),
			},
		)
	}

	reportAssignmentToConstant := func() {
		checker.report(
			&AssignmentToConstantMemberError{
				Name:  target.Identifier.Identifier,
				Range: ast.NewRangeFromPositioned(valueExpression),
			},
		)
	}

	// If this is an assignment to a `self` field, it needs special handling
	// depending on if the assignment is in an initializer or not

	accessedSelfMember := checker.accessedSelfMember(target)
	if accessedSelfMember != nil {

		functionActivation := checker.functionActivations.Current()

		// If this is an assignment to a `self` field in the initializer,
		// ensure it is only assigned once, to initialize it

		if functionActivation.InitializationInfo != nil {

			// If the function has already returned, the initialization
			// is not definitive, and it must be ignored

			// NOTE: assignment can still be considered definitive
			//  if the function maybe halted

			if !functionActivation.ReturnInfo.MaybeReturned {

				// If the field is constant and it has already previously been
				// initialized, report an error for the repeated assignment

				initializedFieldMembers := functionActivation.InitializationInfo.InitializedFieldMembers

				if accessedSelfMember.VariableKind == ast.VariableKindConstant &&
					initializedFieldMembers.Contains(accessedSelfMember) {

					// TODO: dedicated error: assignment to constant after initialization

					reportAssignmentToConstant()
				} else if functionActivation.InitializationInfo.FieldMembers[accessedSelfMember] == nil {
					// This field is not supposed to be initialized

					reportAssignmentToConstant()
				} else {
					// This is the initial assignment to the field, record it

					initializedFieldMembers.Add(accessedSelfMember)
				}
			}

		} else if accessedSelfMember.VariableKind == ast.VariableKindConstant {

			// If this is an assignment outside the initializer,
			// an assignment to a constant field is invalid

			reportAssignmentToConstant()
		}

	} else {

		// The assignment is not to a `self` field. Report if there is an attempt
		// to assign to a constant field, which is always invalid,
		// independent of the location of the assignment (initializer or not)

		if member.VariableKind == ast.VariableKindConstant {

			reportAssignmentToConstant()
		}
	}

	return member.TypeAnnotation.Type
}
