package stack

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSortVersionsDescending(t *testing.T) {
	versions := []string{
		"v2.9.0",
		"v3.0.0-rc.1",
		"v2.10.0",
		"v3.0.0",
		"v1.0.0+build.1",
		"v1.0.0+build.2",
		"preview",
		"development",
	}

	sortVersionsDescending(versions)

	assert.Equal(t, []string{
		"v3.0.0",
		"v3.0.0-rc.1",
		"v2.10.0",
		"v2.9.0",
		"v1.0.0+build.2",
		"v1.0.0+build.1",
		"preview",
		"development",
	}, versions)
}
