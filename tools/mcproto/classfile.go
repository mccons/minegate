package main

import (
	"encoding/binary"
	"fmt"
)

const (
	constantUtf8        = 1
	constantInteger     = 3
	constantFloat       = 4
	constantLong        = 5
	constantDouble      = 6
	constantClass       = 7
	constantString      = 8
	constantFieldref    = 9
	constantMethodref   = 10
	constantIMethodref  = 11
	constantNameAndType = 12
	constantMethodHandle = 15
	constantMethodType  = 16
	constantInvokeDynamic = 18
)

type cpEntry struct {
	tag     byte
	info    []byte
	parsed  interface{}
}

type javaClass struct {
	minorVersion uint16
	majorVersion uint16
	constantPool []cpEntry
	thisClass    uint16
	superClass   uint16
	interfaces   []uint16
	fields       []fieldInfo
	methods      []methodInfo
}

type fieldInfo struct {
	accessFlags uint16
	nameIdx     uint16
	descIdx     uint16
}

type methodInfo struct {
	accessFlags uint16
	nameIdx     uint16
	descIdx     uint16
}

func parseClass(data []byte) (*javaClass, error) {
	r := &classReader{data: data, off: 0}

	magic := r.u4()
	if magic != 0xCAFEBABE {
		return nil, fmt.Errorf("bad magic: 0x%08X", magic)
	}

	jc := &javaClass{
		minorVersion: r.u2(),
		majorVersion: r.u2(),
	}

	cpCount := r.u2()
	jc.constantPool = make([]cpEntry, cpCount)

	for i := 1; i < int(cpCount); i++ {
		tag := r.u1()
		entry := cpEntry{tag: tag}
		switch tag {
		case constantUtf8:
			length := r.u2()
			entry.info = r.bytes(int(length))
			entry.parsed = string(entry.info)
		case constantInteger, constantFloat:
			entry.info = r.bytes(4)
			if tag == constantInteger {
				entry.parsed = int32(binary.BigEndian.Uint32(entry.info))
			}
		case constantLong, constantDouble:
			entry.info = r.bytes(8)
			i++ // longs and doubles take two constant pool slots
		case constantClass, constantString, constantMethodType:
			entry.info = r.bytes(2)
		case constantFieldref, constantMethodref, constantIMethodref, constantNameAndType, constantMethodHandle, constantInvokeDynamic:
			if tag == constantMethodHandle {
				entry.info = r.bytes(3)
			} else {
				entry.info = r.bytes(4)
			}
		default:
			return nil, fmt.Errorf("unknown constant pool tag %d at offset %d", tag, r.off-1)
		}
		jc.constantPool[i] = entry
	}

	jc.thisClass = r.u2()
	jc.superClass = r.u2()
	intCount := r.u2()
	jc.interfaces = make([]uint16, intCount)
	for i := range jc.interfaces {
		jc.interfaces[i] = r.u2()
	}

	fieldCount := r.u2()
	jc.fields = make([]fieldInfo, fieldCount)
	for i := range jc.fields {
		jc.fields[i] = fieldInfo{
			accessFlags: r.u2(),
			nameIdx:     r.u2(),
			descIdx:     r.u2(),
		}
		attrCount := r.u2()
		for j := 0; j < int(attrCount); j++ {
			r.u2()
			r.bytes(int(r.u4()))
		}
	}

	methodCount := r.u2()
	jc.methods = make([]methodInfo, methodCount)
	for i := range jc.methods {
		jc.methods[i] = methodInfo{
			accessFlags: r.u2(),
			nameIdx:     r.u2(),
			descIdx:     r.u2(),
		}
		attrCount := r.u2()
		for j := 0; j < int(attrCount); j++ {
			r.u2()
			r.bytes(int(r.u4()))
		}
	}

	return jc, nil
}

func (jc *javaClass) className(cpIdx uint16) string {
	if int(cpIdx) >= len(jc.constantPool) {
		return "?"
	}
	entry := jc.constantPool[cpIdx]
	if entry.tag != constantClass {
		return "?"
	}
	nameIdx := binary.BigEndian.Uint16(entry.info)
	return jc.utf8(nameIdx)
}

func (jc *javaClass) utf8(idx uint16) string {
	if int(idx) >= len(jc.constantPool) {
		return "?"
	}
	entry := jc.constantPool[idx]
	if entry.tag != constantUtf8 {
		return "?"
	}
	return entry.parsed.(string)
}

func (jc *javaClass) integerConstants() []int32 {
	var vals []int32
	for _, e := range jc.constantPool {
		if e.tag == constantInteger {
			if v, ok := e.parsed.(int32); ok {
				vals = append(vals, v)
			}
		}
	}
	return vals
}

type classReader struct {
	data []byte
	off  int
}

func (r *classReader) u1() byte {
	b := r.data[r.off]
	r.off++
	return b
}

func (r *classReader) u2() uint16 {
	v := binary.BigEndian.Uint16(r.data[r.off:])
	r.off += 2
	return v
}

func (r *classReader) u4() uint32 {
	v := binary.BigEndian.Uint32(r.data[r.off:])
	r.off += 4
	return v
}

func (r *classReader) bytes(n int) []byte {
	b := r.data[r.off : r.off+n]
	r.off += n
	return b
}


