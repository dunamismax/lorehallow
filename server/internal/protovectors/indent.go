package protovectors

import (
	"bytes"
	"encoding/json"
)

func indentJSON(dst *bytes.Buffer, src []byte) error {
	return json.Indent(dst, src, "", "  ")
}
