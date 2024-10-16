package mqtt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSwitchCommandParse(t *testing.T) {

	assert := assert.New(t)

	baseTopic := "loremTopic"
	topic := "loremTopic/switch/my_device/command"
	r := switchCommandExtractor(baseTopic)
	matches := r.FindAllStringSubmatch(topic, 1)

	assert.Equal(matches[0][1], "my_device", "device extract")
}

func TestSwitchCommandParseFail(t *testing.T) {

	assert := assert.New(t)

	baseTopic := "loremTopic"
	topic := "loremTopic/switch/my_device/state"
	r := switchCommandExtractor(baseTopic)
	matches := r.FindAllStringSubmatch(topic, 1)

	assert.Equal(len(matches), 0, "no matches")
}

func TestInputNumberCommandParse(t *testing.T) {

	assert := assert.New(t)

	baseTopic := "loremTopic"
	topic := "loremTopic/number/number_name/set"
	r := inputNumberCommandExtractor(baseTopic)
	matches := r.FindAllStringSubmatch(topic, 1)

	assert.Equal(matches[0][1], "number_name", "number_id extract")
}

func TestInputNumberCommandParseFail(t *testing.T) {

	assert := assert.New(t)

	baseTopic := "loremTopic"
	topic := "loremTopic/switch/number_name/command"
	r := inputNumberCommandExtractor(baseTopic)
	matches := r.FindAllStringSubmatch(topic, 1)

	assert.Equal(len(matches), 0, "no matches")
}
