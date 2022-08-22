package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/webx-top/com"
)

func TestGoVersion(t *testing.T) {
	a := &App{}
	gover, err := a.goVersion()
	assert.NoError(t, err)
	assert.Equal(t, true, len(gover) > 0)
	fmt.Println(gover)

	assert.True(t, com.VersionComparex(`1.18`, `1.18.0`, ">="))
	assert.True(t, com.VersionComparex(`1.18.1`, `1.18.0`, ">="))
}
