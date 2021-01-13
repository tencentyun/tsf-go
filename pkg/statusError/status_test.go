package statusError

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEqual(t *testing.T) {
	assert.True(t, IsOK(nil))
	assert.True(t, IsOK(OK("")))
	assert.True(t, IsOK(OK("test")))

	assert.True(t, IsNotFound(NotFound("test")))
	assert.True(t, IsNotFound(NotFound("")))

	assert.False(t, IsNotFound(BadRequest("")))
	assert.False(t, IsNotFound(BadRequest("2233")))
}
