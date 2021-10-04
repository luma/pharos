package protocol

type Marshaler interface {
	Marshal() ([]byte, error)
}

type Unmarshaler interface {
	Unmarshal(data []byte) error
}

type Marshalable interface {
	Marshaler
	Unmarshaler
}

type Update struct {
	Key  string
	Body Marshalable
}
