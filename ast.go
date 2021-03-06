package main

import (
	"fmt"
)

type ASTType int

const (
	TypeTrigger        ASTType = iota
	TypeMethodCall
	TypeFunc
	TypeBlockStmt
	TypeVarDecl
	TypeExprStmt
	TypeLiteral
	TypeIfStmt
	TypeLogicalExpr
	TypeIdentifierExpr  // Can be either a var or a method ref
	TypeVarAssign
)

type ASTNode interface {
	Type() ASTType
}

func (t ASTType) Type() ASTType {
	return t
}

type ASTTrigger struct {
	ASTType
	trigger   string
	value     string
	statement ASTNode

	entry  *Trigger
	method *Method
}

func (t ASTTrigger) String() string {
	return fmt.Sprintf("ASTTrigger{on=%s, id=%s, statement=...}", t.trigger, t.value)
}

func newTrigger(trigger string, value string, statement ASTNode) *ASTTrigger {
	return &ASTTrigger{
		trigger:   trigger,
		value:     value,
		ASTType:   TypeTrigger,
		statement: statement,
	}
}

type ASTMethodExpr struct {
	ASTType
	name       string
	parameters []ASTNode

	local  *Method
	native *RuntimeFunction
}

func (m ASTMethodExpr) String() string {
	return fmt.Sprintf("ASTMethodExpr{name=%s, params=%s}", m.name, m.parameters)
}

func newMethodExpr(name string, parameters ...ASTNode) *ASTMethodExpr {
	return &ASTMethodExpr{
		ASTType:    TypeMethodCall,
		name:       name,
		parameters: parameters,
	}
}

type ASTFunc struct {
	ASTType
	name      string
	arguments []FuncArgument
	body      ASTNode
}

type FuncArgument struct {
	name    string
	argtype string
}

func (p ASTFunc) String() string {
	return fmt.Sprintf("ASTFunc{name=%s, args=%+v}", p.name, p.arguments)
}

func newFunc(name string, body ASTNode, arguments ...FuncArgument) *ASTFunc {
	return &ASTFunc{
		ASTType:   TypeFunc,
		name:      name,
		body:      body,
		arguments: arguments,
	}
}

type ASTExprStatement struct {
	ASTType
	expression ASTNode
}

func newStmt(expr ASTNode) *ASTExprStatement {
	return &ASTExprStatement{
		ASTType:    TypeExprStmt,
		expression: expr,
	}
}

type LiteralType int

const (
	LiteralInteger LiteralType = iota
	LiteralLong    LiteralType = iota
	LiteralString
	LiteralBoolean

	LiteralUnknown = -1
)

func LiteralToVarType(t LiteralType) VariableType {
	if t == LiteralInteger {
		return VarTypeInt
	} else if t == LiteralLong {
		return VarTypeLong
	} else if t == LiteralString {
		return VarTypeString
	} else if t == LiteralBoolean {
		return VarTypeBool
	}

	return VarTypeUnresolved
}

type ASTLiteralExpr struct {
	ASTType
	literalType LiteralType
	value       interface{}
}

func newLiteral(t LiteralType, value interface{}) *ASTLiteralExpr {
	return &ASTLiteralExpr{
		ASTType:     TypeLiteral,
		literalType: t,
		value:       value,
	}
}

type ASTBlockStatement struct {
	ASTType
	statements []ASTNode
}

func newBlock(statements ...ASTNode) *ASTBlockStatement {
	return &ASTBlockStatement{
		ASTType:    TypeBlockStmt,
		statements: statements,
	}
}

type ASTVarDeclaration struct {
	ASTType
	varType  string
	varName  string
	varValue ASTNode // Optional. If non-nil, becomes an assign instruction too.

	variable *LocalVariable
}

func newAssignment(varType, varName string, varValue ASTNode) *ASTVarDeclaration {
	return &ASTVarDeclaration{
		ASTType:  TypeVarDecl,
		varType:  varType,
		varName:  varName,
		varValue: varValue,
	}
}

type ASTIfStmt struct {
	ASTType
	condition ASTNode
	ifTrue    ASTNode
	ifFalse   ASTNode
}

func newIfStmt(condition ASTNode, ifTrue ASTNode, ifFalse ASTNode) *ASTIfStmt {
	return &ASTIfStmt{
		ASTType:   TypeIfStmt,
		condition: condition,
		ifTrue:    ifTrue,
		ifFalse:   ifFalse,
	}
}

type ASTVarAssign struct {
	ASTType
	varName  string
	varValue ASTNode
}

func newVarAssign(varName string, varValue ASTNode) *ASTVarAssign {
	return &ASTVarAssign{
		ASTType:  TypeVarAssign,
		varName:  varName,
		varValue: varValue,
	}
}

type ASTLogicalExpr struct {
	ASTType
	left       ASTNode
	comparator tokenType
	right      ASTNode
}

func newLogicalExpr(left ASTNode, comparator tokenType, right ASTNode) *ASTLogicalExpr {
	return &ASTLogicalExpr{
		ASTType:    TypeLogicalExpr,
		left:       left,
		comparator: comparator,
		right:      right,
	}
}

type ASTIdentifierExpr struct {
	ASTType
	identifier string

	resolved *LocalVariable
}

func newIdentifier(identifier string) *ASTIdentifierExpr {
	return &ASTIdentifierExpr{
		ASTType:    TypeIdentifierExpr,
		identifier: identifier,
	}
}
