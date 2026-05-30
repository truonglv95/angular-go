package i18n

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/ml_parser"
	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
)

type MessageBundle struct {
	htmlParser         *ml_parser.HtmlParser
	implicitTags       []string
	implicitAttrs      map[string][]string
	locale             *string
	messages           []*Message
	preserveWhitespace bool
}

func NewMessageBundle(
	htmlParser *ml_parser.HtmlParser,
	implicitTags []string,
	implicitAttrs map[string][]string,
	locale *string,
) *MessageBundle {
	return &MessageBundle{
		htmlParser:         htmlParser,
		implicitTags:       implicitTags,
		implicitAttrs:      implicitAttrs,
		locale:             locale,
		messages:           []*Message{},
		preserveWhitespace: true,
	}
}

func (b *MessageBundle) UpdateFromTemplate(source string, url string, interpolationConfig any) []*parse_util.ParseError {
	htmlParserResult := b.htmlParser.Parse(source, url, &ml_parser.TokenizeOptions{TokenizeExpansionForms: true})

	if len(htmlParserResult.Errors) > 0 {
		var parseErrors []*parse_util.ParseError
		for _, err := range htmlParserResult.Errors {
			if pe, ok := err.(*parse_util.ParseError); ok {
				parseErrors = append(parseErrors, pe)
			}
		}
		return parseErrors
	}

	var rootNodes []ml_parser.Node
	if b.preserveWhitespace {
		rootNodes = htmlParserResult.RootNodes
	} else {
		rootNodes = ml_parser.VisitAllWithSiblingsForNodes(
			ml_parser.NewWhitespaceVisitor(false),
			htmlParserResult.RootNodes,
		)
	}

	i18nParserResult := ExtractMessages(
		rootNodes,
		b.implicitTags,
		b.implicitAttrs,
		b.preserveWhitespace,
	)

	if len(i18nParserResult.Errors) > 0 {
		return i18nParserResult.Errors
	}

	b.messages = append(b.messages, i18nParserResult.Messages...)
	return nil
}

func (b *MessageBundle) GetMessages() []*Message {
	return b.messages
}

func (b *MessageBundle) Write(serializer Serializer, filterSources func(path string) string) string {
	messages := make(map[string]*Message)
	var messageOrders []string

	// Deduplicate messages based on their ID
	for _, message := range b.messages {
		id := serializer.Digest(message)
		if _, ok := messages[id]; !ok {
			messages[id] = message
			messageOrders = append(messageOrders, id)
		} else {
			messages[id].Sources = append(messages[id].Sources, message.Sources...)
		}
	}

	// Transform placeholder names using the serializer mapping
	var msgList []Message
	mapperVisitor := &MapPlaceholderNames{}

	for _, id := range messageOrders {
		src := messages[id]
		mapper := serializer.CreateNameMapper(src)
		var nodes []Node
		if mapper != nil {
			nodes = mapperVisitor.Convert(src.Nodes, mapper)
		} else {
			nodes = src.Nodes
		}
		transformedMessage := NewMessage(nodes, make(map[string]*MessagePlaceholder), make(map[string]*Message), src.Meaning, src.Description, id)
		transformedMessage.Sources = src.Sources
		if filterSources != nil {
			for _, source := range transformedMessage.Sources {
				source.FilePath = filterSources(source.FilePath)
			}
		}
		msgList = append(msgList, *transformedMessage)
	}

	return serializer.Write(msgList, b.locale)
}

type MapPlaceholderNames struct {
	CloneVisitor
}

func (v *MapPlaceholderNames) Convert(nodes []Node, mapper PlaceholderMapper) []Node {
	if mapper == nil {
		return nodes
	}
	result := make([]Node, len(nodes))
	for i, n := range nodes {
		result[i] = n.Visit(v, mapper).(Node)
	}
	return result
}

func (v *MapPlaceholderNames) VisitTagPlaceholder(ph *TagPlaceholder, context any) any {
	mapper := context.(PlaceholderMapper)
	startName := ""
	if startNamePtr := mapper.ToPublicName(ph.StartName); startNamePtr != nil {
		startName = *startNamePtr
	}
	closeName := ""
	if ph.CloseName != "" {
		if closeNamePtr := mapper.ToPublicName(ph.CloseName); closeNamePtr != nil {
			closeName = *closeNamePtr
		}
	}
	children := make([]Node, len(ph.Children))
	for i, n := range ph.Children {
		children[i] = n.Visit(v, mapper).(Node)
	}
	return NewTagPlaceholder(
		ph.Tag,
		ph.Attrs,
		startName,
		closeName,
		children,
		ph.IsVoid,
		ph.SourceSpan,
		ph.StartSourceSpan,
		ph.EndSourceSpan,
	)
}

func (v *MapPlaceholderNames) VisitBlockPlaceholder(ph *BlockPlaceholder, context any) any {
	mapper := context.(PlaceholderMapper)
	startName := ""
	if startNamePtr := mapper.ToPublicName(ph.StartName); startNamePtr != nil {
		startName = *startNamePtr
	}
	closeName := ""
	if ph.CloseName != "" {
		if closeNamePtr := mapper.ToPublicName(ph.CloseName); closeNamePtr != nil {
			closeName = *closeNamePtr
		}
	}
	children := make([]Node, len(ph.Children))
	for i, n := range ph.Children {
		children[i] = n.Visit(v, mapper).(Node)
	}
	return NewBlockPlaceholder(
		ph.Name,
		ph.Parameters,
		startName,
		closeName,
		children,
		ph.SourceSpan,
		ph.StartSourceSpan,
		ph.EndSourceSpan,
	)
}

func (v *MapPlaceholderNames) VisitPlaceholder(ph *Placeholder, context any) any {
	mapper := context.(PlaceholderMapper)
	name := ""
	if namePtr := mapper.ToPublicName(ph.Name); namePtr != nil {
		name = *namePtr
	}
	return NewPlaceholder(ph.Value, name, ph.SourceSpan)
}

func (v *MapPlaceholderNames) VisitIcuPlaceholder(ph *IcuPlaceholder, context any) any {
	mapper := context.(PlaceholderMapper)
	name := ""
	if namePtr := mapper.ToPublicName(ph.Name); namePtr != nil {
		name = *namePtr
	}
	return NewIcuPlaceholder(ph.Value, name, ph.SourceSpan)
}
