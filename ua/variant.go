// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"fmt"
	"time"
)

// These flags define the size and dimension of a Variant value.
const (
	VariantArrayDimensions = 0x40
	VariantArrayValues     = 0x80
)

// Variant is a union of the built-in types.
//
// Specification: Part 6, 5.2.2.16
type Variant struct {
	// EncodingMask contains the type and the array flags
	// bits 0:5: built-in type id 1-25
	// bit 6: array dimensions
	// bit 7: array values
	EncodingMask byte

	// ArrayLength is the number of elements in the array.
	// This field is only present if the 'array values'
	// flag is set.
	//
	// Multi-dimensional arrays are encoded as a one-dimensional array and this
	// field specifies the total number of elements. The original array can be
	// reconstructed from the dimensions that are encoded after the value
	// field.
	ArrayLength int32

	// ArrayDimensionsLength is the numer of dimensions.
	// This field is only present if the 'array dimensions' flag
	// is set.
	ArrayDimensionsLength int32

	// ArrayDimensions is the size for each dimension.
	// This field is only present if the 'array dimensions' flag
	// is set.
	ArrayDimensions []int32

	Value interface{}
}

func NewVariant(v interface{}) (*Variant, error) {
	va := &Variant{}
	if err := va.Set(v); err != nil {
		return nil, err
	}
	return va, nil
}

func MustVariant(v interface{}) *Variant {
	va, err := NewVariant(v)
	if err != nil {
		panic(err)
	}
	return va
}

func (m *Variant) Type() Type {
	return Type(m.EncodingMask & 0x3f)
}

func (m *Variant) SetType(t Type) {
	m.EncodingMask = byte(t & 0x3f)
}

func (m *Variant) Has(mask byte) bool {
	return m.EncodingMask&mask == mask
}

func (m *Variant) Decode(b []byte) (int, error) {
	buf := NewBuffer(b)

	m.EncodingMask = buf.ReadByte()

	elems := 1
	if m.Has(VariantArrayValues) {
		m.ArrayLength = buf.ReadInt32()
		elems = int(m.ArrayLength)
	}

	values := make([]interface{}, elems)
	for i := 0; i < elems; i++ {
		switch m.Type() {
		case TypeBoolean:
			values[i] = buf.ReadBool()
		case TypeSByte:
			values[i] = buf.ReadInt8()
		case TypeByte:
			values[i] = buf.ReadByte()
		case TypeInt16:
			values[i] = buf.ReadInt16()
		case TypeUint16:
			values[i] = buf.ReadUint16()
		case TypeInt32:
			values[i] = buf.ReadInt32()
		case TypeUint32:
			values[i] = buf.ReadUint32()
		case TypeInt64:
			values[i] = buf.ReadInt64()
		case TypeUint64:
			values[i] = buf.ReadUint64()
		case TypeFloat:
			values[i] = buf.ReadFloat32()
		case TypeDouble:
			values[i] = buf.ReadFloat64()
		case TypeString:
			values[i] = buf.ReadString()
		case TypeDateTime:
			values[i] = buf.ReadTime()
		case TypeGUID:
			v := new(GUID)
			buf.ReadStruct(v)
			values[i] = v
		case TypeByteString:
			values[i] = buf.ReadBytes()
		case TypeXMLElement:
			values[i] = XmlElement(buf.ReadString())
		case TypeNodeID:
			v := new(NodeID)
			buf.ReadStruct(v)
			values[i] = v
		case TypeExpandedNodeID:
			v := new(ExpandedNodeID)
			buf.ReadStruct(v)
			values[i] = v
		case TypeStatusCode:
			values[i] = StatusCode(buf.ReadUint32())
		case TypeQualifiedName:
			v := new(QualifiedName)
			buf.ReadStruct(v)
			values[i] = v
		case TypeLocalizedText:
			v := new(LocalizedText)
			buf.ReadStruct(v)
			values[i] = v
		case TypeExtensionObject:
			v := new(ExtensionObject)
			buf.ReadStruct(v)
			values[i] = v
		case TypeDataValue:
			v := new(DataValue)
			buf.ReadStruct(v)
			values[i] = v
		case TypeVariant:
			// todo(fs): limit recursion depth to 100
			v := new(Variant)
			buf.ReadStruct(v)
			values[i] = v
		case TypeDiagnosticInfo:
			// todo(fs): limit recursion depth to 100
			v := new(DiagnosticInfo)
			buf.ReadStruct(v)
			values[i] = v
		}
	}

	if m.Has(VariantArrayDimensions) {
		m.ArrayDimensionsLength = buf.ReadInt32()
		m.ArrayDimensions = make([]int32, m.ArrayDimensionsLength)
		for i := 0; i < int(m.ArrayDimensionsLength); i++ {
			m.ArrayDimensions[i] = buf.ReadInt32()
		}
	}

	m.Value = values
	if elems == 1 {
		m.Value = values[0]
	}

	return buf.Pos(), buf.Error()
}

func (m *Variant) Encode() ([]byte, error) {
	buf := NewBuffer(nil)

	buf.WriteByte(m.EncodingMask)

	if m.Has(VariantArrayValues) {
		buf.WriteInt32(m.ArrayLength)
	}

	switch v := m.Value.(type) {
	case bool:
		buf.WriteBool(v)
	case int8:
		buf.WriteInt8(v)
	case byte:
		buf.WriteByte(v)
	case int16:
		buf.WriteInt16(v)
	case uint16:
		buf.WriteUint16(v)
	case int32:
		buf.WriteInt32(v)
	case uint32:
		buf.WriteUint32(v)
	case int64:
		buf.WriteInt64(v)
	case uint64:
		buf.WriteUint64(v)
	case float32:
		buf.WriteFloat32(v)
	case float64:
		buf.WriteFloat64(v)
	case string:
		buf.WriteString(v)
	case time.Time:
		buf.WriteTime(v)
	case *GUID:
		buf.WriteStruct(v)
	case []byte:
		buf.WriteByteString(v)
	case XmlElement:
		buf.WriteString(string(v))
	case *NodeID:
		buf.WriteStruct(v)
	case *ExpandedNodeID:
		buf.WriteStruct(v)
	case StatusCode:
		buf.WriteUint32(uint32(v))
	case *QualifiedName:
		buf.WriteStruct(v)
	case *LocalizedText:
		buf.WriteStruct(v)
	case *ExtensionObject:
		buf.WriteStruct(v)
	case *DataValue:
		buf.WriteStruct(v)
	case *Variant:
		buf.WriteStruct(v)
	case *DiagnosticInfo:
		buf.WriteStruct(v)
	}

	if m.Has(VariantArrayDimensions) {
		buf.WriteInt32(m.ArrayDimensionsLength)
		for i := 0; i < int(m.ArrayDimensionsLength); i++ {
			buf.WriteInt32(m.ArrayDimensions[i])
		}
	}

	return buf.Bytes(), buf.Error()
}

func (m *Variant) Set(v interface{}) error {
	switch v.(type) {
	case bool:
		m.SetType(TypeBoolean)
	case int8:
		m.SetType(TypeSByte)
	case byte:
		m.SetType(TypeByte)
	case int16:
		m.SetType(TypeInt16)
	case uint16:
		m.SetType(TypeUint16)
	case int32:
		m.SetType(TypeInt32)
	case uint32:
		m.SetType(TypeUint32)
	case int64:
		m.SetType(TypeInt64)
	case uint64:
		m.SetType(TypeUint64)
	case float32:
		m.SetType(TypeFloat)
	case float64:
		m.SetType(TypeDouble)
	case string:
		m.SetType(TypeString)
	case time.Time:
		m.SetType(TypeDateTime)
	case *GUID:
		m.SetType(TypeGUID)
	case []byte:
		m.SetType(TypeByteString)
	case XmlElement:
		m.SetType(TypeXMLElement)
	case *NodeID:
		m.SetType(TypeNodeID)
	case *ExpandedNodeID:
		m.SetType(TypeExpandedNodeID)
	case StatusCode:
		m.SetType(TypeStatusCode)
	case *QualifiedName:
		m.SetType(TypeQualifiedName)
	case *LocalizedText:
		m.SetType(TypeLocalizedText)
	case *ExtensionObject:
		m.SetType(TypeExtensionObject)
	case *DataValue:
		m.SetType(TypeDataValue)
	case *Variant:
		m.SetType(TypeVariant)
	case *DiagnosticInfo:
		m.SetType(TypeDiagnosticInfo)
	default:
		return fmt.Errorf("opcua: cannot set variant to %T", v)
	}
	m.Value = v
	return nil
}

func (m *Variant) String() string {
	switch m.Type() {
	case TypeString:
		return m.Value.(string)
	case TypeLocalizedText:
		return m.Value.(*LocalizedText).Text
	case TypeQualifiedName:
		return m.Value.(*QualifiedName).Name
	default:
		return fmt.Sprintf("%v", m.Value)
	}
}

func (m *Variant) Bool() bool {
	switch m.Type() {
	case TypeBoolean:
		return m.Value.(bool)
	default:
		return false
	}
}

func (m *Variant) Float() float64 {
	switch m.Type() {
	case TypeFloat:
		return float64(m.Value.(float32))
	case TypeDouble:
		return m.Value.(float64)
	default:
		return 0
	}
}

func (m *Variant) Int() int64 {
	switch m.Type() {
	case TypeSByte:
		return int64(m.Value.(int8))
	case TypeUint16:
		return int64(m.Value.(int16))
	case TypeUint32:
		return int64(m.Value.(int32))
	case TypeUint64:
		return m.Value.(int64)
	default:
		return 0
	}
}

func (m *Variant) Uint() uint64 {
	switch m.Type() {
	case TypeByte:
		return uint64(m.Value.(byte))
	case TypeUint16:
		return uint64(m.Value.(uint16))
	case TypeUint32:
		return uint64(m.Value.(uint32))
	case TypeUint64:
		return m.Value.(uint64)
	default:
		return 0
	}
}

func (m *Variant) Time() time.Time {
	switch m.Type() {
	case TypeDateTime:
		return m.Value.(time.Time)
	default:
		return time.Time{}
	}
}
