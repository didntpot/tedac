package legacypacket

import (
	"github.com/didntpot/tedac/tedac/legacyprotocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// ResourcePackChunkData is sent to the client so that the client can download the resource pack. Each packet
// holds a chunk of the compressed resource pack, of which the size is defined in the ResourcePackDataInfo
// packet sent before.
type ResourcePackChunkData struct {
	// UUID is the unique ID of the resource pack that the chunk of data is taken out of.
	UUID string
	// ChunkIndex is the current chunk index of the chunk. It is a number that starts at 0 and is incremented
	// for each resource pack data chunk sent to the client.
	ChunkIndex uint32
	// DataOffset is the current progress in bytes or offset in the data that the resource pack data chunk is
	// taken from.
	DataOffset uint64
	// RawPayload is a byte slice containing a chunk of data from the resource pack. It must be of the same size or
	// less than the DataChunkSize set in the ResourcePackDataInfo packet.
	Data []byte
}

// ID ...
func (*ResourcePackChunkData) ID() uint32 {
	return packet.IDResourcePackChunkData
}

// Marshal ...
func (pk *ResourcePackChunkData) Marshal(io protocol.IO) {
	io.String(&pk.UUID)
	io.Uint32(&pk.ChunkIndex)
	io.Uint64(&pk.DataOffset)

	legacyprotocol.ByteSlice(io, pk.Data)
}
