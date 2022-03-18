package address_balance_decryptor

import "context"

type addressBalanceDecryptorWorkResult struct {
	decryptedBalance uint64
	err              error
}

type addressBalanceDecryptorWork struct {
	encryptedBalance []byte
	privateKey       []byte
	previousValue    uint64
	wait             chan struct{}
	status           int32 //use atomic
	time             int64
	result           *addressBalanceDecryptorWorkResult
	ctx              context.Context
	statusCallback   func(string)
}

const (
	ADDRESS_BALANCE_DECRYPTED_INIT int32 = iota
	ADDRESS_BALANCE_DECRYPTED_PROCESSED
)