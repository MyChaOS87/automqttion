package automation

import "reflect"

func match(target, data any) bool {
	switch tgt := target.(type) {
	case map[string]any:
		return matchMap(tgt, data)
	case []any:
		return matchSlice(tgt, data)
	}

	if reflect.TypeOf(target) == reflect.TypeOf(data) {
		return reflect.DeepEqual(target, data)
	}

	return numericEqual(target, data)
}

func matchMap(tgt map[string]any, data any) bool {
	dat, ok := data.(map[string]any)
	if !ok {
		return false
	}

	for k, v := range tgt {
		dv, exists := dat[k]
		if !exists || !match(v, dv) {
			return false
		}
	}

	return true
}

func matchSlice(tgt []any, data any) bool {
	dat, ok := data.([]any)
	if !ok || len(tgt) != len(dat) {
		return false
	}

	for i := range tgt {
		if !match(tgt[i], dat[i]) {
			return false
		}
	}

	return true
}

func isNumeric(v any) bool {
	switch v.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return true
	}

	return false
}

func numericEqual(a, b any) bool {
	if !isNumeric(a) || !isNumeric(b) {
		return false
	}

	av := reflect.ValueOf(a)
	bv := reflect.ValueOf(b)

	if !av.CanConvert(bv.Type()) || !bv.CanConvert(av.Type()) {
		return false
	}

	// check equality in both directions as double->int could loose significant digits
	return av.Convert(bv.Type()).Interface() == bv.Interface() &&
		bv.Convert(av.Type()).Interface() == av.Interface()
}
