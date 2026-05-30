package i18n

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"math/bits"
	"strings"
)

func Digest(message *Message) string {
	if message.Id != "" {
		return message.Id
	}
	return ComputeDigest(message)
}

func ComputeDigest(message *Message) string {
	nodesStr := strings.Join(SerializeNodes(message.Nodes), "")
	return Sha1(nodesStr + fmt.Sprintf("[%s]", message.Meaning))
}

func DecimalDigest(message *Message) string {
	if message.Id != "" {
		return message.Id
	}
	return ComputeDecimalDigest(message)
}

func ComputeDecimalDigest(message *Message) string {
	visitor := &_SerializerIgnoreIcuExpVisitor{_SerializerVisitor{}}
	var parts []string
	for _, node := range message.Nodes {
		parts = append(parts, node.Visit(visitor, nil).(string))
	}
	return ComputeMsgId(strings.Join(parts, ""), message.Meaning)
}

type _SerializerVisitor struct{}

func (v *_SerializerVisitor) VisitText(text *Text, context any) any {
	return text.Value
}

func (v *_SerializerVisitor) VisitContainer(container *Container, context any) any {
	var children []string
	for _, child := range container.Children {
		children = append(children, child.Visit(v, nil).(string))
	}
	return fmt.Sprintf("[%s]", strings.Join(children, ", "))
}

func (v *_SerializerVisitor) VisitIcu(icu *Icu, context any) any {
	var strCases []string
	for _, k := range icu.CaseOrders {
		strCases = append(strCases, fmt.Sprintf("%s {%s}", k, icu.Cases[k].Visit(v, nil).(string)))
	}
	return fmt.Sprintf("{%s, %s, %s}", icu.Expression, icu.Type, strings.Join(strCases, ", "))
}

func (v *_SerializerVisitor) VisitTagPlaceholder(ph *TagPlaceholder, context any) any {
	if ph.IsVoid {
		return fmt.Sprintf("<ph tag name=\"%s\"/>", ph.StartName)
	}
	var children []string
	for _, child := range ph.Children {
		children = append(children, child.Visit(v, nil).(string))
	}
	return fmt.Sprintf("<ph tag name=\"%s\">%s</ph name=\"%s\">", ph.StartName, strings.Join(children, ", "), ph.CloseName)
}

func (v *_SerializerVisitor) VisitPlaceholder(ph *Placeholder, context any) any {
	if ph.Value != "" {
		return fmt.Sprintf("<ph name=\"%s\">%s</ph>", ph.Name, ph.Value)
	}
	return fmt.Sprintf("<ph name=\"%s\"/>", ph.Name)
}

func (v *_SerializerVisitor) VisitIcuPlaceholder(ph *IcuPlaceholder, context any) any {
	return fmt.Sprintf("<ph icu name=\"%s\">%s</ph>", ph.Name, ph.Value.Visit(v, nil).(string))
}

func (v *_SerializerVisitor) VisitBlockPlaceholder(ph *BlockPlaceholder, context any) any {
	var children []string
	for _, child := range ph.Children {
		children = append(children, child.Visit(v, nil).(string))
	}
	return fmt.Sprintf("<ph block name=\"%s\">%s</ph name=\"%s\">", ph.StartName, strings.Join(children, ", "), ph.CloseName)
}

var serializerVisitor = &_SerializerVisitor{}

func SerializeNodes(nodes []Node) []string {
	var result []string
	for _, node := range nodes {
		result = append(result, node.Visit(serializerVisitor, nil).(string))
	}
	return result
}

type _SerializerIgnoreIcuExpVisitor struct {
	_SerializerVisitor
}

func (v *_SerializerIgnoreIcuExpVisitor) VisitContainer(container *Container, context any) any {
	var children []string
	for _, child := range container.Children {
		children = append(children, child.Visit(v, nil).(string))
	}
	return fmt.Sprintf("[%s]", strings.Join(children, ", "))
}

func (v *_SerializerIgnoreIcuExpVisitor) VisitIcu(icu *Icu, context any) any {
	var strCases []string
	for _, k := range icu.CaseOrders {
		strCases = append(strCases, fmt.Sprintf("%s {%s}", k, icu.Cases[k].Visit(v, nil).(string)))
	}
	return fmt.Sprintf("{%s, %s}", icu.Type, strings.Join(strCases, ", "))
}

func (v *_SerializerIgnoreIcuExpVisitor) VisitTagPlaceholder(ph *TagPlaceholder, context any) any {
	if ph.IsVoid {
		return fmt.Sprintf("<ph tag name=\"%s\"/>", ph.StartName)
	}
	var children []string
	for _, child := range ph.Children {
		children = append(children, child.Visit(v, nil).(string))
	}
	return fmt.Sprintf("<ph tag name=\"%s\">%s</ph name=\"%s\">", ph.StartName, strings.Join(children, ", "), ph.CloseName)
}

func (v *_SerializerIgnoreIcuExpVisitor) VisitIcuPlaceholder(ph *IcuPlaceholder, context any) any {
	return fmt.Sprintf("<ph icu name=\"%s\">%s</ph>", ph.Name, ph.Value.Visit(v, nil).(string))
}

func (v *_SerializerIgnoreIcuExpVisitor) VisitBlockPlaceholder(ph *BlockPlaceholder, context any) any {
	var children []string
	for _, child := range ph.Children {
		children = append(children, child.Visit(v, nil).(string))
	}
	return fmt.Sprintf("<ph block name=\"%s\">%s</ph name=\"%s\">", ph.StartName, strings.Join(children, ", "), ph.CloseName)
}

func Sha1(str string) string {
	h := sha1.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}

func fingerprint(str string) uint64 {
	utf8 := []byte(str)

	hi := hash32(utf8, uint32(len(utf8)), 0)
	lo := hash32(utf8, uint32(len(utf8)), 102072)

	if hi == 0 && (lo == 0 || lo == 1) {
		hi = hi ^ 0x130f9bef
		lo = lo ^ 0x94a0a928 // -0x6b5f56d8 is 0x94a0a928 in uint32
	}

	return (uint64(hi) << 32) | uint64(lo)
}

func ComputeMsgId(msg string, meaning string) string {
	msgFingerprint := fingerprint(msg)

	if meaning != "" {
		msgFingerprint = bits.RotateLeft64(msgFingerprint, 1)
		msgFingerprint += fingerprint(meaning)
	}

	// BigInt.asUintN(63, msgFingerprint) masks to 63 bits
	return fmt.Sprintf("%d", msgFingerprint&0x7FFFFFFFFFFFFFFF)
}

func hash32(view []byte, length uint32, c uint32) uint32 {
	a := uint32(0x9e3779b9)
	b := uint32(0x9e3779b9)
	index := uint32(0)

	var end uint32
	if length >= 12 {
		end = length - 12
	} else {
		end = 0xffffffff // avoid underflow issues if handled properly, but we use index + 12 <= length
	}

	for index <= end && length >= 12 && index+12 <= length {
		a += getUint32(view, index, true)
		b += getUint32(view, index+4, true)
		c += getUint32(view, index+8, true)
		a, b, c = mix(a, b, c)
		index += 12
	}

	remainder := length - index

	c += length

	if remainder >= 4 {
		a += getUint32(view, index, true)
		index += 4

		if remainder >= 8 {
			b += getUint32(view, index, true)
			index += 4

			if remainder >= 9 {
				c += uint32(view[index]) << 8
				index++
			}
			if remainder >= 10 {
				c += uint32(view[index]) << 16
				index++
			}
			if remainder == 11 {
				c += uint32(view[index]) << 24
				index++
			}
		} else {
			if remainder >= 5 {
				b += uint32(view[index])
				index++
			}
			if remainder >= 6 {
				b += uint32(view[index]) << 8
				index++
			}
			if remainder == 7 {
				b += uint32(view[index]) << 16
				index++
			}
		}
	} else {
		if remainder >= 1 {
			a += uint32(view[index])
			index++
		}
		if remainder >= 2 {
			a += uint32(view[index]) << 8
			index++
		}
		if remainder == 3 {
			a += uint32(view[index]) << 16
			index++
		}
	}

	_, _, c = mix(a, b, c)
	return c
}

func getUint32(b []byte, index uint32, littleEndian bool) uint32 {
	if littleEndian {
		return uint32(b[index]) | uint32(b[index+1])<<8 | uint32(b[index+2])<<16 | uint32(b[index+3])<<24
	}
	return uint32(b[index+3]) | uint32(b[index+2])<<8 | uint32(b[index+1])<<16 | uint32(b[index])<<24
}

func mix(a uint32, b uint32, c uint32) (uint32, uint32, uint32) {
	a -= b
	a -= c
	a ^= c >> 13
	b -= c
	b -= a
	b ^= a << 8
	c -= a
	c -= b
	c ^= b >> 13
	a -= b
	a -= c
	a ^= c >> 12
	b -= c
	b -= a
	b ^= a << 16
	c -= a
	c -= b
	c ^= b >> 5
	a -= b
	a -= c
	a ^= c >> 3
	b -= c
	b -= a
	b ^= a << 10
	c -= a
	c -= b
	c ^= b >> 15
	return a, b, c
}

func IsI18nRootNode(meta I18nMeta) bool {
	if meta == nil {
		return false
	}
	if _, ok := meta.(*Message); ok {
		return true
	}
	return false
}
