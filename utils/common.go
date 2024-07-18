package utils

func KeysOfMap[T comparable, V any](dict map[T]V) []T {
	keys := make([]T, 0, len(dict))
	for k := range dict {
		keys = append(keys, k)
	}
	return keys
}

func ValuesOfMap[T comparable, V any](dict map[T]V) []V {
	values := make([]V, 0, len(dict))
	for _, v := range dict {
		values = append(values, v)
	}
	return values
}
