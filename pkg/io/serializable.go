package io

type Serializable interface {
	DecodeBinary(*BinReader)
	EncodeBinary(*BinWriter)
}

type decodable interface {
	DecodeBinary(*BinReader)
}

type encodable interface {
	EncodeBinary(*BinWriter)
}

func ToByteArray(s Serializable) ([]byte, error) {
	br := NewBufBinWriter()
	s.EncodeBinary(br.BinWriter)
	if br.Err != nil {
		return nil, br.Err
	}
	return br.Bytes(), nil
}

func FromByteArray(s Serializable, data []byte) error {
	br := NewBinReaderFromBuf(data)
	s.DecodeBinary(br)
	return br.Err
}
