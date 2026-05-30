package i18n

import (
	"fmt"
	"sort"
)

type Serializer interface {
	Write(messages []Message, locale *string) string
	Load(content string, url string) LoadResult
	Digest(message *Message) string
	CreateNameMapper(message *Message) PlaceholderMapper
}

type TranslationStore interface {
	Get(id string) ([]Node, bool)
	Has(id string) bool
	Keys() []string
}

type LoadResult struct {
	Locale           *string
	I18nNodesByMsgId TranslationStore
}

type eagerTranslationStore struct {
	data map[string][]Node
}

func NewEagerTranslationStore(data map[string][]Node) TranslationStore {
	if data == nil {
		data = map[string][]Node{}
	}
	return &eagerTranslationStore{data: data}
}

func (s *eagerTranslationStore) Get(id string) ([]Node, bool) {
	nodes, ok := s.data[id]
	return nodes, ok
}

func (s *eagerTranslationStore) Has(id string) bool {
	_, ok := s.data[id]
	return ok
}

func (s *eagerTranslationStore) Keys() []string {
	keys := make([]string, 0, len(s.data))
	for id := range s.data {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	return keys
}

type lazyTranslationStore struct {
	ids     []string
	loaders map[string]func() ([]Node, error)
	cache   map[string][]Node
}

func NewLazyTranslationStore(ids []string, loaders map[string]func() ([]Node, error)) TranslationStore {
	return &lazyTranslationStore{
		ids:     ids,
		loaders: loaders,
		cache:   map[string][]Node{},
	}
}

func (s *lazyTranslationStore) Get(id string) ([]Node, bool) {
	for _, knownID := range s.ids {
		if knownID != id {
			continue
		}
		if nodes, ok := s.cache[id]; ok {
			return nodes, true
		}
		loader, ok := s.loaders[id]
		if !ok {
			return nil, false
		}
		nodes, err := loader()
		if err != nil {
			panic(err.Error())
		}
		s.cache[id] = nodes
		return nodes, true
	}
	return nil, false
}

func (s *lazyTranslationStore) Has(id string) bool {
	for _, knownID := range s.ids {
		if knownID == id {
			return true
		}
	}
	return false
}

func (s *lazyTranslationStore) Keys() []string {
	keys := make([]string, len(s.ids))
	copy(keys, s.ids)
	return keys
}

type PlaceholderMapper interface {
	ToPublicName(internalName string) *string
	ToInternalName(publicName string) *string
}

type SimplePlaceholderMapper struct {
	mapName          func(name string) string
	internalToPublic map[string]string
	publicToNextId   map[string]int
	publicToInternal map[string]string
}

func NewSimplePlaceholderMapper(message *Message, mapName func(name string) string) *SimplePlaceholderMapper {
	m := &SimplePlaceholderMapper{
		mapName:          mapName,
		internalToPublic: make(map[string]string),
		publicToNextId:   make(map[string]int),
		publicToInternal: make(map[string]string),
	}
	for _, node := range message.Nodes {
		node.Visit(m, nil)
	}
	return m
}

func (m *SimplePlaceholderMapper) ToPublicName(internalName string) *string {
	if val, ok := m.internalToPublic[internalName]; ok {
		return &val
	}
	return nil
}

func (m *SimplePlaceholderMapper) ToInternalName(publicName string) *string {
	if val, ok := m.publicToInternal[publicName]; ok {
		return &val
	}
	return nil
}

func (m *SimplePlaceholderMapper) VisitText(text *Text, context any) any {
	return nil
}

func (m *SimplePlaceholderMapper) VisitContainer(container *Container, context any) any {
	for _, child := range container.Children {
		child.Visit(m, context)
	}
	return nil
}

func (m *SimplePlaceholderMapper) VisitIcu(icu *Icu, context any) any {
	var keys []string
	for k := range icu.Cases {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		icu.Cases[k].Visit(m, context)
	}
	return nil
}

func (m *SimplePlaceholderMapper) VisitTagPlaceholder(ph *TagPlaceholder, context any) any {
	m.visitPlaceholderName(ph.StartName)
	for _, child := range ph.Children {
		child.Visit(m, context)
	}
	m.visitPlaceholderName(ph.CloseName)
	return nil
}

func (m *SimplePlaceholderMapper) VisitPlaceholder(ph *Placeholder, context any) any {
	m.visitPlaceholderName(ph.Name)
	return nil
}

func (m *SimplePlaceholderMapper) VisitBlockPlaceholder(ph *BlockPlaceholder, context any) any {
	m.visitPlaceholderName(ph.StartName)
	for _, child := range ph.Children {
		child.Visit(m, context)
	}
	m.visitPlaceholderName(ph.CloseName)
	return nil
}

func (m *SimplePlaceholderMapper) VisitIcuPlaceholder(ph *IcuPlaceholder, context any) any {
	m.visitPlaceholderName(ph.Name)
	return nil
}

func (m *SimplePlaceholderMapper) visitPlaceholderName(internalName string) {
	if internalName == "" {
		return
	}
	if _, ok := m.internalToPublic[internalName]; ok {
		return
	}

	publicName := m.mapName(internalName)

	if _, ok := m.publicToInternal[publicName]; ok {
		nextId := m.publicToNextId[publicName]
		m.publicToNextId[publicName] = nextId + 1
		publicName = fmt.Sprintf("%s_%d", publicName, nextId)
	} else {
		m.publicToNextId[publicName] = 1
	}

	m.internalToPublic[internalName] = publicName
	m.publicToInternal[publicName] = internalName
}
