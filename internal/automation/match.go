package automation

import "reflect"

func match(target, data any) bool {
	if tgt, ok := target.(map[string]any); ok {
		dat, ok := data.(map[string]any)
		if !ok {
			return false
		}

		for k, v := range tgt {
			if ret := match(v, dat[k]); !ret {
				return false
			}
		}
		return true
	} else if reflect.TypeOf(target) == reflect.TypeOf(data) {
		return reflect.DeepEqual(target, data)
	}

	return numericEqual(target, data)
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
