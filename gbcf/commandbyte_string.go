// Code generated by "stringer -type=CommandByte"; DO NOT EDIT.

package gbcf

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[CONFIG-0]
	_ = x[NORMAL_DATA-1]
	_ = x[LAST_DATA-2]
	_ = x[ERASE-3]
	_ = x[STATUS-4]
}

const _CommandByte_name = "CONFIGNORMAL_DATALAST_DATAERASESTATUS"

var _CommandByte_index = [...]uint8{0, 6, 17, 26, 31, 37}

func (i CommandByte) String() string {
	if i >= CommandByte(len(_CommandByte_index)-1) {
		return "CommandByte(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _CommandByte_name[_CommandByte_index[i]:_CommandByte_index[i+1]]
}
