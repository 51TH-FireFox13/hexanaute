// Package foxchain implémente un journal chaîné chiffré (append-only log)
// pour stocker les données utilisateur (favoris, mots de passe, historique,
// état des onglets) de manière portable et sécurisée.
package foxchain

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"time"
)

// BlockType identifie le type de données dans un bloc.
type BlockType uint8

const (
	BlockBookmark  BlockType = 1
	BlockPassword  BlockType = 2
	BlockHistory   BlockType = 3
	BlockTabState  BlockType = 4
	BlockSetting   BlockType = 5
)

// Block représente un maillon du journal chaîné.
type Block struct {
	Index     uint64
	Timestamp int64
	PrevHash  [32]byte
	Type      BlockType
	Data      []byte // chiffré XChaCha20-Poly1305
	Signature []byte // Ed25519
}

// Hash calcule le hash SHA-256 du bloc (sans la signature).
func (b *Block) Hash() [32]byte {
	buf := make([]byte, 8+8+32+1+len(b.Data))
	binary.BigEndian.PutUint64(buf[0:8], b.Index)
	binary.BigEndian.PutUint64(buf[8:16], uint64(b.Timestamp))
	copy(buf[16:48], b.PrevHash[:])
	buf[48] = byte(b.Type)
	copy(buf[49:], b.Data)
	return sha256.Sum256(buf)
}

// Chain est le journal chaîné complet.
type Chain struct {
	Blocks []Block
	pubKey ed25519.PublicKey
}

// NewChain crée un nouveau journal avec un bloc genesis.
func NewChain(pubKey ed25519.PublicKey, privKey ed25519.PrivateKey) *Chain {
	genesis := Block{
		Index:     0,
		Timestamp: time.Now().Unix(),
		PrevHash:  [32]byte{},
		Type:      BlockSetting,
		Data:      []byte("fox:genesis"),
	}

	hash := genesis.Hash()
	genesis.Signature = ed25519.Sign(privKey, hash[:])

	return &Chain{
		Blocks: []Block{genesis},
		pubKey: pubKey,
	}
}

// Append ajoute un bloc au journal.
func (c *Chain) Append(blockType BlockType, encryptedData []byte, privKey ed25519.PrivateKey) *Block {
	last := c.Blocks[len(c.Blocks)-1]

	block := Block{
		Index:     last.Index + 1,
		Timestamp: time.Now().Unix(),
		PrevHash:  last.Hash(),
		Type:      blockType,
		Data:      encryptedData,
	}

	hash := block.Hash()
	block.Signature = ed25519.Sign(privKey, hash[:])

	c.Blocks = append(c.Blocks, block)
	return &block
}

// Verify vérifie l'intégrité de toute la chaîne.
func (c *Chain) Verify() bool {
	for i, block := range c.Blocks {
		// Vérifier le chaînage
		if i > 0 {
			prevHash := c.Blocks[i-1].Hash()
			if block.PrevHash != prevHash {
				return false
			}
		}

		// Vérifier la signature
		hash := block.Hash()
		if !ed25519.Verify(c.pubKey, hash[:], block.Signature) {
			return false
		}
	}
	return true
}
