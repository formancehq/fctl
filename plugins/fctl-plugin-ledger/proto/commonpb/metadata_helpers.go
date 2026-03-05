package commonpb

import "strconv"

// MetadataValueToString converts any MetadataValue to its string representation.
func MetadataValueToString(v *MetadataValue) string {
	if v == nil {
		return ""
	}
	switch t := v.Type.(type) {
	case *MetadataValue_StringValue:
		return t.StringValue
	case *MetadataValue_IntValue:
		return strconv.FormatInt(t.IntValue, 10)
	case *MetadataValue_UintValue:
		return strconv.FormatUint(t.UintValue, 10)
	case *MetadataValue_BoolValue:
		return strconv.FormatBool(t.BoolValue)
	case *MetadataValue_NullValue:
		if t.NullValue != nil {
			return t.NullValue.Original
		}
		return ""
	default:
		return ""
	}
}
