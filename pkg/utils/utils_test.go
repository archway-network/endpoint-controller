package utils_test

import (
	"testing"

	"github.com/archway-network/endpoint-controller/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func TestRemoveFromSlice(t *testing.T) {
	testCases := map[string][]string{
		"3.3.3.3": {"1.1.1.1", "2.2.2.2", "4.4.4.4"},
		"1.1.1.1": {"2.2.2.2", "3.3.3.3", "4.4.4.4"},
		"4.4.4.4": {"1.1.1.1", "2.2.2.2", "3.3.3.3"},
	}

	for k, v := range testCases {
		data := []string{"1.1.1.1", "2.2.2.2", "3.3.3.3", "4.4.4.4"}
		assert.Equal(t, v, utils.RemoveFromSlice(data, k))
	}
}
