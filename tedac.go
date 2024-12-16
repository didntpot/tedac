package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/df-mc/atomic"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/didntpot/tedac/tedac"
	"github.com/didntpot/tedac/tedac/chunk"
	"github.com/didntpot/tedac/tedac/latestmappings"
	"github.com/didntpot/tedac/tedac/legacyprotocol/legacypacket"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/sandertv/gophertunnel/minecraft/resource"
	"golang.org/x/oauth2"
	"log/slog"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

// Tedac ...
type Tedac struct {
	listener *minecraft.Listener

	localAddress  string
	remoteAddress string

	src oauth2.TokenSource
	ctx context.Context

	log *slog.Logger

	c chan interface{}
}

// NewTedac ...
func NewTedac(localAddress string) *Tedac {
	return &Tedac{localAddress: localAddress, src: tokenSource(), log: slog.Default(), c: make(chan interface{})}
}

// ProxyInfo ...
type ProxyInfo struct {
	LocalAddress  string
	RemoteAddress string
}

// ProxyingInfo ...
func (t *Tedac) ProxyingInfo() (ProxyInfo, error) {
	if t.listener == nil {
		return ProxyInfo{}, errors.New("no connection active")
	}
	return ProxyInfo{
		LocalAddress:  t.localAddress,
		RemoteAddress: t.remoteAddress,
	}, nil
}

// LocalAddress ...
func (t *Tedac) LocalAddress() string {
	address, _, _ := net.SplitHostPort(t.localAddress)
	return address
}

// LocalPort ...
func (t *Tedac) LocalPort() uint16 {
	_, str, _ := net.SplitHostPort(t.localAddress)
	port, _ := strconv.Atoi(str)
	return uint16(port)
}

// Terminate ...
func (t *Tedac) Terminate() {
	if t.listener == nil {
		return
	}
	t.c <- struct{}{}
	_ = t.listener.Close()
}

// Connect ...
func (t *Tedac) Connect(remoteAddress string) error {
	p, err := minecraft.NewForeignStatusProvider(remoteAddress)
	if err != nil {
		return err
	}

	err = os.Mkdir("packcache", 0644)
	useCache := err == nil || os.IsExist(err)

	var cachedPackNames []string
	conn, err := minecraft.Dialer{
		TokenSource: t.src,
		DownloadResourcePack: func(id uuid.UUID, version string, _, _ int) bool {
			if useCache {
				name := fmt.Sprintf("%s_%s", id, version)
				_, err = os.Stat(fmt.Sprintf("packcache/%s.mcpack", name))
				if err == nil {
					cachedPackNames = append(cachedPackNames, name)
					return false
				}
			}
			return true
		},
	}.Dial("raknet", remoteAddress)
	if err != nil {
		return err
	}
	packs := conn.ResourcePacks()
	_ = conn.Close()

	var cachedPacks []*resource.Pack
	if useCache {
		for _, name := range cachedPackNames {
			pack, err := resource.ReadPath(fmt.Sprintf("packcache/%s.mcpack", name))
			if err != nil {
				continue
			}
			cachedPacks = append(cachedPacks, pack)
		}
		for _, pack := range packs {
			packData := make([]byte, pack.Len())
			_, err = pack.ReadAt(packData, 0)
			if err != nil {
				continue
			}
			name := fmt.Sprintf("%s_%s", pack.UUID(), pack.Version())
			_ = os.WriteFile(fmt.Sprintf("packcache/%s.mcpack", name), packData, 0644)
		}
	}

	t.remoteAddress = remoteAddress

	go t.startRPC()

	t.listener, err = minecraft.ListenConfig{
		AllowInvalidPackets: true,
		AllowUnknownPackets: true,

		StatusProvider: p,

		ResourcePacks:     append(packs, cachedPacks...),
		AcceptedProtocols: []minecraft.Protocol{tedac.Protocol{}},
	}.Listen("raknet", t.localAddress)
	if err != nil {
		return err
	}
	go func() {
		for {
			c, err := t.listener.Accept()
			if err != nil {
				break
			}
			go t.handleConn(c.(*minecraft.Conn))
		}
	}()
	return nil
}

var (
	// airRID is the runtime ID of the air block in the latest version of the game.
	airRID, _ = latestmappings.StateToRuntimeID("minecraft:air", nil)
)

// handleConn ...
func (t *Tedac) handleConn(conn *minecraft.Conn) {
	clientData := conn.ClientData()
	if _, ok := conn.Protocol().(tedac.Protocol); ok {
		clientData.GameVersion = protocol.CurrentVersion
		clientData.DeviceOS = protocol.DeviceLinux
		clientData.DeviceModel = "TEDAC CLIENT"

		data, _ := base64.StdEncoding.DecodeString(clientData.SkinData)
		switch len(data) {
		case 32 * 64 * 4:
			clientData.SkinImageHeight = 32
			clientData.SkinImageWidth = 64
		case 64 * 64 * 4:
			clientData.SkinImageHeight = 64
			clientData.SkinImageWidth = 64
		case 128 * 128 * 4:
			clientData.SkinImageHeight = 128
			clientData.SkinImageWidth = 128
		}
	}

	serverConn, err := minecraft.Dialer{
		TokenSource: t.src,
		ClientData:  clientData,
	}.Dial("raknet", t.remoteAddress)
	if err != nil {
		t.log.Error("error while dialing: " + err.Error())
		return
	}

	data := serverConn.GameData()

	var g sync.WaitGroup
	g.Add(2)
	go func() {
		if err := conn.StartGame(data); err != nil {
			panic(err)
		}
		g.Done()
	}()
	go func() {
		if err := serverConn.DoSpawn(); err != nil {
			panic(err)
		}
		g.Done()
	}()
	g.Wait()

	rid := data.EntityRuntimeID
	oldMovementSystem := data.PlayerMovementSettings.MovementType == protocol.PlayerMovementModeClient
	if _, ok := conn.Protocol().(tedac.Protocol); ok {
		oldMovementSystem = true
	}

	r := world.Overworld.Range()
	pos := atomic.NewValue(data.PlayerPosition)
	lastPos := atomic.NewValue(data.PlayerPosition)
	yaw, pitch := atomic.NewValue(data.Yaw), atomic.NewValue(data.Pitch)

	startedSneaking, stoppedSneaking := atomic.NewValue(false), atomic.NewValue(false)
	startedSprinting, stoppedSprinting := atomic.NewValue(false), atomic.NewValue(false)
	startedGliding, stoppedGliding := atomic.NewValue(false), atomic.NewValue(false)
	startedSwimming, stoppedSwimming := atomic.NewValue(false), atomic.NewValue(false)
	startedJumping := atomic.NewValue(false)

	biomeBufferCache := make(map[protocol.ChunkPos][]byte)

	if oldMovementSystem {
		go func() {
			t := time.NewTicker(time.Millisecond * 500 / 20)
			defer t.Stop()

			var tick uint64
			for range t.C {
				currentPos, originalPos := pos.Load(), lastPos.Load()
				lastPos.Store(currentPos)

				currentYaw, currentPitch := yaw.Load(), pitch.Load()

				inputs := protocol.NewBitset(packet.PlayerAuthInputBitsetSize)
				if startedSneaking.CompareAndSwap(true, false) {
					inputs.Set(packet.InputFlagStartSneaking)
				}
				if stoppedSneaking.CompareAndSwap(true, false) {
					inputs.Set(packet.InputFlagStopSneaking)
				}
				if startedSprinting.CompareAndSwap(true, false) {
					inputs.Set(packet.InputFlagStartSprinting)
				}
				if stoppedSprinting.CompareAndSwap(true, false) {
					inputs.Set(packet.InputFlagStopSprinting)
				}
				if startedGliding.CompareAndSwap(true, false) {
					inputs.Set(packet.InputFlagStartGliding)
				}
				if stoppedGliding.CompareAndSwap(true, false) {
					inputs.Set(packet.InputFlagStopGliding)
				}
				if startedSwimming.CompareAndSwap(true, false) {
					inputs.Set(packet.InputFlagStartSwimming)
				}
				if stoppedSwimming.CompareAndSwap(true, false) {
					inputs.Set(packet.InputFlagStopSwimming)
				}
				if startedJumping.CompareAndSwap(true, false) {
					inputs.Set(packet.InputFlagJumping)
				}

				err := serverConn.WritePacket(&packet.PlayerAuthInput{
					Delta:            currentPos.Sub(originalPos),
					HeadYaw:          currentYaw,
					InputData:        inputs,
					InputMode:        packet.InputModeMouse,
					InteractionModel: packet.InteractionModelCrosshair,
					Pitch:            currentPitch,
					PlayMode:         packet.PlayModeNormal,
					Position:         currentPos,
					Tick:             tick,
					Yaw:              currentYaw,
				})
				if err != nil {
					return
				}
				_ = conn.WritePacket(&packet.NetworkChunkPublisherUpdate{ // cry about it
					Position: protocol.BlockPos{int32(currentPos.X()), int32(currentPos.Y()), int32(currentPos.Z())},
					Radius:   uint32(data.ChunkRadius) << 4,
				})
				tick++
			}
		}()
	}
	go func() {
		defer t.listener.Disconnect(conn, "connection lost")
		defer serverConn.Close()
		for {
			pk, err := conn.ReadPacket()
			if err != nil {
				return
			}
			switch pk := pk.(type) {
			case *packet.MovePlayer:
				if !oldMovementSystem {
					break
				}
				pos.Store(pk.Position)
				yaw.Store(pk.Yaw)
				pitch.Store(pk.Pitch)
				continue
			case *packet.PlayerAction:
				if !oldMovementSystem {
					break
				}
				switch pk.ActionType {
				case legacypacket.PlayerActionJump:
					startedJumping.Store(true)
					continue
				case legacypacket.PlayerActionStartSprint:
					startedSprinting.Store(true)
					continue
				case legacypacket.PlayerActionStopSprint:
					stoppedSprinting.Store(true)
					continue
				case legacypacket.PlayerActionStartSneak:
					startedSneaking.Store(true)
					continue
				case legacypacket.PlayerActionStopSneak:
					stoppedSneaking.Store(true)
					continue
				case legacypacket.PlayerActionStartSwimming:
					startedSwimming.Store(true)
					continue
				case legacypacket.PlayerActionStopSwimming:
					stoppedSwimming.Store(true)
					continue
				case legacypacket.PlayerActionStartGlide:
					startedGliding.Store(true)
					continue
				case legacypacket.PlayerActionStopGlide:
					stoppedGliding.Store(true)
					continue
				}
			}
			if err := serverConn.WritePacket(pk); err != nil {
				var disconnect minecraft.DisconnectError
				if errors.As(errors.Unwrap(err), &disconnect) {
					_ = t.listener.Disconnect(conn, disconnect.Error())
				}
				return
			}
		}
	}()
	go func() {
		defer serverConn.Close()
		defer t.listener.Disconnect(conn, "connection lost")
		for {
			pk, err := serverConn.ReadPacket()
			if err != nil {
				var disconnect minecraft.DisconnectError
				if errors.As(errors.Unwrap(err), &disconnect) {
					_ = t.listener.Disconnect(conn, disconnect.Error())
				}
				return
			}
			switch pk := pk.(type) {
			case *packet.MovePlayer:
				if !oldMovementSystem {
					break
				}
				if pk.EntityRuntimeID == rid {
					pos.Store(pk.Position)
					yaw.Store(pk.Yaw)
					pitch.Store(pk.Pitch)
				}
			case *packet.MoveActorAbsolute:
				if !oldMovementSystem {
					break
				}
				if pk.EntityRuntimeID == rid {
					pos.Store(pk.Position)
					yaw.Store(pk.Rotation[2])
					pitch.Store(pk.Rotation[0])
				}
			case *packet.MoveActorDelta:
				if !oldMovementSystem {
					break
				}
				if pk.EntityRuntimeID == rid {
					pos.Store(pk.Position)
					yaw.Store(pk.Rotation[2])
					pitch.Store(pk.Rotation[0])
				}
			case *packet.SubChunk:
				if _, ok := conn.Protocol().(tedac.Protocol); !ok {
					// Only Tedac clients should receive the old format.
					break
				}

				chunkBuf := bytes.NewBuffer(nil)
				blockEntities := make([]map[string]any, 0)
				for _, entry := range pk.SubChunkEntries {
					if entry.Result != protocol.SubChunkResultSuccess {
						chunkBuf.Write([]byte{
							chunk.SubChunkVersion,
							0, // The client will treat this as all air.
							uint8(entry.Offset[1]),
						})
						continue
					}

					var ind uint8
					readBuf := bytes.NewBuffer(entry.RawPayload)
					sub, err := chunk.DecodeSubChunk(airRID, r, readBuf, &ind, chunk.NetworkEncoding)
					if err != nil {
						fmt.Println(err)
						continue
					}

					var blockEntity map[string]any
					dec := nbt.NewDecoderWithEncoding(readBuf, nbt.NetworkLittleEndian)
					for {
						if err := dec.Decode(&blockEntity); err != nil {
							break
						}
						blockEntities = append(blockEntities, blockEntity)
					}

					chunkBuf.Write(chunk.EncodeSubChunk(sub, chunk.NetworkEncoding, r, int(ind)))
				}

				chunkPos := protocol.ChunkPos{pk.Position.X(), pk.Position.Z()}
				_, _ = chunkBuf.Write(append(biomeBufferCache[chunkPos], 0))
				delete(biomeBufferCache, chunkPos)

				enc := nbt.NewEncoderWithEncoding(chunkBuf, nbt.NetworkLittleEndian)
				for _, b := range blockEntities {
					_ = enc.Encode(b)
				}

				_ = conn.WritePacket(&packet.LevelChunk{
					Position:      chunkPos,
					SubChunkCount: uint32(len(pk.SubChunkEntries)),
					RawPayload:    append([]byte(nil), chunkBuf.Bytes()...),
				})
				continue
			case *packet.LevelChunk:
				if pk.SubChunkCount != protocol.SubChunkRequestModeLimitless && pk.SubChunkCount != protocol.SubChunkRequestModeLimited {
					// No changes to be made here.
					break
				}

				if _, ok := conn.Protocol().(tedac.Protocol); !ok {
					// Only Tedac clients should receive the old format.
					break
				}

				max := r.Height() >> 4
				if pk.SubChunkCount == protocol.SubChunkRequestModeLimited {
					max = int(pk.HighestSubChunk)
				}

				offsets := make([]protocol.SubChunkOffset, 0, max)
				for i := 0; i < max; i++ {
					offsets = append(offsets, protocol.SubChunkOffset{0, int8(i + (r[0] >> 4)), 0})
				}

				biomeBufferCache[pk.Position] = pk.RawPayload[:len(pk.RawPayload)-1]
				_ = serverConn.WritePacket(&packet.SubChunkRequest{
					Position: protocol.SubChunkPos{pk.Position.X(), 0, pk.Position.Z()},
					Offsets:  offsets,
				})
				continue
			case *packet.Transfer:
				t.remoteAddress = fmt.Sprintf("%s:%d", pk.Address, pk.Port)

				pk.Address = t.LocalAddress()
				pk.Port = t.LocalPort()
			}
			if err := conn.WritePacket(pk); err != nil {
				return
			}
		}
	}()
}
