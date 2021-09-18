package testnet

import (
	"encoding/hex"
	"github.com/tevino/abool"
	"math/rand"
	"pandora-pay/addresses"
	"pandora-pay/blockchain"
	"pandora-pay/blockchain/data/accounts"
	"pandora-pay/blockchain/data/accounts/account"
	plain_accounts "pandora-pay/blockchain/data/plain-accounts"
	plain_account "pandora-pay/blockchain/data/plain-accounts/plain-account"
	"pandora-pay/blockchain/data/registrations"
	"pandora-pay/blockchain/transactions/transaction"
	transaction_simple "pandora-pay/blockchain/transactions/transaction/transaction-simple"
	transaction_simple_parts "pandora-pay/blockchain/transactions/transaction/transaction-simple/transaction-simple-parts"
	"pandora-pay/config"
	"pandora-pay/config/config_stake"
	"pandora-pay/cryptography/crypto"
	"pandora-pay/gui"
	"pandora-pay/helpers"
	"pandora-pay/mempool"
	"pandora-pay/recovery"
	"pandora-pay/store"
	store_db_interface "pandora-pay/store/store-db/store-db-interface"
	transactions_builder "pandora-pay/transactions-builder"
	"pandora-pay/transactions-builder/wizard"
	"pandora-pay/wallet"
	wallet_address "pandora-pay/wallet/address"
	"time"
)

type Testnet struct {
	wallet              *wallet.Wallet
	mempool             *mempool.Mempool
	chain               *blockchain.Blockchain
	transactionsBuilder *transactions_builder.TransactionsBuilder
	nodes               uint64
}

func (testnet *Testnet) testnetCreateClaimTx(reg bool, amount uint64) (tx *transaction.Transaction, err error) {

	addr, err := testnet.wallet.GetWalletAddress(0)
	if err != nil {
		return
	}

	var registrationSignature []byte
	if !reg {
		registrationSignature = addr.Registration
	}

	if tx, err = testnet.transactionsBuilder.CreateClaimTx(addr.AddressEncoded, 0, []*transaction_simple_parts.TransactionSimpleOutput{
		{
			Amount:                amount,
			PublicKey:             addr.PublicKey,
			HasRegistration:       !reg,
			RegistrationSignature: registrationSignature,
		},
	}, &wizard.TransactionsWizardData{nil, false}, &wizard.TransactionsWizardFee{0, 0, true}, true, true, true, func(string) {}); err != nil {
		return nil, err
	}

	gui.GUI.Info("Claim tx was created: " + hex.EncodeToString(tx.Bloom.Hash))
	return
}

func (testnet *Testnet) testnetCreateUnstakeTx(blockHeight uint64, amount uint64) (tx *transaction.Transaction, err error) {

	addr, err := testnet.wallet.GetWalletAddress(0)
	if err != nil {
		return
	}

	if tx, err = testnet.transactionsBuilder.CreateUnstakeTx(addr.AddressEncoded, 0, amount, &wizard.TransactionsWizardData{nil, false}, &wizard.TransactionsWizardFee{0, 0, true}, true, true, true, func(string) {}); tx != nil {
		return nil, err
	}

	gui.GUI.Info("Unstake tx was created: " + hex.EncodeToString(tx.Bloom.Hash))
	return
}

func (testnet *Testnet) testnetCreateTransfersNewWallets(blockHeight uint64) (tx *transaction.Transaction, err error) {

	dsts := []string{}
	dstsAmounts := []uint64{}
	dstsTokens := [][]byte{}
	for i := uint64(0); i < testnet.nodes; i++ {
		if uint64(testnet.wallet.GetAddressesCount()) <= i+1 {
			if _, err = testnet.wallet.AddNewAddress(true); err != nil {
				return
			}
		}

		var addr *wallet_address.WalletAddress
		addr, err = testnet.wallet.GetWalletAddress(int(i + 1))
		if err != nil {
			return
		}

		dsts = append(dsts, addr.AddressEncoded)
		dstsAmounts = append(dstsAmounts, config_stake.GetRequiredStake(blockHeight))
		dstsTokens = append(dstsTokens, config.NATIVE_TOKEN_FULL)
	}

	addr, err := testnet.wallet.GetWalletAddress(0)
	if err != nil {
		return
	}

	if tx, err = testnet.transactionsBuilder.CreateZetherTx([]string{addr.AddressEncoded}, 0, config.NATIVE_TOKEN_FULL, []uint64{testnet.nodes * config_stake.GetRequiredStake(blockHeight)}, dsts, dstsAmounts, &wizard.TransactionsWizardData{}, &wizard.TransactionsWizardFee{0, 0, true}, true, true, true, func(string) {}); err != nil {
		return nil, err
	}

	gui.GUI.Info("Create Transfers Tx: ", tx.TransactionBaseInterface.(*transaction_simple.TransactionSimple).Nonce, hex.EncodeToString(tx.Bloom.Hash))
	return
}

func (testnet *Testnet) testnetCreateTransfers(blockHeight uint64) (tx *transaction.Transaction, err error) {

	dsts := []string{}
	dstsAmounts := []uint64{}
	dstsTokens := [][]byte{}

	count := rand.Intn(10) + 1
	sum := uint64(0)
	for i := 0; i < count; i++ {
		privateKey := addresses.GenerateNewPrivateKey()

		var addr *addresses.Address
		if addr, err = privateKey.GenerateAddress(false, 0, helpers.EmptyBytes(0)); err != nil {
			return
		}

		dsts = append(dsts, addr.EncodeAddr())
		amount := uint64(rand.Int63n(6))
		dstsAmounts = append(dstsAmounts, amount)
		dstsTokens = append(dstsTokens, config.NATIVE_TOKEN_FULL)
		sum += amount
	}

	addr, err := testnet.wallet.GetWalletAddress(0)
	if err != nil {
		return
	}

	if tx, err = testnet.transactionsBuilder.CreateZetherTx([]string{addr.AddressEncoded}, 0, config.NATIVE_TOKEN_FULL, []uint64{sum}, dsts, dstsAmounts, &wizard.TransactionsWizardData{}, &wizard.TransactionsWizardFee{0, 0, true}, true, true, true, func(string) {}); err != nil {
		return nil, err
	}

	gui.GUI.Info("Create Transfers Tx: ", tx.TransactionBaseInterface.(*transaction_simple.TransactionSimple).Nonce, hex.EncodeToString(tx.Bloom.Hash))
	return
}

func (testnet *Testnet) run() {

	updateChannel := testnet.chain.UpdateNewChain.AddListener()
	defer testnet.chain.UpdateNewChain.RemoveChannel(updateChannel)

	creatingTransactions := abool.New()

	for {

		blockHeightReceived, ok := <-updateChannel
		if !ok {
			return
		}

		blockHeight := blockHeightReceived.(uint64)
		syncTime := testnet.chain.Sync.GetSyncTime()

		recovery.SafeGo(func() {

			gui.GUI.Log("UpdateNewChain received! 1")
			defer gui.GUI.Log("UpdateNewChain received! DONE")

			err := func() (err error) {

				if blockHeight == 20 {
					if _, err = testnet.testnetCreateUnstakeTx(blockHeight, testnet.nodes*config_stake.GetRequiredStake(blockHeight)); err != nil {
						return
					}
				}
				if blockHeight == 30 {
					if _, err = testnet.testnetCreateTransfersNewWallets(blockHeight); err != nil {
						return
					}
				}

				if blockHeight >= 40 && syncTime != 0 {

					var addr *wallet_address.WalletAddress
					addr, err = testnet.wallet.GetWalletAddress(0)
					if err != nil {
						return
					}

					publicKey := addr.PublicKey

					var delegatedStakeAvailable, delegatedUnstakePending, claimable uint64
					var balanceHomo *crypto.ElGamal

					var acc *account.Account
					var plainAcc *plain_account.PlainAccount
					var reg bool

					gui.GUI.Log("UpdateNewChain received! 2")

					if err = store.StoreBlockchain.DB.View(func(reader store_db_interface.StoreDBTransactionInterface) (err error) {

						accsCollection := accounts.NewAccountsCollection(reader)
						regs := registrations.NewRegistrations(reader)

						accs, err := accsCollection.GetMap(config.NATIVE_TOKEN_FULL)
						if err != nil {
							return
						}
						if acc, err = accs.GetAccount(publicKey); err != nil {
							return
						}

						if reg, err = regs.Exists(string(publicKey)); err != nil {
							return
						}

						plainAccs := plain_accounts.NewPlainAccounts(reader)
						if plainAcc, err = plainAccs.GetPlainAccount(publicKey, blockHeight); err != nil {
							return
						}

						if acc != nil {
							balanceHomo = acc.GetBalance()
						}

						if plainAcc != nil {
							delegatedStakeAvailable = plainAcc.GetDelegatedStakeAvailable()
							delegatedUnstakePending, _ = plainAcc.ComputeDelegatedUnstakePending()
							claimable = plainAcc.Claimable
						}

						return
					}); err != nil {
						return
					}

					if acc != nil || plainAcc != nil {

						var balance uint64
						if acc != nil {
							if balance, err = testnet.wallet.DecodeBalanceByPublicKey(publicKey, balanceHomo, config.NATIVE_TOKEN_FULL, false); err != nil {
								return
							}
						}

						if claimable > 0 {
							if !testnet.mempool.ExistsTxSimpleVersion(addr.PublicKey, transaction_simple.SCRIPT_CLAIM) {
								if _, err = testnet.testnetCreateClaimTx(reg, claimable); err != nil {
									return
								}
							}
						} else if delegatedStakeAvailable > 0 && balance < delegatedStakeAvailable/4 && delegatedUnstakePending == 0 {
							if !testnet.mempool.ExistsTxSimpleVersion(addr.PublicKey, transaction_simple.SCRIPT_UNSTAKE) {
								if _, err = testnet.testnetCreateUnstakeTx(blockHeight, delegatedStakeAvailable/2-balance); err != nil {
									return
								}
							}
						} else {

							if creatingTransactions.IsNotSet() {
								creatingTransactions.Set()
								for {
									time.Sleep(time.Millisecond*time.Duration(rand.Intn(500)) + time.Millisecond*time.Duration(500))
									if testnet.mempool.CountInputTxs(addr.PublicKey) < 20 {
										if _, err = testnet.testnetCreateTransfers(blockHeight); err != nil {
											return
										}
									}
								}
							}
						}

					}

				}

				return
			}()

			if err != nil {
				gui.GUI.Error("Error creating testnet Tx", err)
				err = nil
			}

		})

	}
}

func TestnetInit(wallet *wallet.Wallet, mempool *mempool.Mempool, chain *blockchain.Blockchain, transactionsBuilder *transactions_builder.TransactionsBuilder) (testnet *Testnet) {

	testnet = &Testnet{
		wallet:              wallet,
		mempool:             mempool,
		chain:               chain,
		transactionsBuilder: transactionsBuilder,
		nodes:               uint64(config.CPU_THREADS),
	}

	recovery.SafeGo(testnet.run)

	return
}
