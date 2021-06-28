package audio

import (
	"github.com/hajimehoshi/oto"
	"github.com/pokemium/magia/pkg/gba/apu"
)

var context *oto.Context
var player *oto.Player
var Stream []byte

func Init() {
	Stream = make([]byte, apu.STREAM_LEN)

	var err error
	context, err = oto.NewContext(apu.SND_FREQUENCY, 2, 2, apu.STREAM_LEN)
	if err != nil {
		panic(err)
	}

	player = context.NewPlayer()
}

func Play() {
	if player == nil {
		return
	}
	player.Write(Stream)
}
