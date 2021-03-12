package forging

import (
	"bytes"
	bolt "go.etcd.io/bbolt"
	"pandora-pay/addresses"
	"pandora-pay/blockchain/accounts"
	"pandora-pay/blockchain/accounts/account"
	"pandora-pay/store"
	"sync"
)

type ForgingWallet struct {
	addresses    []*ForgingWalletAddress
	addressesMap map[string]*ForgingWalletAddress

	sync.RWMutex
}

type ForgingWalletAddress struct {
	delegatedPrivateKey *addresses.PrivateKey
	delegatedPublicKey  []byte //33 byte

	publicKeyHash []byte //20byte

	account *account.Account
}

type ForgingWalletAddressRequired struct {
	publicKeyHash []byte //20 byte
	wallet        *ForgingWalletAddress
	stakingAmount uint64
}

func (w *ForgingWallet) AddWallet(delegatedPub []byte, delegatedPriv []byte, pubKeyHash []byte) {

	w.Lock()
	defer w.Unlock()

	private := addresses.PrivateKey{Key: delegatedPriv}

	store.StoreBlockchain.DB.View(func(tx *bolt.Tx) (err error) {

		accs := accounts.NewAccounts(tx)

		acc := accs.GetAccount(pubKeyHash)

		address := ForgingWalletAddress{
			&private,
			delegatedPub,
			pubKeyHash,
			acc,
		}
		w.addresses = append(w.addresses, &address)
		w.addressesMap[string(pubKeyHash)] = &address

		return
	})

}

func (w *ForgingWallet) UpdateBalanceChanges(accs *accounts.Accounts) {

	w.Lock()
	defer w.Unlock()

	for k, v := range accs.HashMap.Committed {

		if w.addressesMap[k] != nil {

			if v.Commit == "update" {
				w.addressesMap[k].account = new(account.Account)
				w.addressesMap[k].account.Deserialize(v.Data)
			} else if v.Commit == "delete" {
				w.addressesMap[k].account = nil
			}

		}

	}

}

func (w *ForgingWallet) RemoveWallet(delegatedPublicKey []byte) { //33 byte

	w.Lock()
	defer w.Unlock()

	for i, address := range w.addresses {
		if bytes.Equal(address.delegatedPublicKey, delegatedPublicKey) {
			w.addresses = append(w.addresses[:i], w.addresses[:i+1]...)
			return
		}
	}

}

func (w *ForgingWallet) loadBalances() error {

	w.Lock()
	defer w.Unlock()

	return store.StoreBlockchain.DB.View(func(tx *bolt.Tx) (err error) {

		accs := accounts.NewAccounts(tx)

		for _, address := range w.addresses {

			account := accs.GetAccount(address.publicKeyHash)
			address.account = account

		}

		return
	})

}