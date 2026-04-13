// Package foxota — module P2P pour la distribution décentralisée des mises à jour.
//
// Chaque instance HexaNaute peut servir de nœud de distribution.
// Les binaires sont vérifiés par hash SHA-256 (dans le manifest signé)
// quel que soit le pair source — aucune confiance n'est requise envers les pairs.
//
// Architecture :
//
//	Développeur → signe manifest → publie sur seed nodes
//	Seed nodes → distribuent via libp2p → tous les clients Fox
//	Chaque client devient redistributeur après réception
//
// Protocole : /fox/update/1.0.0
// Transport : QUIC (UDP chiffré, IPv6 natif)
package foxota

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

const (
	// Protocoles Fox P2P
	ProtoManifest = protocol.ID("/fox/manifest/1.0.0")
	ProtoBinary   = protocol.ID("/fox/binary/1.0.0")

	// Timeouts
	p2pStreamTimeout = 60 * time.Second
	p2pDialTimeout   = 10 * time.Second
)

// P2PNode est un nœud du réseau P2P Fox pour la distribution des MAJ.
type P2PNode struct {
	host    host.Host
	dht     *dht.IpfsDHT
	ctx     context.Context
	cancel  context.CancelFunc

	// Cache local des données servies
	mu             sync.RWMutex
	manifest       *Manifest
	manifestJSON   []byte
	binaryCache    map[string][]byte // hash -> binary data
	currentVersion uint64

	// Callbacks
	OnUpdateAvailable func(manifest *Manifest)
	OnPeerConnected   func(peerID string)
}

// NewP2PNode crée et démarre un nœud P2P.
func NewP2PNode(currentVersion uint64) (*P2PNode, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Créer le host libp2p avec transport QUIC (UDP chiffré, IPv6 natif)
	h, err := libp2p.New(
		libp2p.ListenAddrStrings(
			"/ip6/::/udp/0/quic-v1",       // IPv6 QUIC
			"/ip4/0.0.0.0/udp/0/quic-v1",  // IPv4 QUIC fallback
			"/ip6/::/tcp/0",               // IPv6 TCP fallback
			"/ip4/0.0.0.0/tcp/0",          // IPv4 TCP fallback
		),
		libp2p.EnableRelay(),
		libp2p.EnableHolePunching(),
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("foxota-p2p: échec création host: %w", err)
	}

	// Démarrer le DHT pour la découverte de pairs
	kdht, err := dht.New(ctx, h, dht.Mode(dht.ModeAutoServer))
	if err != nil {
		h.Close()
		cancel()
		return nil, fmt.Errorf("foxota-p2p: échec DHT: %w", err)
	}

	if err := kdht.Bootstrap(ctx); err != nil {
		h.Close()
		cancel()
		return nil, fmt.Errorf("foxota-p2p: échec bootstrap DHT: %w", err)
	}

	node := &P2PNode{
		host:           h,
		dht:            kdht,
		ctx:            ctx,
		cancel:         cancel,
		binaryCache:    make(map[string][]byte),
		currentVersion: currentVersion,
	}

	// Enregistrer les handlers de protocole
	h.SetStreamHandler(ProtoManifest, node.handleManifestRequest)
	h.SetStreamHandler(ProtoBinary, node.handleBinaryRequest)

	return node, nil
}

// PeerID retourne l'identifiant du nœud.
func (n *P2PNode) PeerID() string {
	return n.host.ID().String()
}

// Addresses retourne les adresses d'écoute du nœud.
func (n *P2PNode) Addresses() []string {
	addrs := make([]string, 0)
	for _, addr := range n.host.Addrs() {
		addrs = append(addrs, fmt.Sprintf("%s/p2p/%s", addr, n.host.ID()))
	}
	return addrs
}

// ServeUpdate rend une mise à jour disponible pour les pairs.
func (n *P2PNode) ServeUpdate(manifest *Manifest, binaries map[string][]byte) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	data, err := json.Marshal(manifest)
	if err != nil {
		return err
	}

	n.manifest = manifest
	n.manifestJSON = data

	// Stocker les binaires par hash
	for _, bin := range manifest.Binaries {
		hashBytes, _ := hex.DecodeString(bin.Hash)
		var hash [32]byte
		copy(hash[:], hashBytes)

		if binaryData, ok := binaries[bin.Filename]; ok {
			// Vérifier le hash
			actualHash := sha256.Sum256(binaryData)
			if actualHash == hash {
				n.binaryCache[bin.Hash] = binaryData
			}
		}
	}

	return nil
}

// RequestManifest demande le manifest à un pair.
func (n *P2PNode) RequestManifest(peerID peer.ID) (*Manifest, error) {
	ctx, cancel := context.WithTimeout(n.ctx, p2pDialTimeout)
	defer cancel()

	stream, err := n.host.NewStream(ctx, peerID, ProtoManifest)
	if err != nil {
		return nil, fmt.Errorf("connexion échouée: %w", err)
	}
	defer stream.Close()

	// Envoyer la requête
	stream.SetDeadline(time.Now().Add(p2pStreamTimeout))
	stream.Write([]byte("GET"))

	// Lire la réponse
	data, err := io.ReadAll(stream)
	if err != nil {
		return nil, err
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

// RequestBinary demande un binaire à un pair par son hash.
func (n *P2PNode) RequestBinary(peerID peer.ID, hash string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(n.ctx, p2pStreamTimeout)
	defer cancel()

	stream, err := n.host.NewStream(ctx, peerID, ProtoBinary)
	if err != nil {
		return nil, fmt.Errorf("connexion échouée: %w", err)
	}
	defer stream.Close()

	stream.SetDeadline(time.Now().Add(p2pStreamTimeout))

	// Envoyer le hash demandé
	stream.Write([]byte(hash))
	stream.CloseWrite()

	// Lire le binaire
	data, err := io.ReadAll(stream)
	if err != nil {
		return nil, err
	}

	// Vérifier le hash
	actualHash := sha256.Sum256(data)
	if hex.EncodeToString(actualHash[:]) != hash {
		return nil, fmt.Errorf("hash invalide ! attendu %s", hash)
	}

	// Mettre en cache pour redistribution
	n.mu.Lock()
	n.binaryCache[hash] = data
	n.mu.Unlock()

	return data, nil
}

// ConnectToPeer se connecte à un pair par son adresse multiaddr.
func (n *P2PNode) ConnectToPeer(addr string) error {
	peerInfo, err := peer.AddrInfoFromString(addr)
	if err != nil {
		return fmt.Errorf("adresse invalide: %w", err)
	}

	ctx, cancel := context.WithTimeout(n.ctx, p2pDialTimeout)
	defer cancel()

	return n.host.Connect(ctx, *peerInfo)
}

// ConnectedPeers retourne les pairs connectés.
func (n *P2PNode) ConnectedPeers() []string {
	peers := n.host.Network().Peers()
	result := make([]string, len(peers))
	for i, p := range peers {
		result[i] = p.String()
	}
	return result
}

// Close arrête le nœud P2P.
func (n *P2PNode) Close() error {
	n.cancel()
	n.dht.Close()
	return n.host.Close()
}

// ── Handlers de protocole ──

func (n *P2PNode) handleManifestRequest(stream network.Stream) {
	defer stream.Close()
	stream.SetDeadline(time.Now().Add(p2pStreamTimeout))

	// Lire la requête (on s'attend à "GET")
	buf := make([]byte, 16)
	stream.Read(buf)

	n.mu.RLock()
	data := n.manifestJSON
	n.mu.RUnlock()

	if data == nil {
		stream.Write([]byte("{}"))
		return
	}

	stream.Write(data)
}

func (n *P2PNode) handleBinaryRequest(stream network.Stream) {
	defer stream.Close()
	stream.SetDeadline(time.Now().Add(p2pStreamTimeout))

	// Lire le hash demandé
	hashBuf, err := io.ReadAll(stream)
	if err != nil {
		return
	}
	hash := string(hashBuf)

	n.mu.RLock()
	data, ok := n.binaryCache[hash]
	n.mu.RUnlock()

	if !ok {
		stream.Write([]byte("NOT_FOUND"))
		return
	}

	stream.Write(data)
}
