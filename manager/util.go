package manager

import "fmt"

func VerifyFields(req map[string]string, fields ...string) (err error) {
	for _, field := range fields {
		if _, ok := req[field]; !ok {
			err = fmt.Errorf("no field %s in req payload", field)
			return
		}
	}
	return
}
