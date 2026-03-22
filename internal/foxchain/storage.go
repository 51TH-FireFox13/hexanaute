package foxchain

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
)

// Format de fichier FoxChain (.foxchain)
//
// Header (16 bytes):
//   Magic:   "FOXC" (4 bytes)
//   Version: uint16 (2 bytes)
//   Flags:   uint16 (2 bytes)
//   Count:   uint64 (8 bytes) — nombre de blocs
//
// Pour chaque bloc:
//   Index:     uint64 (8 bytes)
//   Timestamp: int64  (8 bytes)
//   PrevHash:  [32]byte
//   Type:      uint8  (1 byte)
//   DataLen:   uint32 (4 bytes)
//   Data:      []byte (DataLen bytes)
//   SigLen:    uint16 (2 bytes)
//   Signature: []byte (SigLen bytes)

var (
	magic      = []byte("FOXC")
	version    uint16 = 1
	ErrBadFile = errors.New("foxchain: fichier invalide ou corrompu")
)

// Save sauvegarde la chaîne dans un fichier.
func (c *Chain) Save(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("foxchain: impossible de créer %s: %w", path, err)
	}
	defer f.Close()

	// Header
	f.Write(magic)
	binary.Write(f, binary.BigEndian, version)
	binary.Write(f, binary.BigEndian, uint16(0)) // flags
	binary.Write(f, binary.BigEndian, uint64(len(c.Blocks)))

	// Blocs
	for _, b := range c.Blocks {
		binary.Write(f, binary.BigEndian, b.Index)
		binary.Write(f, binary.BigEndian, b.Timestamp)
		f.Write(b.PrevHash[:])
		f.Write([]byte{byte(b.Type)})
		binary.Write(f, binary.BigEndian, uint32(len(b.Data)))
		f.Write(b.Data)
		binary.Write(f, binary.BigEndian, uint16(len(b.Signature)))
		f.Write(b.Signature)
	}

	return nil
}

// LoadBlocks charge les blocs depuis un fichier.
func LoadBlocks(path string) ([]Block, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if len(data) < 16 {
		return nil, ErrBadFile
	}

	// Vérifier le magic
	if string(data[0:4]) != "FOXC" {
		return nil, ErrBadFile
	}

	fileVersion := binary.BigEndian.Uint16(data[4:6])
	if fileVersion != version {
		return nil, fmt.Errorf("foxchain: version %d non supportée (attendu %d)", fileVersion, version)
	}

	count := binary.BigEndian.Uint64(data[8:16])
	blocks := make([]Block, 0, count)
	pos := 16

	for i := uint64(0); i < count; i++ {
		if pos+49 > len(data) {
			return nil, ErrBadFile
		}

		var b Block
		b.Index = binary.BigEndian.Uint64(data[pos:])
		pos += 8
		b.Timestamp = int64(binary.BigEndian.Uint64(data[pos:]))
		pos += 8
		copy(b.PrevHash[:], data[pos:pos+32])
		pos += 32
		b.Type = BlockType(data[pos])
		pos++

		if pos+4 > len(data) {
			return nil, ErrBadFile
		}
		dataLen := int(binary.BigEndian.Uint32(data[pos:]))
		pos += 4

		if pos+dataLen > len(data) {
			return nil, ErrBadFile
		}
		b.Data = make([]byte, dataLen)
		copy(b.Data, data[pos:pos+dataLen])
		pos += dataLen

		if pos+2 > len(data) {
			return nil, ErrBadFile
		}
		sigLen := int(binary.BigEndian.Uint16(data[pos:]))
		pos += 2

		if pos+sigLen > len(data) {
			return nil, ErrBadFile
		}
		b.Signature = make([]byte, sigLen)
		copy(b.Signature, data[pos:pos+sigLen])
		pos += sigLen

		blocks = append(blocks, b)
	}

	return blocks, nil
}
