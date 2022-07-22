package signing

import (
	"sync"

	"github.com/ElrondNetwork/elrond-go-core/core/check"
	crypto "github.com/ElrondNetwork/elrond-go-crypto"
)

// ArgsSignatureHolder defines the arguments needed to create a new signature holder component
type ArgsSignatureHolder struct {
	PubKeys      []string
	OwnIndex     uint16
	PrivKey      crypto.PrivateKey
	SingleSigner crypto.SingleSigner
	MultiSigner  crypto.MultiSigner
	KeyGenerator crypto.KeyGenerator
}

type signatureHolderData struct {
	pubKeys   [][]byte
	privKey   crypto.PrivateKey
	sigShares [][]byte
	aggSig    []byte
	ownIndex  uint16
}

type signatureHolder struct {
	data           *signatureHolderData
	mutSigningData sync.RWMutex
	singleSigner   crypto.SingleSigner
	multiSigner    crypto.MultiSigner
	keyGen         crypto.KeyGenerator
}

// NewSignatureHolder will create a new signature holder component
func NewSignatureHolder(args ArgsSignatureHolder) (*signatureHolder, error) {
	err := checkArgs(args)
	if err != nil {
		return nil, err
	}

	sigSharesSize := uint16(len(args.PubKeys))
	sigShares := make([][]byte, sigSharesSize)
	pk, err := convertStringsToPubKeysBytes(args.PubKeys)
	if err != nil {
		return nil, err
	}

	data := &signatureHolderData{
		pubKeys:   pk,
		privKey:   args.PrivKey,
		sigShares: sigShares,
		ownIndex:  args.OwnIndex,
	}

	return &signatureHolder{
		data:           data,
		mutSigningData: sync.RWMutex{},
		singleSigner:   args.SingleSigner,
		multiSigner:    args.MultiSigner,
		keyGen:         args.KeyGenerator,
	}, nil
}

func checkArgs(args ArgsSignatureHolder) error {
	if check.IfNil(args.SingleSigner) {
		return ErrNilSingleSigner
	}
	if check.IfNil(args.MultiSigner) {
		return ErrNilMultiSigner
	}
	if check.IfNil(args.PrivKey) {
		return ErrNilPrivateKey
	}
	if check.IfNil(args.KeyGenerator) {
		return ErrNilKeyGenerator
	}
	if len(args.PubKeys) == 0 {
		return ErrNoPublicKeySet
	}
	if args.OwnIndex >= uint16(len(args.PubKeys)) {
		return ErrIndexOutOfBounds
	}

	return nil
}

// Create generated a signature holder component and initializes corresponding fields
func (sh *signatureHolder) Create(pubKeys []string, index uint16) (*signatureHolder, error) {
	sh.mutSigningData.RLock()
	privKey := sh.data.privKey
	sh.mutSigningData.RUnlock()

	args := ArgsSignatureHolder{
		PubKeys:      pubKeys,
		PrivKey:      privKey,
		SingleSigner: sh.singleSigner,
		MultiSigner:  sh.multiSigner,
		KeyGenerator: sh.keyGen,
	}
	return NewSignatureHolder(args)
}

// Reset resets the data inside the signature holder component
func (sh *signatureHolder) Reset(pubKeys []string, index uint16) error {
	if pubKeys == nil {
		return ErrNilPublicKeys
	}

	if index >= uint16(len(pubKeys)) {
		return ErrIndexOutOfBounds
	}

	sigSharesSize := uint16(len(pubKeys))
	sigShares := make([][]byte, sigSharesSize)
	pk, err := convertStringsToPubKeysBytes(pubKeys)
	if err != nil {
		return err
	}

	sh.mutSigningData.Lock()
	defer sh.mutSigningData.Unlock()

	privKey := sh.data.privKey

	data := &signatureHolderData{
		pubKeys:   pk,
		privKey:   privKey,
		sigShares: sigShares,
		ownIndex:  index,
	}

	sh.data = data

	return nil
}

// CreateSignatureShare returns a signature over a message
func (sh *signatureHolder) CreateSignatureShare(message []byte, _ []byte) ([]byte, error) {
	if message == nil {
		return nil, ErrNilMessage
	}

	sh.mutSigningData.Lock()
	defer sh.mutSigningData.Unlock()

	privKeyBytes, err := sh.data.privKey.ToByteArray()
	if err != nil {
		return nil, err
	}

	sigShareBytes, err := sh.multiSigner.CreateSignatureShare(privKeyBytes, message)
	if err != nil {
		return nil, err
	}

	sh.data.sigShares[sh.data.ownIndex] = sigShareBytes

	return sigShareBytes, nil
}

// VerifySignatureShare will verify the signature share based on the specified index
func (sh *signatureHolder) VerifySignatureShare(index uint16, sig []byte, message []byte, _ []byte) error {
	if sig == nil {
		return ErrNilSignature
	}

	sh.mutSigningData.Lock()
	defer sh.mutSigningData.Unlock()

	indexOutOfBounds := index >= uint16(len(sh.data.pubKeys))
	if indexOutOfBounds {
		return ErrIndexOutOfBounds
	}

	pubKey := sh.data.pubKeys[index]

	return sh.multiSigner.VerifySignatureShare(pubKey, message, sig)
}

// StoreSignatureShare stores the partial signature of the signer with specified position
func (sh *signatureHolder) StoreSignatureShare(index uint16, sig []byte) error {
	// TODO: evaluate verifying if sig bytes is a valid BLS signature
	if sig == nil {
		return ErrNilSignature
	}

	sh.mutSigningData.Lock()
	defer sh.mutSigningData.Unlock()

	if int(index) >= len(sh.data.sigShares) {
		return ErrIndexOutOfBounds
	}

	sh.data.sigShares[index] = sig

	return nil
}

// SignatureShare returns the partial signature set for given index
func (sh *signatureHolder) SignatureShare(index uint16) ([]byte, error) {
	sh.mutSigningData.Lock()
	defer sh.mutSigningData.Unlock()

	if int(index) >= len(sh.data.sigShares) {
		return nil, ErrIndexOutOfBounds
	}

	if sh.data.sigShares[index] == nil {
		return nil, ErrNilElement
	}

	return sh.data.sigShares[index], nil
}

// not concurrent safe, should be used under RLock mutex
func (sh *signatureHolder) isIndexInBitmap(index uint16, bitmap []byte) error {
	indexOutOfBounds := index >= uint16(len(sh.data.pubKeys))
	if indexOutOfBounds {
		return ErrIndexOutOfBounds
	}

	indexNotInBitmap := bitmap[index/8]&(1<<uint8(index%8)) == 0
	if indexNotInBitmap {
		return ErrIndexNotSelected
	}

	return nil
}

// AggregateSigs aggregates all collected partial signatures
func (sh *signatureHolder) AggregateSigs(bitmap []byte) ([]byte, error) {
	if bitmap == nil {
		return nil, ErrNilBitmap
	}

	sh.mutSigningData.Lock()
	defer sh.mutSigningData.Unlock()

	maxFlags := len(bitmap) * 8
	flagsMismatch := maxFlags < len(sh.data.pubKeys)
	if flagsMismatch {
		return nil, ErrBitmapMismatch
	}

	signatures := make([][]byte, 0, len(sh.data.sigShares))
	pubKeysSigners := make([][]byte, 0, len(sh.data.sigShares))

	for i := range sh.data.sigShares {
		err := sh.isIndexInBitmap(uint16(i), bitmap)
		if err != nil {
			continue
		}

		signatures = append(signatures, sh.data.sigShares[i])
		pubKeysSigners = append(pubKeysSigners, sh.data.pubKeys[i])
	}

	return sh.multiSigner.AggregateSigs(pubKeysSigners, signatures)
}

// SetAggregatedSig sets the aggregated signature
func (sh *signatureHolder) SetAggregatedSig(aggSig []byte) error {
	sh.mutSigningData.Lock()
	defer sh.mutSigningData.Unlock()

	sh.data.aggSig = aggSig

	return nil
}

// Verify verifies the aggregated signature by checking that aggregated signature is valid with respect
// to aggregated public keys.
func (sh *signatureHolder) Verify(message []byte, bitmap []byte) error {
	if bitmap == nil {
		return ErrNilBitmap
	}

	sh.mutSigningData.Lock()
	defer sh.mutSigningData.Unlock()

	maxFlags := len(bitmap) * 8
	flagsMismatch := maxFlags < len(sh.data.pubKeys)
	if flagsMismatch {
		return ErrBitmapMismatch
	}

	pubKeys := make([][]byte, 0)
	for i := range sh.data.pubKeys {
		err := sh.isIndexInBitmap(uint16(i), bitmap)
		if err != nil {
			continue
		}

		pubKeys = append(pubKeys, sh.data.pubKeys[i])
	}

	return sh.multiSigner.VerifyAggregatedSig(pubKeys, message, sh.data.aggSig)
}

func convertStringsToPubKeysBytes(pubKeys []string) ([][]byte, error) {
	pk := make([][]byte, 0, len(pubKeys))

	for _, pubKeyStr := range pubKeys {
		if pubKeyStr == "" {
			return nil, ErrEmptyPubKeyString
		}

		pubKey := []byte(pubKeyStr)
		pk = append(pk, pubKey)
	}

	return pk, nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (sh *signatureHolder) IsInterfaceNil() bool {
	return sh == nil
}
