package cmdbuilder

func BoolToString(v *bool) string {
	if v == nil || !*v {
		return "No"
	}
	return "Yes"
}
