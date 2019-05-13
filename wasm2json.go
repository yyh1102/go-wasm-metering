package go_wasm_metering

import (
	"encoding/json"
	"math/big"
	"os"
	"reflect"
	"strings"
)

var (
	// https://github.com/WebAssembly/design/blob/master/BinaryEncoding.md#language-types
	// All types are distinguished by a negative varint7 values that is the first
	// byte of their encoding (representing a type constructor)
	W2J_LANGUAGE_TYPES = map[byte]string{
		0x7f: "i32",
		0x7e: "i64",
		0x7d: "f32",
		0x7c: "f64",
		0x70: "anyFunc",
		0x60: "func",
		0x40: "block_type",
	}

	// https://github.com/WebAssembly/design/blob/master/BinaryEncoding.md#external_kind
	// A single-byte unsigned integer indicating the kind of definition being imported or defined:
	W2J_EXTERNAL_KIND = map[byte]string{
		0x00: "function",
		0x01: "table",
		0x02: "memory",
		0x03: "global",
	}

	W2J_OPCODES = map[byte]string{
		// flow control
		0x0: "unreachable",
		0x1: "nop",
		0x2: "block",
		0x3: "loop",
		0x4: "if",
		0x5: "else",
		0xb: "end",
		0xc: "br",
		0xd: "br_if",
		0xe: "br_table",
		0xf: "return",

		// calls
		0x10: "call",
		0x11: "call_indirect",

		// Parametric operators
		0x1a: "drop",
		0x1b: "select",

		// Varibale access
		0x20: "get_local",
		0x21: "set_local",
		0x22: "tee_local",
		0x23: "get_global",
		0x24: "set_global",

		// Memory-related operators
		0x28: "i32.load",
		0x29: "i64.load",
		0x2a: "f32.load",
		0x2b: "f64.load",
		0x2c: "i32.load8_s",
		0x2d: "i32.load8_u",
		0x2e: "i32.load16_s",
		0x2f: "i32.load16_u",
		0x30: "i64.load8_s",
		0x31: "i64.load8_u",
		0x32: "i64.load16_s",
		0x33: "i64.load16_u",
		0x34: "i64.load32_s",
		0x35: "i64.load32_u",
		0x36: "i32.store",
		0x37: "i64.store",
		0x38: "f32.store",
		0x39: "f64.store",
		0x3a: "i32.store8",
		0x3b: "i32.store16",
		0x3c: "i64.store8",
		0x3d: "i64.store16",
		0x3e: "i64.store32",
		0x3f: "current_memory",
		0x40: "grow_memory",

		// Constants
		0x41: "i32.const",
		0x42: "i64.const",
		0x43: "f32.const",
		0x44: "f64.const",

		// Comparison operators
		0x45: "i32.eqz",
		0x46: "i32.eq",
		0x47: "i32.ne",
		0x48: "i32.lt_s",
		0x49: "i32.lt_u",
		0x4a: "i32.gt_s",
		0x4b: "i32.gt_u",
		0x4c: "i32.le_s",
		0x4d: "i32.le_u",
		0x4e: "i32.ge_s",
		0x4f: "i32.ge_u",
		0x50: "i64.eqz",
		0x51: "i64.eq",
		0x52: "i64.ne",
		0x53: "i64.lt_s",
		0x54: "i64.lt_u",
		0x55: "i64.gt_s",
		0x56: "i64.gt_u",
		0x57: "i64.le_s",
		0x58: "i64.le_u",
		0x59: "i64.ge_s",
		0x5a: "i64.ge_u",
		0x5b: "f32.eq",
		0x5c: "f32.ne",
		0x5d: "f32.lt",
		0x5e: "f32.gt",
		0x5f: "f32.le",
		0x60: "f32.ge",
		0x61: "f64.eq",
		0x62: "f64.ne",
		0x63: "f64.lt",
		0x64: "f64.gt",
		0x65: "f64.le",
		0x66: "f64.ge",

		// Numeric operators
		0x67: "i32.clz",
		0x68: "i32.ctz",
		0x69: "i32.popcnt",
		0x6a: "i32.add",
		0x6b: "i32.sub",
		0x6c: "i32.mul",
		0x6d: "i32.div_s",
		0x6e: "i32.div_u",
		0x6f: "i32.rem_s",
		0x70: "i32.rem_u",
		0x71: "i32.and",
		0x72: "i32.or",
		0x73: "i32.xor",
		0x74: "i32.shl",
		0x75: "i32.shr_s",
		0x76: "i32.shr_u",
		0x77: "i32.rotl",
		0x78: "i32.rotr",
		0x79: "i64.clz",
		0x7a: "i64.ctz",
		0x7b: "i64.popcnt",
		0x7c: "i64.add",
		0x7d: "i64.sub",
		0x7e: "i64.mul",
		0x7f: "i64.div_s",
		0x80: "i64.div_u",
		0x81: "i64.rem_s",
		0x82: "i64.rem_u",
		0x83: "i64.and",
		0x84: "i64.or",
		0x85: "i64.xor",
		0x86: "i64.shl",
		0x87: "i64.shr_s",
		0x88: "i64.shr_u",
		0x89: "i64.rotl",
		0x8a: "i64.rotr",
		0x8b: "f32.abs",
		0x8c: "f32.neg",
		0x8d: "f32.ceil",
		0x8e: "f32.floor",
		0x8f: "f32.trunc",
		0x90: "f32.nearest",
		0x91: "f32.sqrt",
		0x92: "f32.add",
		0x93: "f32.sub",
		0x94: "f32.mul",
		0x95: "f32.div",
		0x96: "f32.min",
		0x97: "f32.max",
		0x98: "f32.copysign",
		0x99: "f64.abs",
		0x9a: "f64.neg",
		0x9b: "f64.ceil",
		0x9c: "f64.floor",
		0x9d: "f64.trunc",
		0x9e: "f64.nearest",
		0x9f: "f64.sqrt",
		0xa0: "f64.add",
		0xa1: "f64.sub",
		0xa2: "f64.mul",
		0xa3: "f64.div",
		0xa4: "f64.min",
		0xa5: "f64.max",
		0xa6: "f64.copysign",

		// Conversions
		0xa7: "i32.wrap/i64",
		0xa8: "i32.trunc_s/f32",
		0xa9: "i32.trunc_u/f32",
		0xaa: "i32.trunc_s/f64",
		0xab: "i32.trunc_u/f64",
		0xac: "i64.extend_s/i32",
		0xad: "i64.extend_u/i32",
		0xae: "i64.trunc_s/f32",
		0xaf: "i64.trunc_u/f32",
		0xb0: "i64.trunc_s/f64",
		0xb1: "i64.trunc_u/f64",
		0xb2: "f32.convert_s/i32",
		0xb3: "f32.convert_u/i32",
		0xb4: "f32.convert_s/i64",
		0xb5: "f32.convert_u/i64",
		0xb6: "f32.demote/f64",
		0xb7: "f64.convert_s/i32",
		0xb8: "f64.convert_u/i32",
		0xb9: "f64.convert_s/i64",
		0xba: "f64.convert_u/i64",
		0xbb: "f64.promote/f32",

		// Reinterpretations
		0xbc: "i32.reinterpret/f32",
		0xbd: "i64.reinterpret/f64",
		0xbe: "f32.reinterpret/i32",
		0xbf: "f64.reinterpret/i64",
	}

	SECTION_IDS = map[byte]string{
		0:  "custom",
		1:  "type",
		2:  "import",
		3:  "function",
		4:  "table",
		5:  "memory",
		6:  "global",
		7:  "export",
		8:  "start",
		9:  "element",
		10: "code",
		11: "data",
	}

	W2J_OP_IMMEDIATES = make(JSON)
)

func init() {
	immediates, err := os.Open("immediates.json")
	if err != nil {
		panic(err)
	}

	jsonParser := json.NewDecoder(immediates)
	if err := jsonParser.Decode(&W2J_OP_IMMEDIATES); err != nil {
		panic(err)
	}
}

type immediataryParsers struct{}

func (immediataryParsers) varuint1(stream *Stream) int8 {
	return int8(stream.ReadByte())
}

func (immediataryParsers) varuint32(stream *Stream) uint32 {
	return uint32(DecodeULEB128(stream))
}

func (immediataryParsers) varint32(stream *Stream) int32 {
	return int32(DecodeSLEB128(stream))
}

func (immediataryParsers) varint64(stream *Stream) int64 {
	return int64(DecodeSLEB128(stream))
}

func (immediataryParsers) uint32(stream *Stream) uint32 {
	buf := stream.Read(4)

	return uint32(new(big.Int).SetBytes(buf).Uint64())
}

func (immediataryParsers) uint64(stream *Stream) uint64 {
	buf := stream.Read(8)
	return new(big.Int).SetBytes(buf).Uint64()
}

func (immediataryParsers) block_type(stream *Stream) string {
	return W2J_LANGUAGE_TYPES[stream.ReadByte()]
}

func (immediataryParsers) br_table(stream *Stream) JSON {
	jsonObj := make(JSON)
	targets := []uint64{}

	num := DecodeULEB128(stream)
	for i := uint64(0); i < num; i++ {
		target := DecodeULEB128(stream)
		targets = append(targets, target)
	}

	jsonObj["Targets"] = targets
	jsonObj["DefaultTarget"] = DecodeULEB128(stream)
	return jsonObj
}

func (immediataryParsers) call_indirect(stream *Stream) JSON {
	jsonObj := make(JSON)
	jsonObj["Index"] = DecodeULEB128(stream)
	jsonObj["Reserved"] = stream.ReadByte()
	return jsonObj
}

func (immediataryParsers) memory_immediate(stream *Stream) JSON {
	jsonObj := make(JSON)
	jsonObj["Flags"] = DecodeULEB128(stream)
	jsonObj["Offset"] = DecodeULEB128(stream)
	return jsonObj
}

type typeParsers struct{}

func (typeParsers) function(stream *Stream) uint64 {
	return DecodeULEB128(stream)
}

func (t typeParsers) table(stream *Stream) Table {
	typ := stream.ReadByte()
	return Table{
		ElementType: W2J_LANGUAGE_TYPES[typ],
		Limits:      t.memory(stream),
	}
}

func (typeParsers) global(stream *Stream) Global {
	typ := stream.ReadByte()
	mutability := stream.ReadByte()
	return Global{
		ContentType: W2J_LANGUAGE_TYPES[typ],
		Mutability:  mutability,
	}
}

func (typeParsers) memory(stream *Stream) MemLimits {
	flags := DecodeULEB128(stream)
	intial := DecodeULEB128(stream)
	limits := MemLimits{
		Flags:  flags,
		Intial: intial,
	}
	if flags == 1 {
		limits.Maximum = DecodeULEB128(stream)
	}
	return limits
}

func (typeParsers) initExpr(stream *Stream) OP {
	op := ParseOp(stream)
	stream.ReadByte() // skip the `end`
	return op
}

type sectionParsers struct{}

func (sectionParsers) custom(stream *Stream, header SectionHeader) CustomSec {
	sec := CustomSec{Name: "custom"}

	section := NewStream(stream.Read(int(header.Size)))
	nameLen := DecodeULEB128(section)
	name := stream.Read(int(nameLen))

	sec.SectionName = string(name)
	sec.Payload = section.Bytes()
	return sec
}

func (sectionParsers) _type(stream *Stream) TypeSec {
	numberOfEntries := DecodeULEB128(stream)
	typSec := TypeSec{
		Name: "type",
	}

	for i := uint64(0); i < numberOfEntries; i++ {
		typ := stream.ReadByte()
		entry := TypeEntry{
			Form:   W2J_LANGUAGE_TYPES[typ],
			Params: []string{},
		}

		paramCount := DecodeULEB128(stream)

		for j := uint64(0); j < paramCount; j++ {
			typ := stream.ReadByte()
			entry.Params = append(entry.Params, W2J_LANGUAGE_TYPES[typ])
		}

		numOfReturns := DecodeULEB128(stream)
		if numOfReturns > 0 {
			typ = stream.ReadByte()
			entry.ReturnType = W2J_LANGUAGE_TYPES[typ]
		}

		typSec.Entries = append(typSec.Entries, entry)
	}

	return typSec
}

func (s sectionParsers) _import(stream *Stream) ImportSec {
	numberOfEntries := DecodeULEB128(stream)
	importSec := ImportSec{
		Name: "import",
	}

	rParser := reflect.ValueOf(typeParsers{})
	for i := uint64(0); i < numberOfEntries; i++ {
		moduleLen := DecodeULEB128(stream)
		moduleStr := stream.Read(int(moduleLen))

		fieldLen := DecodeULEB128(stream)
		fieldStr := make([]byte, fieldLen)

		kind := stream.ReadByte()
		externalKind := W2J_EXTERNAL_KIND[kind]
		returned := rParser.MethodByName(externalKind).Call([]reflect.Value{reflect.ValueOf(stream)})

		entry := ImportEntry{
			ModuleStr: string(moduleStr),
			FieldStr:  string(fieldStr),
			Kind:      externalKind,
			Type:      returned[0].Interface(),
		}

		importSec.Entries = append(importSec.Entries, entry)
	}

	return importSec
}

func (sectionParsers) function(stream *Stream) FuncSec {
	numberOfEntries := DecodeULEB128(stream)
	funcSec := FuncSec{
		Name: "function",
	}

	for i := uint64(0); i < numberOfEntries; i++ {
		entry := DecodeULEB128(stream)
		funcSec.Entries = append(funcSec.Entries, entry)
	}
	return funcSec
}

func (s sectionParsers) table(stream *Stream) TableSec {
	numberOfEntries := DecodeULEB128(stream)
	tableSec := TableSec{
		Name: "table",
	}

	tparser := typeParsers{}
	for i := uint64(0); i < numberOfEntries; i++ {
		entry := tparser.table(stream)
		tableSec.Entries = append(tableSec.Entries, entry)
	}

	return tableSec
}

func (sectionParsers) memory(stream *Stream) MemSec {
	numberOfEntries := DecodeULEB128(stream)
	memSec := MemSec{
		Name: "memory",
	}

	tparser := typeParsers{}
	for i := uint64(0); i < numberOfEntries; i++ {
		entry := tparser.memory(stream)
		memSec.Entries = append(memSec.Entries, entry)
	}
	return memSec
}

func (sectionParsers) global(stream *Stream) GlobalSec {
	numberOfEntries := DecodeULEB128(stream)
	globalSec := GlobalSec{
		Name: "global",
	}

	tparser := typeParsers{}
	for i := uint64(0); i < numberOfEntries; i++ {
		entry := GlobalEntry{
			Type: tparser.global(stream),
			Init: tparser.initExpr(stream),
		}

		globalSec.Entries = append(globalSec.Entries, entry)
	}

	return globalSec
}

func (sectionParsers) export(stream *Stream) ExportSec {
	numberOfEntries := DecodeULEB128(stream)
	exportSec := ExportSec{
		Name: "export",
	}

	for i := uint64(0); i < numberOfEntries; i++ {
		strLength := DecodeULEB128(stream)
		fieldStr := string(stream.Read(int(strLength)))
		kind := stream.ReadByte()
		index := DecodeULEB128(stream)

		entry := ExportEntry{
			FieldStr: fieldStr,
			Kind:     W2J_EXTERNAL_KIND[kind],
			Index:    uint32(index),
		}

		exportSec.Entries = append(exportSec.Entries, entry)
	}

	return exportSec
}

func (sectionParsers) start(stream *Stream) StartSec {
	startSec := StartSec{
		Name:  "start",
		Index: uint32(DecodeULEB128(stream)),
	}
	return startSec
}

func (sectionParsers) element(stream *Stream) ElementSec {
	numberOfEntries := DecodeULEB128(stream)
	elSec := ElementSec{
		Name: "element",
	}

	tparser := typeParsers{}
	for i := uint64(0); i < numberOfEntries; i++ {
		entry := ElementEntry{}
		entry.Index = uint32(DecodeULEB128(stream))
		entry.Offset = tparser.initExpr(stream)

		numElem := DecodeULEB128(stream)
		for j := uint64(0); j < numElem; j++ {
			elem := DecodeULEB128(stream)
			entry.Elements = append(entry.Elements, elem)
		}

		elSec.Entries = append(elSec.Entries, entry)
	}

	return elSec
}

func (sectionParsers) code(stream *Stream) CodeSec {
	numberOfEntries := DecodeULEB128(stream)
	codeSec := CodeSec{
		Name: "code",
	}

	for i := uint64(0); i < numberOfEntries; i++ {
		codeBody := CodeBody{}

		bodySize := DecodeULEB128(stream)
		endBytes := stream.bytesRead + int(bodySize)

		// parse locals
		localCount := DecodeULEB128(stream)
		for j := uint64(0); j < localCount; j++ {
			local := LocalEntry{}
			local.Count = uint32(DecodeULEB128(stream))
			local.Type = W2J_LANGUAGE_TYPES[stream.ReadByte()]
			codeBody.Locals = append(codeBody.Locals, local)
		}

		// parse code
		for stream.bytesRead < endBytes {
			op := ParseOp(stream)
			codeBody.Code = append(codeBody.Code, op)
		}

		codeSec.Entries = append(codeSec.Entries, codeBody)
	}

	return codeSec
}

func (sectionParsers) data(stream *Stream) DataSec {
	numberOfEntries := DecodeULEB128(stream)
	dataSec := DataSec{}

	tparser := typeParsers{}
	for i := uint64(0); i < numberOfEntries; i++ {
		entry := DataSegment{}
		entry.Index = uint32(DecodeULEB128(stream))
		entry.Offset = tparser.initExpr(stream)
		segmentSize := DecodeULEB128(stream)
		entry.Data = append([]byte{}, stream.Read(int(segmentSize))...)

		dataSec.Entries = append(dataSec.Entries, entry)
	}

	return dataSec
}

func Parse(buf []byte) []JSON {
	stream := NewStream(buf)
	preramble := ParsePreramble(stream)
	resJson := []JSON{preramble}

	rSecParsers := reflect.ValueOf(sectionParsers{})
	for stream.Len() != 0 {
		header := ParseSectionHeader(stream)
		name := header.Name
		switch name {
		case "type":
			name = "_type"
		case "import":
			name = "_import"
		}
		parser := rSecParsers.MethodByName(name)
		in := []reflect.Value{reflect.ValueOf(stream)}
		if parser.Type().NumIn() == 2 {
			in = append(in, reflect.ValueOf(header))
		}
		rsec := parser.Call(in)[0]

		// convert to JSON
		jsonObj := make(JSON)
		rtSec := rsec.Type()
		for i := 0; i < rsec.NumField(); i++ {
			jsonObj[rtSec.Field(i).Name] = rsec.Field(i).Interface()
		}
		resJson = append(resJson, jsonObj)
	}

	return resJson
}

func ParsePreramble(stream *Stream) JSON {
	magic := stream.Read(4)
	version := stream.Read(4)

	jsonObj := make(JSON)
	jsonObj["Name"] = "preramble"
	jsonObj["Magic"] = magic
	jsonObj["Version"] = version

	return jsonObj
}

func ParseSectionHeader(stream *Stream) SectionHeader {
	id := stream.ReadByte()
	return SectionHeader{
		Id:   id,
		Name: SECTION_IDS[id],
		Size: DecodeULEB128(stream),
	}
}

func ParseOp(stream *Stream) OP {
	finalOP := OP{}
	op := stream.ReadByte()
	fullName := strings.Split(W2J_OPCODES[op], ".")
	var (
		typ           = fullName[0]
		name          string
		immediatesKey string
	)

	if len(fullName) < 2 {
		name = typ
	} else {
		finalOP.ReturnType = typ
	}

	finalOP.Name = name

	if name == "const" {
		immediatesKey = typ
	} else {
		immediatesKey = name
	}
	immediates := W2J_OP_IMMEDIATES[immediatesKey]
	if immediates != nil {
		rv := reflect.ValueOf(immediataryParsers{})
		rStream := reflect.ValueOf(stream)
		returned := rv.MethodByName(immediates.(string)).Call([]reflect.Value{rStream})
		finalOP.Immediates = returned[0].Interface()
	}

	return finalOP
}
