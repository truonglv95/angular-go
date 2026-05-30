package output

type MapEntry struct {
	Key    string
	Quoted bool
	Value  Expression
}

type MapLiteral []MapEntry

func NewMapEntry(key string, value Expression) MapEntry {
	return MapEntry{Key: key, Value: value, Quoted: false}
}

func NewMapLiteral(obj map[string]Expression, quoted bool) Expression {
	var entries []LiteralMapEntry
	for key, value := range obj {
		entries = append(entries, &LiteralMapPropertyAssignment{
			Key:    key,
			Quoted: quoted,
			Value:  value,
		})
	}
	return NewLiteralMapExpr(entries, nil, nil, nil)
}
