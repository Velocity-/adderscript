package main

import (
	"fmt"
	"strconv"
)

type Assembler struct {
	methods      []*Method
	triggers     []*Trigger
	runtime      *AdderRuntime
	methodIndex  int
	triggerIndex int
	cpool        ConstantPool
}

type ConstantPool struct {
	values []*ConstantPoolEntry
}

type ConstantPoolEntry struct {
	Type VariableType
	Value interface{}
}

type Trigger struct {
	name       string
	definition *RuntimeListener
	label      *Instruction
	value      uint64
}

type Method struct {
	name         string
	index        int
	instructions []*Instruction
	variables    []*LocalVariable
	arguments    []*LocalVariable
	entry        *Instruction

	lvtIndex int
	labelPtr int
}

type NativeMethod struct {
	name   string
	opcode int
}

type Instruction struct {
	// opcode is the operation code of this instruction
	opcode

	// i is the index of the int in the constant pool
	i int

	// l is the index of the long in the constant pool
	l int

	// s is the index of the string in the constant pool
	s int

	// address is the computed address (offset) of this instruction
	address int

	// labelFunc is a function that will be called once the address of this instruction is defined. Used in labels.
	labelFunc func(address int)
}

type LocalVariable struct {
	index int
	name  string
}

type VariableType int

const (
	VarTypeInt    VariableType = iota
	VarTypeLong
	VarTypeString
	VarTypeBool
	VarTypeVoid    // Not a variable type, but defined to be used with method types

	VarTypeUnresolved VariableType = -1
)

func (a *Assembler) AssembleProgram(rootNodes []ASTNode) {
	// Hoist proc declarations
	for _, v := range rootNodes {
		if v.Type() == TypeProc {
			a.defineProc(v.(*ASTProc))
		}
	}

	for _, v := range rootNodes {
		a.assembleNode(v, nil)
	}

	// Compute addresses
	address := 0
	for _, method := range a.methods {
		for _, instr := range method.instructions {
			instr.address = address

			// Execute instruction function if any
			if instr.labelFunc != nil {
				instr.labelFunc(address)
			}

			// Labels do not alter the address. They're filtered out during encoding.
			if instr.opcode != op_label {
				address++
			}
		}
	}
}

func (a *Assembler) assembleNode(node ASTNode, method *Method) {
	switch n := node.(type) {
	case *ASTTrigger:
		a.assembleTrigger(n)
	case *ASTProc:
		a.assembleProc(n)
	case *ASTBlockStatement:
		a.assembleBlock(n, method)
	case *ASTVarDeclaration:
		a.assembleVarDecl(n, method)
	case *ASTMethodExpr:
		a.assembleMethodExpr(n, method)
	case *ASTLiteralExpr:
		a.assembleLiteralExpr(n, method)
	case *ASTIfStmt:
		a.assembleIfStmt(n, method)
	case *ASTLogicalExpr:
		a.assembleLogicalExpr(n, method)
	case *ASTIdentifierExpr:
		a.assembleIdentifierExpr(n, method)
	default:
		panic(fmt.Sprintf("No function to walk node: %T", node))
	}
}

func TypeOfNode(node ASTNode) VariableType {
	switch t := node.(type) {
	case *ASTLiteralExpr:
		if t.literalType == LiteralInteger {
			return VarTypeInt
		} else if t.literalType == LiteralLong {
			return VarTypeLong
		} else if t.literalType == LiteralString {
			return VarTypeString
		} else if t.literalType == LiteralBoolean {
			return VarTypeBool
		}
	}

	panic(fmt.Sprintf("cannot resolve type of node: %T", node))
}

func (a *Assembler) assembleTrigger(n *ASTTrigger) {
	var trigger Trigger
	trigger.name = n.trigger

	// Resolve the trigger uid
	listener := a.runtime.FindListener(trigger.name)
	if listener == nil {
		panic(fmt.Errorf("unknown trigger %s, not defined in runtime", trigger.name))
	}

	trigger.definition = listener

	// Verify that the filter value is a valid value.
	// For now, values are longs only. This is subject to change.
	parsed, err := strconv.ParseUint(n.value, 10, 64)
	if err != nil {
		panic(fmt.Sprintf("cannot parse trigger value into long: %s", n.value))
	}

	method := a.defineMethod("@" + n.trigger + "@" + n.value + "@" + strconv.Itoa(a.triggerIndex))
	a.triggerIndex++

	label := method.newLabel()
	method.emit(label)
	trigger.value = parsed
	trigger.label = label

	a.triggers = append(a.triggers, &trigger)

	// Assemble the code belonging to this call
	a.assembleNode(n.statement, method)

	// Drop a return statement
	method.emitOp(op_return)
}

func (a *Assembler) defineProc(n *ASTProc) {
	method := a.resolveMethod(n.name)
	if method != nil {
		panic(fmt.Sprintf("redefining method: %s", n.name))
	}

	method = a.defineMethod(n.name)

	// Define method parameters as local variables
	for _, arg := range n.arguments {
		var vt = ResolveVarType(arg.argtype)

		if vt == VarTypeUnresolved {
			panic(fmt.Sprintf("unresolved variable type %s", arg.argtype))
		}

		lv := method.defineVariable(arg.name, vt)
		method.arguments = append(method.arguments, lv)
	}
}

func (a *Assembler) assembleProc(n *ASTProc) {
	m := a.resolveMethod(n.name)
	a.assembleNode(n.body, m)
	m.emitOp(op_return)
}

func (a *Assembler) assembleBlock(n *ASTBlockStatement, m *Method) {
	for _, v := range n.statements {
		a.assembleNode(v, m)
	}
}

func (a *Assembler) assembleIfStmt(n *ASTIfStmt, m *Method) {
	a.assembleNode(n.condition, m)
	lblFalse := m.newLabel()
	lblEnd := m.newLabel()

	// JZ to lblFalse - absolute
	jzToLblfalse := m.emit(instr(op_jz, 0, 0, 0)) // Jump if false (0) to the false block
	lblFalse.labelFunc = func(address int) {
		jzToLblfalse.i = address
	}

	// Encode true block (jz jumps over this if expression is false)
	a.assembleNode(n.ifTrue, m)
	jmpToLblend := m.emit(instr(op_jmp, 0, 0, 0))
	lblEnd.labelFunc = func(address int) {
		jmpToLblend.i = address
	}

	// Encode false block (the true block jumps over this)
	m.emit(lblFalse)

	if n.ifFalse != nil {
		a.assembleNode(n.ifFalse, m)
	}

	// Add the end node - the true block jumps to this to avoid executing the false block.
	m.emit(lblEnd)
}

func (a *Assembler) assembleVarDecl(n *ASTVarDeclaration, m *Method) {
	var vartype = ResolveVarType(n.varType)

	if vartype == VarTypeUnresolved {
		panic("unresolved variable type: " + n.varType)
	}

	// See if this variable is already defined...
	if m.resolveVariable(n.varName) != nil {
		panic("variable redeclared: " + n.varName)
	}

	local := m.defineVariable(n.varName, vartype)

	// If the local var has any expression defined, assemble it
	if n.varValue != nil {
		a.assembleNode(n.varValue, m)
		// todo all types
		m.emit(instr(op_setivar, local.index, 0, 0))
	}
}

func ResolveVarType(varType string) VariableType {
	switch varType {
	case "int":
		return VarTypeInt
	case "long":
		return VarTypeLong
	case "string":
		return VarTypeString
	case "bool":
		return VarTypeBool
	default:
		return VarTypeUnresolved
	}
}

func ResolveType(typ string) VariableType {
	switch typ {
	case "void":
		return VarTypeVoid
	default:
		return ResolveVarType(typ)
	}
}

func (a *Assembler) assembleMethodExpr(n *ASTMethodExpr, m *Method) {
	// Form list of argument types
	var types []VariableType
	for _, v := range n.parameters {
		types = append(types, TypeOfNode(v))
	}

	// See if this is a native method first. Likelihood is much greater.
	nativeMethod := a.runtime.FindFunctionWithArguments(n.name, types...)
	var localMethod *Method

	if nativeMethod == nil {
		localMethod = a.resolveMethod(n.name)

		// Still not found? Panic.
		if localMethod == nil {
			params := "(" + TypeListToString(", ", types...) + ")"
			panic("Cannot resolve local or native method: " + n.name + params)
		}
	}

	// Assemble method parameters
	for _, v := range n.parameters {
		a.assembleNode(v, m)
	}

	if nativeMethod != nil {
		m.emit(instr(op_nativecall, nativeMethod.InternalId, 0, 0))
	} else {
		m.emit(instr(op_call, localMethod.index, 0, 0))
	}
}

func (a *Assembler) assembleLogicalExpr(n *ASTLogicalExpr, m *Method) {
	a.assembleNode(n.left, m)
	a.assembleNode(n.right, m)

	switch n.comparator {
	case tokenEqual:
		m.emitOp(op_eq)
	default:
		panic("unknown comparator node! " + strconv.Itoa(int(n.comparator)))
	}
}

func (a *Assembler) assembleIdentifierExpr(n *ASTIdentifierExpr, m *Method) {
	//TODO type checks
	local := m.resolveVariable(n.identifier)
	if local == nil {
		panic("undefined variable: " + n.identifier)
	}

	m.emit(instr(op_getivar, local.index, 0, 0))
}

func (a *Assembler) assembleLiteralExpr(n *ASTLiteralExpr, m *Method) {
	if n.literalType == LiteralString {
		m.emit(instr(op_pushconst, a.cpool.getString(n.value.(string)), 0, 0))
	} else if n.literalType == LiteralInteger {
		m.emit(instr(op_pushconst, a.cpool.getInt(int(n.value.(int))), 0, 0))
	} else if n.literalType == LiteralLong {
		m.emit(instr(op_pushconst, a.cpool.getLong(n.value.(int64)), 0, 0))
	} else if n.literalType == LiteralBoolean {
		if n.value == "true" {
			m.emit(instr(op_pushconst, a.cpool.getInt(int(1)), 0, 0))
		} else {
			m.emit(instr(op_pushconst, a.cpool.getInt(int(0)), 0, 0))
		}
	} else {
		panic(fmt.Sprintf("unknown literal type %d (value %s)", n.literalType, n.value))
	}
}

func (a *Assembler) resolveMethod(name string) *Method {
	for _, v := range a.methods {
		if v.name == name {
			return v
		}
	}

	return nil
}

func (a *Assembler) defineMethod(name string) *Method {
	index := a.methodIndex
	a.methodIndex++

	method := &Method{
		name:         name,
		index:        index,
		instructions: make([]*Instruction, 512)[:0],
		variables:    make([]*LocalVariable, 4)[:0],
	}

	// Define entry point, drop a label.
	label := method.newLabel()
	method.emit(label)
	method.entry = label

	a.methods = append(a.methods, method)
	return method
}

func (m *Method) resolveVariable(name string) *LocalVariable {
	for _, v := range m.variables {
		if v.name == name {
			return v
		}
	}

	return nil
}

func (a *Method) defineVariable(name string, t VariableType) *LocalVariable {
	index := a.lvtIndex
	a.lvtIndex++

	local := &LocalVariable{
		name:  name,
		index: index,
	}

	a.variables = append(a.variables, local)
	return local
}

func (m *Method) emit(instruction *Instruction) *Instruction {
	m.instructions = append(m.instructions, instruction)
	return instruction
}

func (m *Method) emitOp(op opcode) {
	instruction := instr(op, 0, 0, 0)
	m.instructions = append(m.instructions, instruction)
}

func (c *ConstantPool) getInt(i int) int {
	for k, v := range c.values {
		if v.Type == VarTypeInt && v.Value.(int) == i {
			return k
		}
	}

	c.values = append(c.values, &ConstantPoolEntry{
		Type:  VarTypeInt,
		Value: i,
	})

	return len(c.values) - 1
}

func (c *ConstantPool) getLong(i int64) int {
	for k, v := range c.values {
		if v.Type == VarTypeLong && v.Value.(int64) == i {
			return k
		}
	}

	c.values = append(c.values, &ConstantPoolEntry{
		Type:  VarTypeLong,
		Value: i,
	})

	return len(c.values) - 1
}

func (c *ConstantPool) getString(s string) int {
	for k, v := range c.values {
		if v.Type == VarTypeString && v.Value.(string) == s {
			return k
		}
	}

	c.values = append(c.values, &ConstantPoolEntry{
		Type:  VarTypeString,
		Value: s,
	})

	return len(c.values) - 1
}

func instr(op opcode, i int, l int, s int) *Instruction {
	return &Instruction{opcode: op, i: i, l: l, s: s}
}

func (m *Method) newLabel() *Instruction {
	m.labelPtr++

	return &Instruction{
		opcode: op_label,
	}
}
