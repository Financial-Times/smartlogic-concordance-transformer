package smartlogic

import (
	"crypto/md5"

	uuid "github.com/pborman/uuid"
)

func convertFactsetIDToUUID(factsetID string) string {
	h := md5.New()
	h.Write([]byte(factsetID))
	md5One := h.Sum(nil)
	return uuid.NewMD5(uuid.UUID{}, md5One).String()
}
