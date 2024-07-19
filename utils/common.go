package utils

import "empyrean_lens/consts"

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

func GetApiAlias(hostName, apiName string) string {
	if apiName == "" {
		return apiName
	}
	for host, apis := range consts.NGINX_INGRESS_APIS {
		if hostName != host {
			continue
		}
		for _, api := range apis {
			if api.Api == apiName {
				return api.Alias
			}
		}
	}
	for host, apis := range consts.MODEL_NGINX_INGRESS_APIS {
		if hostName != host {
			continue
		}
		for _, api := range apis {
			if api.Api == apiName {
				return api.Alias
			}
		}
	}
	return apiName
}
