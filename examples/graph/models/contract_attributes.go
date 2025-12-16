package models

import (
	"encoding/json"
	"fmt"
)

type StringMap map[string]string

func (s *StringMap) Scan(src any) error {
	switch v := src.(type) {
	case []byte:
		if len(v) == 0 {
			return nil
		}
		return json.Unmarshal(v, s)
	case string:
		if v == "" {
			return nil
		}
		return json.Unmarshal([]byte(v), s)
	case nil:
		return nil
	default:
		return fmt.Errorf("cannot scan %T into StringMap", src)
	}
}
