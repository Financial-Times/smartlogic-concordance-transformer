package smartlogic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertFactsetIDToUUID_Success(t *testing.T) {
	ids := []string{
		"0DFC00-E",
		"0C0NP2-E",
		"0D0WTT-E",
		"0CZQN1-E",
		"0CBDM7-E",
		"0CKQM0-E",
		"0CKQM0-E",
		"0D2QL9-E",
	}

	uuids := []string{
		"3c409b48-1a47-3ae1-a1f0-4c77d7d4df99",
		"bdd0438e-54d1-31bc-8748-a1a9a0bfb8ea",
		"2a9ecb75-82d0-3e2c-b128-7e20093049d5",
		"2da3ae10-110b-3d4a-bb8f-e0e3664f7ddd",
		"8b92eeb6-6d6e-3c17-8620-c581ccfe6b13",
		"e361ced1-e28d-3797-be40-1586d159b576",
		"e361ced1-e28d-3797-be40-1586d159b576",
		"33629787-d20d-396c-84d3-cb03362392c1",
	}

	for i, id := range ids {
		assert.Equal(t, convertFactsetIDToUUID(id), uuids[i])
	}
}
