# Mettaur

[![Go Report Card](https://goreportcard.com/badge/github.com/pokemium/mettaur)](https://goreportcard.com/report/github.com/pokemium/mettaur)
[![GitHub stars](https://img.shields.io/github/stars/pokemium/mettaur)](https://github.com/pokemium/mettaur/stargazers)
[![GitHub license](https://img.shields.io/github/license/pokemium/mettaur)](https://github.com/pokemium/mettaur/blob/main/LICENSE)

Mettaur is GBA emulator written in golang.

**Warning: This emulator is WIP, so many ROMs don't work correctly now.**

<img src="img/exe6.png" width="320" alt="exe6g" />&nbsp;<img src="img/pokered.png" width="320" alt="pokered" />

<img src="img/exe4b.png" width="320" alt="exe4b" />&nbsp;<img src="img/dqmc.png" width="320" alt="dqmc" />

## Run

Please download latest binary from [Release](https://github.com/pokemium/mettaur/releases).

```sh
$ mettaur XXXX.gba
```

## Build

```sh
# go1.16.x
$ make build
$ ./build/darwin-amd64/mettaur XXXX.gba
```

## Key

| keyboard             | game pad      |
| -------------------- | ------------- |
| <kbd>&larr;</kbd>    | &larr; button |
| <kbd>&uarr;</kbd>    | &uarr; button |
| <kbd>&darr;</kbd>    | &darr; button |
| <kbd>&rarr;</kbd>    | &rarr; button |
| <kbd>X</kbd>         | A button      |
| <kbd>Z</kbd>         | B button      |
| <kbd>S</kbd>         | R button      |
| <kbd>A</kbd>         | L button      |
| <kbd>Enter</kbd>     | Start button  |
| <kbd>Backspace</kbd> | Select button |

## ToDo

- [ ] Window
- [ ] Mosaic
- [ ] Blend
- [ ] GUI
- [ ] Serial communication
- [ ] BG mode5
- [ ] GameBoy Compatibility
- [ ] Debug feature
- [ ] Fix some bugs

## Game Compatibility List

| Game Title             | Compatibility      |
| -------------------- | ------------- |
| バトルネットワーク ロックマンエグゼ3 BLACK | ✅ |
| ロックマンエグゼ4 トーナメント ブルームーン | ✅ |
| ロックマンエグゼ6 電脳獣グレイガ・電脳獣ファルザー | ✅ |
| ポケットモンスター ファイアレッド | ✅ |
| ドラゴンクエストモンスターズ キャラバンハート | 🔺 |
| MOTHER1+2 | 🔺 |


## References

- [GBATEK](https://problemkaputt.de/gbatek.htm)
- [gba_doc_ja](https://github.com/pokemium/gba_doc_ja)
- [gdkchan/gdkGBA](https://github.com/gdkchan/gdkGBA)
