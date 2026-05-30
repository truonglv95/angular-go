package i18n

import (
	"fmt"
	"strings"

	xmlser "github.com/microsoft/typescript-go/angular-packages/compiler/i18n/serializers"
)

const xmbHandler = "angular"
const xmbMessagesTag = "messagebundle"
const xmbMessageTag = "msg"
const xmbPlaceholderTag = "ph"
const xmbExampleTag = "ex"
const xmbSourceTag = "source"

const xmbDocType = `<!ELEMENT messagebundle (msg)*>
<!ATTLIST messagebundle class CDATA #IMPLIED>

<!ELEMENT msg (#PCDATA|ph|source)*>
<!ATTLIST msg id CDATA #IMPLIED>
<!ATTLIST msg seq CDATA #IMPLIED>
<!ATTLIST msg name CDATA #IMPLIED>
<!ATTLIST msg desc CDATA #IMPLIED>
<!ATTLIST msg meaning CDATA #IMPLIED>
<!ATTLIST msg obsolete (obsolete) #IMPLIED>
<!ATTLIST msg xml:space (default|preserve) "default">
<!ATTLIST msg is_hidden CDATA #IMPLIED>

<!ELEMENT source (#PCDATA)>

<!ELEMENT ph (#PCDATA|ex)*>
<!ATTLIST ph name CDATA #REQUIRED>

<!ELEMENT ex (#PCDATA)>`

type Xmb struct{}

func (x *Xmb) Write(messages []Message, locale *string) string {
	_ = locale
	exampleVisitor := &xmbExampleVisitor{}
	visitor := &xmbWriteVisitor{}
	rootNode := xmlser.NewTag(xmbMessagesTag, map[string]string{"handler": xmbHandler}, nil)

	for _, message := range messages {
		attrs := map[string]string{"id": message.Id}
		if message.Description != "" {
			attrs["desc"] = message.Description
		}
		if message.Meaning != "" {
			attrs["meaning"] = message.Meaning
		}

		children := make([]xmlser.Node, 0, len(message.Sources)+len(message.Nodes))
		for _, source := range message.Sources {
			lineText := fmt.Sprintf("%s:%d", source.FilePath, source.StartLine)
			if source.EndLine != source.StartLine {
				lineText = fmt.Sprintf("%s,%d", lineText, source.EndLine)
			}
			children = append(children, xmlser.NewTag(xmbSourceTag, map[string]string{}, []xmlser.Node{xmlser.NewText(lineText)}))
		}
		children = append(children, visitor.serialize(message.Nodes)...)

		rootNode.Children = append(rootNode.Children,
			xmlser.NewCR(2),
			xmlser.NewTag(xmbMessageTag, attrs, children),
		)
	}
	rootNode.Children = append(rootNode.Children, xmlser.NewCR(0))

	return serializeI18nXML([]xmlser.Node{
		xmlser.NewDeclaration(map[string]string{"version": "1.0", "encoding": "UTF-8"}),
		xmlser.NewCR(0),
		xmlser.NewDoctype(xmbMessagesTag, xmbDocType),
		xmlser.NewCR(0),
		exampleVisitor.addDefaultExamples(rootNode),
		xmlser.NewCR(0),
	})
}

func (x *Xmb) Load(content string, url string) LoadResult {
	_ = content
	_ = url
	panic("Unsupported")
}

func (x *Xmb) Digest(message *Message) string {
	return DecimalDigest(message)
}

func (x *Xmb) CreateNameMapper(message *Message) PlaceholderMapper {
	return NewSimplePlaceholderMapper(message, xmbToPublicName)
}

func NewXmb() Serializer { return &Xmb{} }

type xmbWriteVisitor struct{}

func (v *xmbWriteVisitor) VisitText(text *Text, context any) any {
	return []xmlser.Node{xmlser.NewText(text.Value)}
}

func (v *xmbWriteVisitor) VisitContainer(container *Container, context any) any {
	var nodes []xmlser.Node
	for _, node := range container.Children {
		nodes = append(nodes, node.Visit(v, nil).([]xmlser.Node)...)
	}
	return nodes
}

func (v *xmbWriteVisitor) VisitIcu(icu *Icu, context any) any {
	nodes := []xmlser.Node{xmlser.NewText(fmt.Sprintf("{%s, %s, ", icu.ExpressionPlaceholder, icu.Type))}
	for _, c := range icu.CaseOrders {
		nodes = append(nodes, xmlser.NewText(fmt.Sprintf("%s {", c)))
		nodes = append(nodes, icu.Cases[c].Visit(v, nil).([]xmlser.Node)...)
		nodes = append(nodes, xmlser.NewText("} "))
	}
	nodes = append(nodes, xmlser.NewText("}"))
	return nodes
}

func (v *xmbWriteVisitor) VisitTagPlaceholder(ph *TagPlaceholder, context any) any {
	startTagAsText := xmlser.NewText(fmt.Sprintf("<%s>", ph.Tag))
	startEx := xmlser.NewTag(xmbExampleTag, map[string]string{}, []xmlser.Node{startTagAsText})
	startTagPh := xmlser.NewTag(xmbPlaceholderTag, map[string]string{"name": xmbToPublicName(ph.StartName)}, []xmlser.Node{startEx, startTagAsText})
	if ph.IsVoid {
		return []xmlser.Node{startTagPh}
	}

	closeTagAsText := xmlser.NewText(fmt.Sprintf("</%s>", ph.Tag))
	closeEx := xmlser.NewTag(xmbExampleTag, map[string]string{}, []xmlser.Node{closeTagAsText})
	closeTagPh := xmlser.NewTag(xmbPlaceholderTag, map[string]string{"name": xmbToPublicName(ph.CloseName)}, []xmlser.Node{closeEx, closeTagAsText})

	nodes := []xmlser.Node{startTagPh}
	nodes = append(nodes, v.serialize(ph.Children)...)
	nodes = append(nodes, closeTagPh)
	return nodes
}

func (v *xmbWriteVisitor) VisitPlaceholder(ph *Placeholder, context any) any {
	interpolationAsText := xmlser.NewText("{{" + ph.Value + "}}")
	exTag := xmlser.NewTag(xmbExampleTag, map[string]string{}, []xmlser.Node{interpolationAsText})
	return []xmlser.Node{xmlser.NewTag(xmbPlaceholderTag, map[string]string{"name": xmbToPublicName(ph.Name)}, []xmlser.Node{exTag, interpolationAsText})}
}

func (v *xmbWriteVisitor) VisitBlockPlaceholder(ph *BlockPlaceholder, context any) any {
	startAsText := xmlser.NewText("@" + ph.Name)
	startEx := xmlser.NewTag(xmbExampleTag, map[string]string{}, []xmlser.Node{startAsText})
	startTagPh := xmlser.NewTag(xmbPlaceholderTag, map[string]string{"name": xmbToPublicName(ph.StartName)}, []xmlser.Node{startEx, startAsText})

	closeAsText := xmlser.NewText("}")
	closeEx := xmlser.NewTag(xmbExampleTag, map[string]string{}, []xmlser.Node{closeAsText})
	closeTagPh := xmlser.NewTag(xmbPlaceholderTag, map[string]string{"name": xmbToPublicName(ph.CloseName)}, []xmlser.Node{closeEx, closeAsText})

	nodes := []xmlser.Node{startTagPh}
	nodes = append(nodes, v.serialize(ph.Children)...)
	nodes = append(nodes, closeTagPh)
	return nodes
}

func (v *xmbWriteVisitor) VisitIcuPlaceholder(ph *IcuPlaceholder, context any) any {
	var parts []string
	for _, value := range ph.Value.CaseOrders {
		parts = append(parts, value+" {...}")
	}
	icuAsText := xmlser.NewText(fmt.Sprintf("{%s, %s, %s}", ph.Value.Expression, ph.Value.Type, strings.Join(parts, " ")))
	exTag := xmlser.NewTag(xmbExampleTag, map[string]string{}, []xmlser.Node{icuAsText})
	return []xmlser.Node{xmlser.NewTag(xmbPlaceholderTag, map[string]string{"name": xmbToPublicName(ph.Name)}, []xmlser.Node{exTag, icuAsText})}
}

func (v *xmbWriteVisitor) serialize(nodes []Node) []xmlser.Node {
	var out []xmlser.Node
	for _, node := range nodes {
		out = append(out, node.Visit(v, nil).([]xmlser.Node)...)
	}
	return out
}

type xmbExampleVisitor struct{}

func (v *xmbExampleVisitor) addDefaultExamples(node xmlser.Node) xmlser.Node {
	node.Visit(v)
	return node
}

func (v *xmbExampleVisitor) VisitTag(tag *xmlser.Tag) any {
	if tag.Name == xmbPlaceholderTag {
		if len(tag.Children) == 0 {
			exText := xmlser.NewText(tag.Attrs["name"])
			tag.Children = []xmlser.Node{xmlser.NewTag(xmbExampleTag, map[string]string{}, []xmlser.Node{exText})}
		}
		return nil
	}
	for _, node := range tag.Children {
		node.Visit(v)
	}
	return nil
}

func (v *xmbExampleVisitor) VisitText(text *xmlser.Text) any               { return nil }
func (v *xmbExampleVisitor) VisitDeclaration(decl *xmlser.Declaration) any { return nil }
func (v *xmbExampleVisitor) VisitDoctype(doctype *xmlser.Doctype) any      { return nil }
