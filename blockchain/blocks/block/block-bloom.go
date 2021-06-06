package block

import (
	"errors"
	"pandora-pay/cryptography"
	"pandora-pay/cryptography/ecdsa"
	"pandora-pay/helpers"
)

type BlockBloom struct {
	Hash                       helpers.HexBytes `json:"hash"`
	KernelHash                 helpers.HexBytes `json:"kernelHash"`
	DelegatedPublicKeyHash     helpers.HexBytes `json:"delegatedPublicKeyHash"`
	DelegatedSignatureVerified bool             `json:"delegatedSignatureVerified"`
	bloomed                    bool             `json:"-"`
}

func (blk *Block) BloomNow() (err error) {

	if blk.Bloom != nil {
		return
	}

	bloom := new(BlockBloom)

	bloom.Hash = blk.computeHash()
	bloom.KernelHash = blk.ComputeKernelHash()
	hashForSignature := blk.SerializeForSigning()

	delegatedPublicKey, err := ecdsa.EcrecoverCompressed(hashForSignature, blk.Signature)
	if err != nil {
		return
	}

	bloom.DelegatedPublicKeyHash = cryptography.ComputePublicKeyHash(delegatedPublicKey)

	bloom.DelegatedSignatureVerified = ecdsa.VerifySignature(delegatedPublicKey, hashForSignature, blk.Signature[0:64])
	if !bloom.DelegatedSignatureVerified {
		return errors.New("BLock signature is invalid")
	}

	bloom.bloomed = true
	blk.Bloom = bloom
	return
}

func (bloom *BlockBloom) verifyIfBloomed() error {
	if !bloom.bloomed {
		return errors.New("Bloom was not validated")
	}
	return nil
}