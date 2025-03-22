package display

import (
	"bytes"
)

var (
	CharMap map[string][]byte
)

func init() {
	CharMap = make(map[string][]byte)

	CharMap["「"] = []byte{0xa2}
	CharMap["」"] = []byte{0xa3}

	CharMap["ヲ"] = []byte{0xa6}

	CharMap["ァ"] = []byte{0xa7}
	CharMap["ィ"] = []byte{0xa8}
	CharMap["ゥ"] = []byte{0xa9}
	CharMap["ェ"] = []byte{0xaa}
	CharMap["ォ"] = []byte{0xab}

	CharMap["ャ"] = []byte{0xac}
	CharMap["ュ"] = []byte{0xad}
	CharMap["ョ"] = []byte{0xae}
	CharMap["ッ"] = []byte{0xaf}

	CharMap["ア"] = []byte{0xb1}
	CharMap["イ"] = []byte{0xb2}
	CharMap["ウ"] = []byte{0xb3}
	CharMap["エ"] = []byte{0xb4}
	CharMap["オ"] = []byte{0xb5}

	CharMap["カ"] = []byte{0xb6}
	CharMap["キ"] = []byte{0xb7}
	CharMap["ク"] = []byte{0xb8}
	CharMap["ケ"] = []byte{0xb9}
	CharMap["コ"] = []byte{0xba}

	CharMap["サ"] = []byte{0xbb}
	CharMap["シ"] = []byte{0xbc}
	CharMap["ス"] = []byte{0xbd}
	CharMap["セ"] = []byte{0xbe}
	CharMap["ソ"] = []byte{0xbf}

	CharMap["タ"] = []byte{0xc0}
	CharMap["チ"] = []byte{0xc1}
	CharMap["ツ"] = []byte{0xc2}
	CharMap["テ"] = []byte{0xc3}
	CharMap["ト"] = []byte{0xc4}

	CharMap["ナ"] = []byte{0xc5}
	CharMap["ニ"] = []byte{0xc6}
	CharMap["ヌ"] = []byte{0xc7}
	CharMap["ネ"] = []byte{0xc8}
	CharMap["ノ"] = []byte{0xc9}

	CharMap["ハ"] = []byte{0xca}
	CharMap["ヒ"] = []byte{0xcb}
	CharMap["フ"] = []byte{0xcc}
	CharMap["ヘ"] = []byte{0xcd}
	CharMap["ホ"] = []byte{0xce}

	CharMap["マ"] = []byte{0xcf}
	CharMap["ミ"] = []byte{0xd0}
	CharMap["ム"] = []byte{0xd1}
	CharMap["メ"] = []byte{0xd2}
	CharMap["モ"] = []byte{0xd3}

	CharMap["ヤ"] = []byte{0xd4}
	CharMap["ユ"] = []byte{0xd5}
	CharMap["ヨ"] = []byte{0xd6}

	CharMap["ラ"] = []byte{0xd7}
	CharMap["リ"] = []byte{0xd8}
	CharMap["ル"] = []byte{0xd9}
	CharMap["レ"] = []byte{0xda}
	CharMap["ロ"] = []byte{0xdb}

	CharMap["ワ"] = []byte{0xdc}
	CharMap["ン"] = []byte{0xdd}

	CharMap["”"] = []byte{0xde}
	CharMap["º"] = []byte{0xdf}
}

func ReplaceRuneWithLCDCharMap(original string) []byte {
	buff := []byte(original)

	for k, v := range CharMap {
		buff = bytes.ReplaceAll(buff, []byte(k), v)
	}

	return buff
}
