// Code generated by "stringer -type=Origin"; DO NOT EDIT.

package types

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[Token-0]
	_ = x[Form-1]
	_ = x[DB-2]
}

const _Origin_name = "TokenFormDB"

var _Origin_index = [...]uint8{0, 5, 9, 11}

func (i Origin) String() string {
	if i < 0 || i >= Origin(len(_Origin_index)-1) {
		return "Origin(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _Origin_name[_Origin_index[i]:_Origin_index[i+1]]
}
