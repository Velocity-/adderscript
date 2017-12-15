package main

type Opcode int

const (
	op_pushconst  Opcode = 0
	op_jmp               = 1
	op_getlocal          = 2
	op_setlocal          = 3
	op_return            = 4
	op_jz                = 5
	op_eq                = 6
	op_call              = 7
	op_nativecall        = 8
	op_label             = 255
)