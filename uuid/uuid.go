package uuid

import uuid "github.com/satori/go.uuid"

var GenerateUUID = func() string {
	return uuid.NewV4().String()
}
