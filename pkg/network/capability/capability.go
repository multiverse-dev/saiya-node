package capability

import (
	"errors"

	"github.com/multiverse-dev/saiya/pkg/io"
)

// MaxCapabilities is the maximum number of capabilities per payload.
const MaxCapabilities = 32

// Capabilities is a list of Capability.
type Capabilities []Capability

// DecodeBinary implements Serializable interface.
func (cs *Capabilities) DecodeBinary(br *io.BinReader) {
	br.ReadArray(cs, MaxCapabilities)
	br.Err = cs.checkUniqueCapabilities()
}

// EncodeBinary implements Serializable interface.
func (cs *Capabilities) EncodeBinary(br *io.BinWriter) {
	br.WriteArray(*cs)
}

// checkUniqueCapabilities checks whether payload capabilities have unique type.
func (cs Capabilities) checkUniqueCapabilities() error {
	err := errors.New("capabilities with the same type are not allowed")
	var isFullNode, isTCP, isWS bool
	for _, cap := range cs {
		switch cap.Type {
		case FullNode:
			if isFullNode {
				return err
			}
			isFullNode = true
		case TCPServer:
			if isTCP {
				return err
			}
			isTCP = true
		case WSServer:
			if isWS {
				return err
			}
			isWS = true
		}
	}
	return nil
}

// Capability describes network service available for node.
type Capability struct {
	Type Type
	Data io.Serializable
}

// DecodeBinary implements Serializable interface.
func (c *Capability) DecodeBinary(br *io.BinReader) {
	c.Type = Type(br.ReadB())
	switch c.Type {
	case FullNode:
		c.Data = &Node{}
	case TCPServer, WSServer:
		c.Data = &Server{}
	default:
		br.Err = errors.New("unknown node capability type")
		return
	}
	c.Data.DecodeBinary(br)
}

// EncodeBinary implements Serializable interface.
func (c *Capability) EncodeBinary(bw *io.BinWriter) {
	if c.Data == nil {
		bw.Err = errors.New("capability has no data")
		return
	}
	bw.WriteB(byte(c.Type))
	c.Data.EncodeBinary(bw)
}

// Node represents full node capability with start height.
type Node struct {
	StartHeight uint32
}

// DecodeBinary implements Serializable interface.
func (n *Node) DecodeBinary(br *io.BinReader) {
	n.StartHeight = br.ReadU32LE()
}

// EncodeBinary implements Serializable interface.
func (n *Node) EncodeBinary(bw *io.BinWriter) {
	bw.WriteU32LE(n.StartHeight)
}

// Server represents TCP or WS server capability with port.
type Server struct {
	// Port is the port this server is listening on.
	Port uint16
}

// DecodeBinary implements Serializable interface.
func (s *Server) DecodeBinary(br *io.BinReader) {
	s.Port = br.ReadU16LE()
}

// EncodeBinary implements Serializable interface.
func (s *Server) EncodeBinary(bw *io.BinWriter) {
	bw.WriteU16LE(s.Port)
}
