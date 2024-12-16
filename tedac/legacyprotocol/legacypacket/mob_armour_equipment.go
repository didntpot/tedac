package legacypacket

import (
	"github.com/didntpot/tedac/tedac/legacyprotocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// MobArmourEquipment is sent by the server to the client to update the armour an entity is wearing. It is
// sent for both players and other entities, such as zombies.
type MobArmourEquipment struct {
	// EntityRuntimeID is the runtime ID of the entity. The runtime ID is unique for each world session, and
	// entities are generally identified in packets using this runtime ID.
	EntityRuntimeID uint64
	// Helmet is the equipped helmet of the entity. Items that are not wearable on the head will not be
	// rendered by the client. Unlike in Java Edition, blocks cannot be worn.
	Helmet legacyprotocol.ItemStack
	// Chestplate is the chestplate of the entity. Items that are not wearable as chestplate will not be
	// rendered.
	Chestplate legacyprotocol.ItemStack
	// Leggings is the item worn as leggings by the entity. Items not wearable as leggings will not be
	// rendered client-side.
	Leggings legacyprotocol.ItemStack
	// Boots is the item worn as boots by the entity. Items not wearable as boots will not be rendered.
	Boots legacyprotocol.ItemStack
}

// ID ...
func (*MobArmourEquipment) ID() uint32 {
	return packet.IDMobArmourEquipment
}

// Marshal ...
func (pk *MobArmourEquipment) Marshal(io protocol.IO) {
	io.Varuint64(&pk.EntityRuntimeID)
	legacyprotocol.Item(io, &pk.Helmet)
	legacyprotocol.Item(io, &pk.Chestplate)
	legacyprotocol.Item(io, &pk.Leggings)
	legacyprotocol.Item(io, &pk.Boots)
}
