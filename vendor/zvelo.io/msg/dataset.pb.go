// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: zvelo/msg/dataset.proto

package msg

import proto "github.com/gogo/protobuf/proto"
import fmt "fmt"
import math "math"

import strconv "strconv"

import strings "strings"
import reflect "reflect"

import io "io"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

type DataSetType int32

const (
	CATEGORIZATION DataSetType = 0
	// 1 is reserved
	// 2 is reserved
	// 3 is reserved
	MALICIOUS DataSetType = 4
	ECHO      DataSetType = 5
)

var DataSetType_name = map[int32]string{
	0: "CATEGORIZATION",
	4: "MALICIOUS",
	5: "ECHO",
}
var DataSetType_value = map[string]int32{
	"CATEGORIZATION": 0,
	"MALICIOUS":      4,
	"ECHO":           5,
}

func (DataSetType) EnumDescriptor() ([]byte, []int) { return fileDescriptorDataset, []int{0} }

type DataSet_Malicious_Verdict int32

const (
	VERDICT_UNKNOWN   DataSet_Malicious_Verdict = 0
	VERDICT_CLEAN     DataSet_Malicious_Verdict = 1
	VERDICT_MALICIOUS DataSet_Malicious_Verdict = 2
)

var DataSet_Malicious_Verdict_name = map[int32]string{
	0: "VERDICT_UNKNOWN",
	1: "VERDICT_CLEAN",
	2: "VERDICT_MALICIOUS",
}
var DataSet_Malicious_Verdict_value = map[string]int32{
	"VERDICT_UNKNOWN":   0,
	"VERDICT_CLEAN":     1,
	"VERDICT_MALICIOUS": 2,
}

func (DataSet_Malicious_Verdict) EnumDescriptor() ([]byte, []int) {
	return fileDescriptorDataset, []int{0, 1, 0}
}

// DataSet
type DataSet struct {
	Categorization *DataSet_Categorization `protobuf:"bytes,1,opt,name=categorization" json:"categorization,omitempty"`
	Malicious      *DataSet_Malicious      `protobuf:"bytes,5,opt,name=malicious" json:"malicious,omitempty"`
	Echo           *DataSet_Echo           `protobuf:"bytes,6,opt,name=echo" json:"echo,omitempty"`
}

func (m *DataSet) Reset()                    { *m = DataSet{} }
func (*DataSet) ProtoMessage()               {}
func (*DataSet) Descriptor() ([]byte, []int) { return fileDescriptorDataset, []int{0} }

func (m *DataSet) GetCategorization() *DataSet_Categorization {
	if m != nil {
		return m.Categorization
	}
	return nil
}

func (m *DataSet) GetMalicious() *DataSet_Malicious {
	if m != nil {
		return m.Malicious
	}
	return nil
}

func (m *DataSet) GetEcho() *DataSet_Echo {
	if m != nil {
		return m.Echo
	}
	return nil
}

// Categorization
type DataSet_Categorization struct {
	Value []uint32 `protobuf:"varint,2,rep,packed,name=value" json:"value,omitempty"`
}

func (m *DataSet_Categorization) Reset()                    { *m = DataSet_Categorization{} }
func (*DataSet_Categorization) ProtoMessage()               {}
func (*DataSet_Categorization) Descriptor() ([]byte, []int) { return fileDescriptorDataset, []int{0, 0} }

func (m *DataSet_Categorization) GetValue() []uint32 {
	if m != nil {
		return m.Value
	}
	return nil
}

// Malicious
type DataSet_Malicious struct {
	Category uint32 `protobuf:"varint,4,opt,name=category,proto3" json:"category,omitempty"`
	Verdict  uint32 `protobuf:"varint,5,opt,name=verdict,proto3" json:"verdict,omitempty"`
}

func (m *DataSet_Malicious) Reset()                    { *m = DataSet_Malicious{} }
func (*DataSet_Malicious) ProtoMessage()               {}
func (*DataSet_Malicious) Descriptor() ([]byte, []int) { return fileDescriptorDataset, []int{0, 1} }

func (m *DataSet_Malicious) GetCategory() uint32 {
	if m != nil {
		return m.Category
	}
	return 0
}

func (m *DataSet_Malicious) GetVerdict() uint32 {
	if m != nil {
		return m.Verdict
	}
	return 0
}

// Echo
type DataSet_Echo struct {
	Url string `protobuf:"bytes,1,opt,name=url,proto3" json:"url,omitempty"`
}

func (m *DataSet_Echo) Reset()                    { *m = DataSet_Echo{} }
func (*DataSet_Echo) ProtoMessage()               {}
func (*DataSet_Echo) Descriptor() ([]byte, []int) { return fileDescriptorDataset, []int{0, 2} }

func (m *DataSet_Echo) GetUrl() string {
	if m != nil {
		return m.Url
	}
	return ""
}

func init() {
	proto.RegisterType((*DataSet)(nil), "zvelo.msg.DataSet")
	proto.RegisterType((*DataSet_Categorization)(nil), "zvelo.msg.DataSet.Categorization")
	proto.RegisterType((*DataSet_Malicious)(nil), "zvelo.msg.DataSet.Malicious")
	proto.RegisterType((*DataSet_Echo)(nil), "zvelo.msg.DataSet.Echo")
	proto.RegisterEnum("zvelo.msg.DataSetType", DataSetType_name, DataSetType_value)
	proto.RegisterEnum("zvelo.msg.DataSet_Malicious_Verdict", DataSet_Malicious_Verdict_name, DataSet_Malicious_Verdict_value)
}
func (x DataSetType) String() string {
	s, ok := DataSetType_name[int32(x)]
	if ok {
		return s
	}
	return strconv.Itoa(int(x))
}
func (x DataSet_Malicious_Verdict) String() string {
	s, ok := DataSet_Malicious_Verdict_name[int32(x)]
	if ok {
		return s
	}
	return strconv.Itoa(int(x))
}
func (this *DataSet) VerboseEqual(that interface{}) error {
	if that == nil {
		if this == nil {
			return nil
		}
		return fmt.Errorf("that == nil && this != nil")
	}

	that1, ok := that.(*DataSet)
	if !ok {
		that2, ok := that.(DataSet)
		if ok {
			that1 = &that2
		} else {
			return fmt.Errorf("that is not of type *DataSet")
		}
	}
	if that1 == nil {
		if this == nil {
			return nil
		}
		return fmt.Errorf("that is type *DataSet but is nil && this != nil")
	} else if this == nil {
		return fmt.Errorf("that is type *DataSet but is not nil && this == nil")
	}
	if !this.Categorization.Equal(that1.Categorization) {
		return fmt.Errorf("Categorization this(%v) Not Equal that(%v)", this.Categorization, that1.Categorization)
	}
	if !this.Malicious.Equal(that1.Malicious) {
		return fmt.Errorf("Malicious this(%v) Not Equal that(%v)", this.Malicious, that1.Malicious)
	}
	if !this.Echo.Equal(that1.Echo) {
		return fmt.Errorf("Echo this(%v) Not Equal that(%v)", this.Echo, that1.Echo)
	}
	return nil
}
func (this *DataSet) Equal(that interface{}) bool {
	if that == nil {
		if this == nil {
			return true
		}
		return false
	}

	that1, ok := that.(*DataSet)
	if !ok {
		that2, ok := that.(DataSet)
		if ok {
			that1 = &that2
		} else {
			return false
		}
	}
	if that1 == nil {
		if this == nil {
			return true
		}
		return false
	} else if this == nil {
		return false
	}
	if !this.Categorization.Equal(that1.Categorization) {
		return false
	}
	if !this.Malicious.Equal(that1.Malicious) {
		return false
	}
	if !this.Echo.Equal(that1.Echo) {
		return false
	}
	return true
}
func (this *DataSet_Categorization) VerboseEqual(that interface{}) error {
	if that == nil {
		if this == nil {
			return nil
		}
		return fmt.Errorf("that == nil && this != nil")
	}

	that1, ok := that.(*DataSet_Categorization)
	if !ok {
		that2, ok := that.(DataSet_Categorization)
		if ok {
			that1 = &that2
		} else {
			return fmt.Errorf("that is not of type *DataSet_Categorization")
		}
	}
	if that1 == nil {
		if this == nil {
			return nil
		}
		return fmt.Errorf("that is type *DataSet_Categorization but is nil && this != nil")
	} else if this == nil {
		return fmt.Errorf("that is type *DataSet_Categorization but is not nil && this == nil")
	}
	if len(this.Value) != len(that1.Value) {
		return fmt.Errorf("Value this(%v) Not Equal that(%v)", len(this.Value), len(that1.Value))
	}
	for i := range this.Value {
		if this.Value[i] != that1.Value[i] {
			return fmt.Errorf("Value this[%v](%v) Not Equal that[%v](%v)", i, this.Value[i], i, that1.Value[i])
		}
	}
	return nil
}
func (this *DataSet_Categorization) Equal(that interface{}) bool {
	if that == nil {
		if this == nil {
			return true
		}
		return false
	}

	that1, ok := that.(*DataSet_Categorization)
	if !ok {
		that2, ok := that.(DataSet_Categorization)
		if ok {
			that1 = &that2
		} else {
			return false
		}
	}
	if that1 == nil {
		if this == nil {
			return true
		}
		return false
	} else if this == nil {
		return false
	}
	if len(this.Value) != len(that1.Value) {
		return false
	}
	for i := range this.Value {
		if this.Value[i] != that1.Value[i] {
			return false
		}
	}
	return true
}
func (this *DataSet_Malicious) VerboseEqual(that interface{}) error {
	if that == nil {
		if this == nil {
			return nil
		}
		return fmt.Errorf("that == nil && this != nil")
	}

	that1, ok := that.(*DataSet_Malicious)
	if !ok {
		that2, ok := that.(DataSet_Malicious)
		if ok {
			that1 = &that2
		} else {
			return fmt.Errorf("that is not of type *DataSet_Malicious")
		}
	}
	if that1 == nil {
		if this == nil {
			return nil
		}
		return fmt.Errorf("that is type *DataSet_Malicious but is nil && this != nil")
	} else if this == nil {
		return fmt.Errorf("that is type *DataSet_Malicious but is not nil && this == nil")
	}
	if this.Category != that1.Category {
		return fmt.Errorf("Category this(%v) Not Equal that(%v)", this.Category, that1.Category)
	}
	if this.Verdict != that1.Verdict {
		return fmt.Errorf("Verdict this(%v) Not Equal that(%v)", this.Verdict, that1.Verdict)
	}
	return nil
}
func (this *DataSet_Malicious) Equal(that interface{}) bool {
	if that == nil {
		if this == nil {
			return true
		}
		return false
	}

	that1, ok := that.(*DataSet_Malicious)
	if !ok {
		that2, ok := that.(DataSet_Malicious)
		if ok {
			that1 = &that2
		} else {
			return false
		}
	}
	if that1 == nil {
		if this == nil {
			return true
		}
		return false
	} else if this == nil {
		return false
	}
	if this.Category != that1.Category {
		return false
	}
	if this.Verdict != that1.Verdict {
		return false
	}
	return true
}
func (this *DataSet_Echo) VerboseEqual(that interface{}) error {
	if that == nil {
		if this == nil {
			return nil
		}
		return fmt.Errorf("that == nil && this != nil")
	}

	that1, ok := that.(*DataSet_Echo)
	if !ok {
		that2, ok := that.(DataSet_Echo)
		if ok {
			that1 = &that2
		} else {
			return fmt.Errorf("that is not of type *DataSet_Echo")
		}
	}
	if that1 == nil {
		if this == nil {
			return nil
		}
		return fmt.Errorf("that is type *DataSet_Echo but is nil && this != nil")
	} else if this == nil {
		return fmt.Errorf("that is type *DataSet_Echo but is not nil && this == nil")
	}
	if this.Url != that1.Url {
		return fmt.Errorf("Url this(%v) Not Equal that(%v)", this.Url, that1.Url)
	}
	return nil
}
func (this *DataSet_Echo) Equal(that interface{}) bool {
	if that == nil {
		if this == nil {
			return true
		}
		return false
	}

	that1, ok := that.(*DataSet_Echo)
	if !ok {
		that2, ok := that.(DataSet_Echo)
		if ok {
			that1 = &that2
		} else {
			return false
		}
	}
	if that1 == nil {
		if this == nil {
			return true
		}
		return false
	} else if this == nil {
		return false
	}
	if this.Url != that1.Url {
		return false
	}
	return true
}
func (this *DataSet) GoString() string {
	if this == nil {
		return "nil"
	}
	s := make([]string, 0, 7)
	s = append(s, "&msg.DataSet{")
	if this.Categorization != nil {
		s = append(s, "Categorization: "+fmt.Sprintf("%#v", this.Categorization)+",\n")
	}
	if this.Malicious != nil {
		s = append(s, "Malicious: "+fmt.Sprintf("%#v", this.Malicious)+",\n")
	}
	if this.Echo != nil {
		s = append(s, "Echo: "+fmt.Sprintf("%#v", this.Echo)+",\n")
	}
	s = append(s, "}")
	return strings.Join(s, "")
}
func (this *DataSet_Categorization) GoString() string {
	if this == nil {
		return "nil"
	}
	s := make([]string, 0, 5)
	s = append(s, "&msg.DataSet_Categorization{")
	s = append(s, "Value: "+fmt.Sprintf("%#v", this.Value)+",\n")
	s = append(s, "}")
	return strings.Join(s, "")
}
func (this *DataSet_Malicious) GoString() string {
	if this == nil {
		return "nil"
	}
	s := make([]string, 0, 6)
	s = append(s, "&msg.DataSet_Malicious{")
	s = append(s, "Category: "+fmt.Sprintf("%#v", this.Category)+",\n")
	s = append(s, "Verdict: "+fmt.Sprintf("%#v", this.Verdict)+",\n")
	s = append(s, "}")
	return strings.Join(s, "")
}
func (this *DataSet_Echo) GoString() string {
	if this == nil {
		return "nil"
	}
	s := make([]string, 0, 5)
	s = append(s, "&msg.DataSet_Echo{")
	s = append(s, "Url: "+fmt.Sprintf("%#v", this.Url)+",\n")
	s = append(s, "}")
	return strings.Join(s, "")
}
func valueToGoStringDataset(v interface{}, typ string) string {
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		return "nil"
	}
	pv := reflect.Indirect(rv).Interface()
	return fmt.Sprintf("func(v %v) *%v { return &v } ( %#v )", typ, typ, pv)
}
func (m *DataSet) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *DataSet) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if m.Categorization != nil {
		dAtA[i] = 0xa
		i++
		i = encodeVarintDataset(dAtA, i, uint64(m.Categorization.Size()))
		n1, err := m.Categorization.MarshalTo(dAtA[i:])
		if err != nil {
			return 0, err
		}
		i += n1
	}
	if m.Malicious != nil {
		dAtA[i] = 0x2a
		i++
		i = encodeVarintDataset(dAtA, i, uint64(m.Malicious.Size()))
		n2, err := m.Malicious.MarshalTo(dAtA[i:])
		if err != nil {
			return 0, err
		}
		i += n2
	}
	if m.Echo != nil {
		dAtA[i] = 0x32
		i++
		i = encodeVarintDataset(dAtA, i, uint64(m.Echo.Size()))
		n3, err := m.Echo.MarshalTo(dAtA[i:])
		if err != nil {
			return 0, err
		}
		i += n3
	}
	return i, nil
}

func (m *DataSet_Categorization) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *DataSet_Categorization) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.Value) > 0 {
		dAtA5 := make([]byte, len(m.Value)*10)
		var j4 int
		for _, num := range m.Value {
			for num >= 1<<7 {
				dAtA5[j4] = uint8(uint64(num)&0x7f | 0x80)
				num >>= 7
				j4++
			}
			dAtA5[j4] = uint8(num)
			j4++
		}
		dAtA[i] = 0x12
		i++
		i = encodeVarintDataset(dAtA, i, uint64(j4))
		i += copy(dAtA[i:], dAtA5[:j4])
	}
	return i, nil
}

func (m *DataSet_Malicious) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *DataSet_Malicious) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if m.Category != 0 {
		dAtA[i] = 0x20
		i++
		i = encodeVarintDataset(dAtA, i, uint64(m.Category))
	}
	if m.Verdict != 0 {
		dAtA[i] = 0x28
		i++
		i = encodeVarintDataset(dAtA, i, uint64(m.Verdict))
	}
	return i, nil
}

func (m *DataSet_Echo) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *DataSet_Echo) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.Url) > 0 {
		dAtA[i] = 0xa
		i++
		i = encodeVarintDataset(dAtA, i, uint64(len(m.Url)))
		i += copy(dAtA[i:], m.Url)
	}
	return i, nil
}

func encodeFixed64Dataset(dAtA []byte, offset int, v uint64) int {
	dAtA[offset] = uint8(v)
	dAtA[offset+1] = uint8(v >> 8)
	dAtA[offset+2] = uint8(v >> 16)
	dAtA[offset+3] = uint8(v >> 24)
	dAtA[offset+4] = uint8(v >> 32)
	dAtA[offset+5] = uint8(v >> 40)
	dAtA[offset+6] = uint8(v >> 48)
	dAtA[offset+7] = uint8(v >> 56)
	return offset + 8
}
func encodeFixed32Dataset(dAtA []byte, offset int, v uint32) int {
	dAtA[offset] = uint8(v)
	dAtA[offset+1] = uint8(v >> 8)
	dAtA[offset+2] = uint8(v >> 16)
	dAtA[offset+3] = uint8(v >> 24)
	return offset + 4
}
func encodeVarintDataset(dAtA []byte, offset int, v uint64) int {
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return offset + 1
}
func (m *DataSet) Size() (n int) {
	var l int
	_ = l
	if m.Categorization != nil {
		l = m.Categorization.Size()
		n += 1 + l + sovDataset(uint64(l))
	}
	if m.Malicious != nil {
		l = m.Malicious.Size()
		n += 1 + l + sovDataset(uint64(l))
	}
	if m.Echo != nil {
		l = m.Echo.Size()
		n += 1 + l + sovDataset(uint64(l))
	}
	return n
}

func (m *DataSet_Categorization) Size() (n int) {
	var l int
	_ = l
	if len(m.Value) > 0 {
		l = 0
		for _, e := range m.Value {
			l += sovDataset(uint64(e))
		}
		n += 1 + sovDataset(uint64(l)) + l
	}
	return n
}

func (m *DataSet_Malicious) Size() (n int) {
	var l int
	_ = l
	if m.Category != 0 {
		n += 1 + sovDataset(uint64(m.Category))
	}
	if m.Verdict != 0 {
		n += 1 + sovDataset(uint64(m.Verdict))
	}
	return n
}

func (m *DataSet_Echo) Size() (n int) {
	var l int
	_ = l
	l = len(m.Url)
	if l > 0 {
		n += 1 + l + sovDataset(uint64(l))
	}
	return n
}

func sovDataset(x uint64) (n int) {
	for {
		n++
		x >>= 7
		if x == 0 {
			break
		}
	}
	return n
}
func sozDataset(x uint64) (n int) {
	return sovDataset(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (this *DataSet) String() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&DataSet{`,
		`Categorization:` + strings.Replace(fmt.Sprintf("%v", this.Categorization), "DataSet_Categorization", "DataSet_Categorization", 1) + `,`,
		`Malicious:` + strings.Replace(fmt.Sprintf("%v", this.Malicious), "DataSet_Malicious", "DataSet_Malicious", 1) + `,`,
		`Echo:` + strings.Replace(fmt.Sprintf("%v", this.Echo), "DataSet_Echo", "DataSet_Echo", 1) + `,`,
		`}`,
	}, "")
	return s
}
func (this *DataSet_Categorization) String() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&DataSet_Categorization{`,
		`Value:` + fmt.Sprintf("%v", this.Value) + `,`,
		`}`,
	}, "")
	return s
}
func (this *DataSet_Malicious) String() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&DataSet_Malicious{`,
		`Category:` + fmt.Sprintf("%v", this.Category) + `,`,
		`Verdict:` + fmt.Sprintf("%v", this.Verdict) + `,`,
		`}`,
	}, "")
	return s
}
func (this *DataSet_Echo) String() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&DataSet_Echo{`,
		`Url:` + fmt.Sprintf("%v", this.Url) + `,`,
		`}`,
	}, "")
	return s
}
func valueToStringDataset(v interface{}) string {
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		return "nil"
	}
	pv := reflect.Indirect(rv).Interface()
	return fmt.Sprintf("*%v", pv)
}
func (m *DataSet) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowDataset
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: DataSet: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: DataSet: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Categorization", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowDataset
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthDataset
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Categorization == nil {
				m.Categorization = &DataSet_Categorization{}
			}
			if err := m.Categorization.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Malicious", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowDataset
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthDataset
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Malicious == nil {
				m.Malicious = &DataSet_Malicious{}
			}
			if err := m.Malicious.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Echo", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowDataset
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthDataset
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Echo == nil {
				m.Echo = &DataSet_Echo{}
			}
			if err := m.Echo.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipDataset(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthDataset
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *DataSet_Categorization) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowDataset
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Categorization: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Categorization: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 2:
			if wireType == 0 {
				var v uint32
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return ErrIntOverflowDataset
					}
					if iNdEx >= l {
						return io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					v |= (uint32(b) & 0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				m.Value = append(m.Value, v)
			} else if wireType == 2 {
				var packedLen int
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return ErrIntOverflowDataset
					}
					if iNdEx >= l {
						return io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					packedLen |= (int(b) & 0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				if packedLen < 0 {
					return ErrInvalidLengthDataset
				}
				postIndex := iNdEx + packedLen
				if postIndex > l {
					return io.ErrUnexpectedEOF
				}
				for iNdEx < postIndex {
					var v uint32
					for shift := uint(0); ; shift += 7 {
						if shift >= 64 {
							return ErrIntOverflowDataset
						}
						if iNdEx >= l {
							return io.ErrUnexpectedEOF
						}
						b := dAtA[iNdEx]
						iNdEx++
						v |= (uint32(b) & 0x7F) << shift
						if b < 0x80 {
							break
						}
					}
					m.Value = append(m.Value, v)
				}
			} else {
				return fmt.Errorf("proto: wrong wireType = %d for field Value", wireType)
			}
		default:
			iNdEx = preIndex
			skippy, err := skipDataset(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthDataset
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *DataSet_Malicious) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowDataset
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Malicious: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Malicious: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 4:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Category", wireType)
			}
			m.Category = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowDataset
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Category |= (uint32(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 5:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Verdict", wireType)
			}
			m.Verdict = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowDataset
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Verdict |= (uint32(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipDataset(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthDataset
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *DataSet_Echo) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowDataset
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Echo: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Echo: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Url", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowDataset
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthDataset
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Url = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipDataset(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthDataset
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipDataset(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowDataset
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowDataset
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
			return iNdEx, nil
		case 1:
			iNdEx += 8
			return iNdEx, nil
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowDataset
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			iNdEx += length
			if length < 0 {
				return 0, ErrInvalidLengthDataset
			}
			return iNdEx, nil
		case 3:
			for {
				var innerWire uint64
				var start int = iNdEx
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return 0, ErrIntOverflowDataset
					}
					if iNdEx >= l {
						return 0, io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					innerWire |= (uint64(b) & 0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				innerWireType := int(innerWire & 0x7)
				if innerWireType == 4 {
					break
				}
				next, err := skipDataset(dAtA[start:])
				if err != nil {
					return 0, err
				}
				iNdEx = start + next
			}
			return iNdEx, nil
		case 4:
			return iNdEx, nil
		case 5:
			iNdEx += 4
			return iNdEx, nil
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
	}
	panic("unreachable")
}

var (
	ErrInvalidLengthDataset = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowDataset   = fmt.Errorf("proto: integer overflow")
)

func init() { proto.RegisterFile("zvelo/msg/dataset.proto", fileDescriptorDataset) }

var fileDescriptorDataset = []byte{
	// 439 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x6c, 0x92, 0xcf, 0x6e, 0xd3, 0x40,
	0x10, 0xc6, 0xbd, 0xf1, 0xa6, 0x59, 0x4f, 0x71, 0xd8, 0x2e, 0xa0, 0x5a, 0x11, 0x5a, 0x85, 0x9e,
	0x22, 0x40, 0xae, 0x04, 0xb7, 0x1e, 0x90, 0x82, 0x6b, 0x51, 0x9b, 0x36, 0x96, 0xb6, 0x69, 0x91,
	0x7a, 0x41, 0x8b, 0xbb, 0x4a, 0x2d, 0x25, 0x6c, 0x95, 0x6c, 0x22, 0xb5, 0x27, 0x1e, 0x81, 0x17,
	0xe0, 0xce, 0x4b, 0x70, 0xe7, 0xd8, 0x23, 0xc7, 0xc6, 0x5c, 0x38, 0xf6, 0x11, 0x50, 0x36, 0x7f,
	0x4a, 0x51, 0x6f, 0x33, 0xe3, 0xdf, 0x37, 0xf3, 0x79, 0x76, 0x60, 0xf3, 0x72, 0xa2, 0xfa, 0x7a,
	0x7b, 0x30, 0xea, 0x6d, 0x9f, 0x4a, 0x23, 0x47, 0xca, 0x84, 0xe7, 0x43, 0x6d, 0x34, 0xf3, 0xec,
	0x87, 0x70, 0x30, 0xea, 0x6d, 0xfd, 0x70, 0xa1, 0xb6, 0x2b, 0x8d, 0x3c, 0x54, 0x86, 0x25, 0x50,
	0xcf, 0xa5, 0x51, 0x3d, 0x3d, 0x2c, 0x2e, 0xa5, 0x29, 0xf4, 0xe7, 0x00, 0x35, 0x51, 0x6b, 0xfd,
	0xd5, 0xb3, 0x70, 0xc5, 0x87, 0x0b, 0x36, 0x8c, 0xee, 0x80, 0xe2, 0x3f, 0x21, 0xdb, 0x01, 0x6f,
	0x20, 0xfb, 0x45, 0x5e, 0xe8, 0xf1, 0x28, 0xa8, 0xda, 0x2e, 0x4f, 0xef, 0xe9, 0x72, 0xb0, 0x64,
	0xc4, 0x2d, 0xce, 0x5e, 0x00, 0x56, 0xf9, 0x99, 0x0e, 0xd6, 0xac, 0x6c, 0xf3, 0x1e, 0x59, 0x9c,
	0x9f, 0x69, 0x61, 0xa1, 0xc6, 0x4b, 0xa8, 0xdf, 0xb5, 0xc2, 0x1e, 0x43, 0x75, 0x22, 0xfb, 0x63,
	0x15, 0x54, 0x9a, 0x6e, 0xcb, 0x17, 0xf3, 0x24, 0xc5, 0x04, 0xd1, 0x4a, 0xe3, 0x1b, 0x02, 0x6f,
	0x35, 0x93, 0x35, 0x80, 0x2c, 0x6c, 0x5f, 0x04, 0xb8, 0x89, 0x5a, 0xbe, 0x58, 0xe5, 0x2c, 0x80,
	0xda, 0x44, 0x0d, 0x4f, 0x8b, 0xdc, 0x58, 0xfb, 0xbe, 0x58, 0xa6, 0x5b, 0x7b, 0x50, 0x3b, 0x9e,
	0x87, 0xec, 0x11, 0x3c, 0x3c, 0x8e, 0xc5, 0x6e, 0x12, 0x75, 0x3f, 0x1e, 0x75, 0xde, 0x77, 0xb2,
	0x0f, 0x1d, 0xea, 0xb0, 0x0d, 0xf0, 0x97, 0xc5, 0x68, 0x3f, 0x6e, 0x77, 0x28, 0x62, 0x4f, 0x60,
	0x63, 0x59, 0x3a, 0x68, 0xef, 0x27, 0x51, 0x92, 0x1d, 0x1d, 0xd2, 0xca, 0xdc, 0x53, 0x8a, 0x49,
	0x85, 0xba, 0x29, 0x26, 0x2e, 0xc5, 0x8d, 0x00, 0xf0, 0xec, 0xdf, 0x18, 0x05, 0x77, 0x3c, 0xec,
	0xdb, 0xf5, 0x7b, 0x62, 0x16, 0xfe, 0x4b, 0xa5, 0x98, 0x60, 0x5a, 0x4d, 0x31, 0xa9, 0x51, 0xf2,
	0x7c, 0x07, 0xd6, 0x17, 0x5b, 0xe9, 0x5e, 0x9c, 0x2b, 0xc6, 0xa0, 0x1e, 0xb5, 0xbb, 0xf1, 0xbb,
	0x4c, 0x24, 0x27, 0xed, 0x6e, 0x92, 0xcd, 0x0c, 0xf9, 0xe0, 0xdd, 0x4e, 0xc5, 0x8c, 0x00, 0x8e,
	0xa3, 0xbd, 0x8c, 0x56, 0xdf, 0xbe, 0xb9, 0x9a, 0x72, 0xe7, 0xd7, 0x94, 0x3b, 0xd7, 0x53, 0x8e,
	0x6e, 0xa6, 0x1c, 0x7d, 0x29, 0x39, 0xfa, 0x5e, 0x72, 0xf4, 0xb3, 0xe4, 0xe8, 0xaa, 0xe4, 0xe8,
	0xba, 0xe4, 0xe8, 0x4f, 0xc9, 0x9d, 0x9b, 0x92, 0xa3, 0xaf, 0xbf, 0xb9, 0x73, 0xf2, 0x60, 0xfe,
	0x16, 0x85, 0x3d, 0xaa, 0x4f, 0x6b, 0xf6, 0x9a, 0x5e, 0xff, 0x0d, 0x00, 0x00, 0xff, 0xff, 0xd7,
	0xa4, 0xe4, 0xf1, 0x68, 0x02, 0x00, 0x00,
}
