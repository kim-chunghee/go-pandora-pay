package block_complete

import (
	"bytes"
	"errors"
)

type BlockCompleteBloom struct {
	Serialized                []byte `json:"-" msgpack:"-"`
	Size                      uint64 `json:"size" msgpack:"size"`
	merkleTreeVerified        bool
	bloomedSize               bool
	bloomedMerkleTreeVerified bool
}

func (blkComplete *BlockComplete) BloomCompleteBySerialized(serialized []byte) error {
	if err := blkComplete.validate(); err != nil {
		return err
	}

	blkComplete.BloomBlkComplete = &BlockCompleteBloom{
		Serialized:  serialized,
		Size:        uint64(len(serialized)),
		bloomedSize: true,
	}

	return nil
}

func (blkComplete *BlockComplete) BloomCompleteManual() {
	serialized := blkComplete.SerializeManualToBytes()
	blkComplete.BloomBlkComplete = &BlockCompleteBloom{
		Serialized:                serialized,
		Size:                      uint64(len(serialized)),
		bloomedSize:               true,
		merkleTreeVerified:        true,
		bloomedMerkleTreeVerified: true,
	}
}

func (blkComplete *BlockComplete) BloomAll() (err error) {

	for _, tx := range blkComplete.Txs {
		if err = tx.BloomAll(); err != nil {
			return
		}
	}

	if err = blkComplete.Block.BloomNow(); err != nil {
		return
	}
	if err = blkComplete.BloomNow(); err != nil {
		return
	}

	return
}

func (blkComplete *BlockComplete) BloomNow() error {

	if blkComplete.BloomBlkComplete == nil {
		if err := blkComplete.validate(); err != nil {
			return err
		}
		blkComplete.BloomBlkComplete = new(BlockCompleteBloom)
	}

	if !blkComplete.BloomBlkComplete.bloomedSize {
		blkComplete.BloomBlkComplete.Serialized = blkComplete.SerializeManualToBytes()
		blkComplete.BloomBlkComplete.Size = uint64(len(blkComplete.BloomBlkComplete.Serialized))
		blkComplete.BloomBlkComplete.bloomedSize = true
	}
	if !blkComplete.BloomBlkComplete.bloomedMerkleTreeVerified {
		blkComplete.BloomBlkComplete.merkleTreeVerified = bytes.Equal(blkComplete.MerkleHash(), blkComplete.Block.MerkleHash)
		if !blkComplete.BloomBlkComplete.merkleTreeVerified {
			return errors.New("Verify Merkle Hash failed")
		}
		blkComplete.BloomBlkComplete.bloomedMerkleTreeVerified = true
	}

	return nil
}

func (blkCompleteBloom *BlockCompleteBloom) verifyIfBloomed() error {
	if !blkCompleteBloom.bloomedSize || !blkCompleteBloom.bloomedMerkleTreeVerified {
		return errors.New("block complete was not bloomed")
	}
	if blkCompleteBloom.Serialized == nil {
		return errors.New("block complete serialized was not read")
	}
	if blkCompleteBloom.Size == 0 {
		return errors.New("block complete size was not bloomed")
	}
	if !blkCompleteBloom.merkleTreeVerified {
		return errors.New("Verify Merkle Hash failed")
	}
	return nil
}
