package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"math/big"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nspcc-dev/neo-go/internal/testchain"
	"github.com/nspcc-dev/neo-go/pkg/config"
	"github.com/nspcc-dev/neo-go/pkg/core"
	"github.com/nspcc-dev/neo-go/pkg/core/fee"
	"github.com/nspcc-dev/neo-go/pkg/core/native/nativenames"
	"github.com/nspcc-dev/neo-go/pkg/core/transaction"
	"github.com/nspcc-dev/neo-go/pkg/crypto/hash"
	"github.com/nspcc-dev/neo-go/pkg/crypto/keys"
	"github.com/nspcc-dev/neo-go/pkg/encoding/address"
	"github.com/nspcc-dev/neo-go/pkg/io"
	"github.com/nspcc-dev/neo-go/pkg/rpc/client"
	"github.com/nspcc-dev/neo-go/pkg/rpc/client/nns"
	"github.com/nspcc-dev/neo-go/pkg/rpc/response/result"
	"github.com/nspcc-dev/neo-go/pkg/smartcontract"
	"github.com/nspcc-dev/neo-go/pkg/smartcontract/callflag"
	"github.com/nspcc-dev/neo-go/pkg/smartcontract/manifest"
	"github.com/nspcc-dev/neo-go/pkg/smartcontract/trigger"
	"github.com/nspcc-dev/neo-go/pkg/util"
	"github.com/nspcc-dev/neo-go/pkg/vm"
	"github.com/nspcc-dev/neo-go/pkg/vm/emit"
	"github.com/nspcc-dev/neo-go/pkg/vm/opcode"
	"github.com/nspcc-dev/neo-go/pkg/vm/stackitem"
	"github.com/nspcc-dev/neo-go/pkg/wallet"
	"github.com/stretchr/testify/require"
)

func TestClient_NEP17(t *testing.T) {
	chain, rpcSrv, httpSrv := initServerWithInMemoryChain(t)
	defer chain.Close()
	defer rpcSrv.Shutdown()

	c, err := client.New(context.Background(), httpSrv.URL, client.Options{})
	require.NoError(t, err)
	require.NoError(t, c.Init())

	h, err := util.Uint160DecodeStringLE(testContractHash)
	require.NoError(t, err)

	t.Run("Decimals", func(t *testing.T) {
		d, err := c.NEP17Decimals(h)
		require.NoError(t, err)
		require.EqualValues(t, 2, d)
	})
	t.Run("TotalSupply", func(t *testing.T) {
		s, err := c.NEP17TotalSupply(h)
		require.NoError(t, err)
		require.EqualValues(t, 1_000_000, s)
	})
	t.Run("Symbol", func(t *testing.T) {
		sym, err := c.NEP17Symbol(h)
		require.NoError(t, err)
		require.Equal(t, "RUB", sym)
	})
	t.Run("TokenInfo", func(t *testing.T) {
		tok, err := c.NEP17TokenInfo(h)
		require.NoError(t, err)
		require.Equal(t, h, tok.Hash)
		require.Equal(t, "Rubl", tok.Name)
		require.Equal(t, "RUB", tok.Symbol)
		require.EqualValues(t, 2, tok.Decimals)
	})
	t.Run("BalanceOf", func(t *testing.T) {
		acc := testchain.PrivateKeyByID(0).GetScriptHash()
		b, err := c.NEP17BalanceOf(h, acc)
		require.NoError(t, err)
		require.EqualValues(t, 877, b)
	})
}

func TestAddNetworkFeeCalculateNetworkFee(t *testing.T) {
	chain, rpcSrv, httpSrv := initServerWithInMemoryChain(t)
	defer chain.Close()
	defer rpcSrv.Shutdown()
	const extraFee = 10
	var nonce uint32

	c, err := client.New(context.Background(), httpSrv.URL, client.Options{})
	require.NoError(t, err)
	require.NoError(t, c.Init())

	getAccounts := func(t *testing.T, n int) []*wallet.Account {
		accs := make([]*wallet.Account, n)
		var err error
		for i := range accs {
			accs[i], err = wallet.NewAccount()
			require.NoError(t, err)
		}
		return accs
	}

	feePerByte := chain.FeePerByte()

	t.Run("Invalid", func(t *testing.T) {
		tx := transaction.New([]byte{byte(opcode.PUSH1)}, 0)
		accs := getAccounts(t, 2)
		tx.Signers = []transaction.Signer{{
			Account: accs[0].PrivateKey().GetScriptHash(),
			Scopes:  transaction.CalledByEntry,
		}}
		require.Error(t, c.AddNetworkFee(tx, extraFee, accs[0], accs[1]))
	})
	t.Run("Simple", func(t *testing.T) {
		acc0 := wallet.NewAccountFromPrivateKey(testchain.PrivateKeyByID(0))
		check := func(t *testing.T, extraFee int64) {
			tx := transaction.New([]byte{byte(opcode.PUSH1)}, 0)
			tx.ValidUntilBlock = 25
			tx.Signers = []transaction.Signer{{
				Account: acc0.PrivateKey().GetScriptHash(),
				Scopes:  transaction.CalledByEntry,
			}}
			tx.Nonce = nonce
			nonce++

			tx.Scripts = []transaction.Witness{
				{VerificationScript: acc0.GetVerificationScript()},
			}
			actualCalculatedNetFee, err := c.CalculateNetworkFee(tx)
			require.NoError(t, err)

			tx.Scripts = nil
			require.NoError(t, c.AddNetworkFee(tx, extraFee, acc0))
			actual := tx.NetworkFee

			require.NoError(t, acc0.SignTx(testchain.Network(), tx))
			cFee, _ := fee.Calculate(chain.GetBaseExecFee(), acc0.Contract.Script)
			expected := int64(io.GetVarSize(tx))*feePerByte + cFee + extraFee

			require.Equal(t, expected, actual)
			require.Equal(t, expected, actualCalculatedNetFee+extraFee)
			err = chain.VerifyTx(tx)
			if extraFee < 0 {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		}

		t.Run("with extra fee", func(t *testing.T) {
			// check that calculated network fee with extra value is enough
			check(t, extraFee)
		})
		t.Run("without extra fee", func(t *testing.T) {
			// check that calculated network fee without extra value is enough
			check(t, 0)
		})
		t.Run("exactFee-1", func(t *testing.T) {
			// check that we don't add unexpected extra GAS
			check(t, -1)
		})
	})

	t.Run("Multi", func(t *testing.T) {
		acc0 := wallet.NewAccountFromPrivateKey(testchain.PrivateKeyByID(0))
		acc1 := wallet.NewAccountFromPrivateKey(testchain.PrivateKeyByID(0))
		err = acc1.ConvertMultisig(3, keys.PublicKeys{
			testchain.PrivateKeyByID(0).PublicKey(),
			testchain.PrivateKeyByID(1).PublicKey(),
			testchain.PrivateKeyByID(2).PublicKey(),
			testchain.PrivateKeyByID(3).PublicKey(),
		})
		require.NoError(t, err)
		check := func(t *testing.T, extraFee int64) {
			tx := transaction.New([]byte{byte(opcode.PUSH1)}, 0)
			tx.ValidUntilBlock = 25
			tx.Signers = []transaction.Signer{
				{
					Account: acc0.PrivateKey().GetScriptHash(),
					Scopes:  transaction.CalledByEntry,
				},
				{
					Account: hash.Hash160(acc1.Contract.Script),
					Scopes:  transaction.Global,
				},
			}
			tx.Nonce = nonce
			nonce++

			tx.Scripts = []transaction.Witness{
				{VerificationScript: acc0.GetVerificationScript()},
				{VerificationScript: acc1.GetVerificationScript()},
			}
			actualCalculatedNetFee, err := c.CalculateNetworkFee(tx)
			require.NoError(t, err)

			tx.Scripts = nil

			require.NoError(t, c.AddNetworkFee(tx, extraFee, acc0, acc1))
			actual := tx.NetworkFee

			require.NoError(t, acc0.SignTx(testchain.Network(), tx))
			tx.Scripts = append(tx.Scripts, transaction.Witness{
				InvocationScript:   testchain.Sign(tx),
				VerificationScript: acc1.Contract.Script,
			})
			cFee, _ := fee.Calculate(chain.GetBaseExecFee(), acc0.Contract.Script)
			cFeeM, _ := fee.Calculate(chain.GetBaseExecFee(), acc1.Contract.Script)
			expected := int64(io.GetVarSize(tx))*feePerByte + cFee + cFeeM + extraFee

			require.Equal(t, expected, actual)
			require.Equal(t, expected, actualCalculatedNetFee+extraFee)
			err = chain.VerifyTx(tx)
			if extraFee < 0 {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		}

		t.Run("with extra fee", func(t *testing.T) {
			// check that calculated network fee with extra value is enough
			check(t, extraFee)
		})
		t.Run("without extra fee", func(t *testing.T) {
			// check that calculated network fee without extra value is enough
			check(t, 0)
		})
		t.Run("exactFee-1", func(t *testing.T) {
			// check that we don't add unexpected extra GAS
			check(t, -1)
		})
	})
	t.Run("Contract", func(t *testing.T) {
		h, err := util.Uint160DecodeStringLE(verifyContractHash)
		require.NoError(t, err)
		priv := testchain.PrivateKeyByID(0)
		acc0 := wallet.NewAccountFromPrivateKey(priv)
		acc1 := wallet.NewAccountFromPrivateKey(priv) // contract account
		acc1.Contract.Deployed = true
		acc1.Contract.Script, err = base64.StdEncoding.DecodeString(verifyContractAVM)
		require.NoError(t, err)

		newTx := func(t *testing.T) *transaction.Transaction {
			tx := transaction.New([]byte{byte(opcode.PUSH1)}, 0)
			tx.ValidUntilBlock = chain.BlockHeight() + 10
			return tx
		}

		t.Run("Valid", func(t *testing.T) {
			check := func(t *testing.T, extraFee int64) {
				tx := newTx(t)
				tx.Signers = []transaction.Signer{
					{
						Account: acc0.PrivateKey().GetScriptHash(),
						Scopes:  transaction.CalledByEntry,
					},
					{
						Account: h,
						Scopes:  transaction.Global,
					},
				}
				// we need to fill standard verification scripts to use CalculateNetworkFee.
				tx.Scripts = []transaction.Witness{
					{VerificationScript: acc0.GetVerificationScript()},
					{},
				}
				actual, err := c.CalculateNetworkFee(tx)
				require.NoError(t, err)
				tx.Scripts = nil

				require.NoError(t, c.AddNetworkFee(tx, extraFee, acc0, acc1))
				require.NoError(t, acc0.SignTx(testchain.Network(), tx))
				tx.Scripts = append(tx.Scripts, transaction.Witness{})
				require.Equal(t, tx.NetworkFee, actual+extraFee)
				err = chain.VerifyTx(tx)
				if extraFee < 0 {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			}

			t.Run("with extra fee", func(t *testing.T) {
				// check that calculated network fee with extra value is enough
				check(t, extraFee)
			})
			t.Run("without extra fee", func(t *testing.T) {
				// check that calculated network fee without extra value is enough
				check(t, 0)
			})
			t.Run("exactFee-1", func(t *testing.T) {
				// check that we don't add unexpected extra GAS
				check(t, -1)
			})
		})
		t.Run("Invalid", func(t *testing.T) {
			tx := newTx(t)
			acc0, err := wallet.NewAccount()
			require.NoError(t, err)
			tx.Signers = []transaction.Signer{
				{
					Account: acc0.PrivateKey().GetScriptHash(),
					Scopes:  transaction.CalledByEntry,
				},
				{
					Account: h,
					Scopes:  transaction.Global,
				},
			}
			require.Error(t, c.AddNetworkFee(tx, 10, acc0, acc1))
		})
		t.Run("InvalidContract", func(t *testing.T) {
			tx := newTx(t)
			acc0 := wallet.NewAccountFromPrivateKey(priv)
			tx.Signers = []transaction.Signer{
				{
					Account: acc0.PrivateKey().GetScriptHash(),
					Scopes:  transaction.CalledByEntry,
				},
				{
					Account: util.Uint160{},
					Scopes:  transaction.Global,
				},
			}
			require.Error(t, c.AddNetworkFee(tx, 10, acc0, acc1))
		})
	})
}

func TestCalculateNetworkFee(t *testing.T) {
	chain, rpcSrv, httpSrv := initServerWithInMemoryChain(t)
	defer chain.Close()
	defer rpcSrv.Shutdown()
	const extraFee = 10

	c, err := client.New(context.Background(), httpSrv.URL, client.Options{})
	require.NoError(t, err)
	require.NoError(t, c.Init())

	t.Run("ContractWithArgs", func(t *testing.T) {
		check := func(t *testing.T, extraFee int64) {
			h, err := util.Uint160DecodeStringLE(verifyWithArgsContractHash)
			require.NoError(t, err)
			priv := testchain.PrivateKeyByID(0)
			acc0 := wallet.NewAccountFromPrivateKey(priv)
			tx := transaction.New([]byte{byte(opcode.PUSH1)}, 0)
			require.NoError(t, err)
			tx.ValidUntilBlock = chain.BlockHeight() + 10
			tx.Signers = []transaction.Signer{
				{
					Account: acc0.PrivateKey().GetScriptHash(),
					Scopes:  transaction.CalledByEntry,
				},
				{
					Account: h,
					Scopes:  transaction.Global,
				},
			}

			bw := io.NewBufBinWriter()
			emit.Bool(bw.BinWriter, false)
			emit.Int(bw.BinWriter, int64(4))
			emit.String(bw.BinWriter, "good_string") // contract's `verify` return `true` with this string
			require.NoError(t, bw.Err)
			contractInv := bw.Bytes()
			// we need to fill standard verification scripts to use CalculateNetworkFee.
			tx.Scripts = []transaction.Witness{
				{VerificationScript: acc0.GetVerificationScript()},
				{InvocationScript: contractInv},
			}
			tx.NetworkFee, err = c.CalculateNetworkFee(tx)
			require.NoError(t, err)
			tx.NetworkFee += extraFee
			tx.Scripts = nil

			require.NoError(t, acc0.SignTx(testchain.Network(), tx))
			tx.Scripts = append(tx.Scripts, transaction.Witness{InvocationScript: contractInv})
			err = chain.VerifyTx(tx)
			if extraFee < 0 {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		}

		t.Run("with extra fee", func(t *testing.T) {
			// check that calculated network fee with extra value is enough
			check(t, extraFee)
		})
		t.Run("without extra fee", func(t *testing.T) {
			// check that calculated network fee without extra value is enough
			check(t, 0)
		})
		t.Run("exactFee-1", func(t *testing.T) {
			// check that we don't add unexpected extra GAS
			check(t, -1)
		})
	})
}
func TestSignAndPushInvocationTx(t *testing.T) {
	chain, rpcSrv, httpSrv := initServerWithInMemoryChain(t)
	defer chain.Close()
	defer rpcSrv.Shutdown()

	c, err := client.New(context.Background(), httpSrv.URL, client.Options{})
	require.NoError(t, err)
	require.NoError(t, c.Init())

	priv0 := testchain.PrivateKeyByID(0)
	acc0 := wallet.NewAccountFromPrivateKey(priv0)

	verifyWithoutParamsCtr, err := util.Uint160DecodeStringLE(verifyContractHash)
	require.NoError(t, err)
	acc1 := &wallet.Account{
		Address: address.Uint160ToString(verifyWithoutParamsCtr),
		Contract: &wallet.Contract{
			Parameters: []wallet.ContractParam{},
			Deployed:   true,
		},
		Locked:  true,
		Default: false,
	}

	verifyWithParamsCtr, err := util.Uint160DecodeStringLE(verifyWithArgsContractHash)
	require.NoError(t, err)
	acc2 := &wallet.Account{
		Address: address.Uint160ToString(verifyWithParamsCtr),
		Contract: &wallet.Contract{
			Parameters: []wallet.ContractParam{
				{Name: "argString", Type: smartcontract.StringType},
				{Name: "argInt", Type: smartcontract.IntegerType},
				{Name: "argBool", Type: smartcontract.BoolType},
			},
			Deployed: true,
		},
		Locked:  true,
		Default: false,
	}

	priv3 := testchain.PrivateKeyByID(3)
	acc3 := wallet.NewAccountFromPrivateKey(priv3)

	check := func(t *testing.T, h util.Uint256) {
		mp := chain.GetMemPool()
		tx, ok := mp.TryGetValue(h)
		require.True(t, ok)
		require.Equal(t, h, tx.Hash())
		require.EqualValues(t, 30, tx.SystemFee)
	}

	t.Run("good", func(t *testing.T) {
		t.Run("signer0: sig", func(t *testing.T) {
			h, err := c.SignAndPushInvocationTx([]byte{byte(opcode.PUSH1)}, acc0, 30, 0, []client.SignerAccount{
				{
					Signer: transaction.Signer{
						Account: priv0.GetScriptHash(),
						Scopes:  transaction.CalledByEntry,
					},
					Account: acc0,
				},
			})
			require.NoError(t, err)
			check(t, h)
		})
		t.Run("signer0: sig; signer1: sig", func(t *testing.T) {
			h, err := c.SignAndPushInvocationTx([]byte{byte(opcode.PUSH1)}, acc0, 30, 0, []client.SignerAccount{
				{
					Signer: transaction.Signer{
						Account: priv0.GetScriptHash(),
						Scopes:  transaction.CalledByEntry,
					},
					Account: acc0,
				},
				{
					Signer: transaction.Signer{
						Account: priv3.GetScriptHash(),
						Scopes:  transaction.CalledByEntry,
					},
					Account: acc3,
				},
			})
			require.NoError(t, err)
			check(t, h)
		})
		t.Run("signer0: sig; signer1: contract-based paramless", func(t *testing.T) {
			h, err := c.SignAndPushInvocationTx([]byte{byte(opcode.PUSH1)}, acc0, 30, 0, []client.SignerAccount{
				{
					Signer: transaction.Signer{
						Account: priv0.GetScriptHash(),
						Scopes:  transaction.CalledByEntry,
					},
					Account: acc0,
				},
				{
					Signer: transaction.Signer{
						Account: verifyWithoutParamsCtr,
						Scopes:  transaction.CalledByEntry,
					},
					Account: acc1,
				},
			})
			require.NoError(t, err)
			check(t, h)
		})
	})
	t.Run("error", func(t *testing.T) {
		t.Run("signer0: sig; signer1: contract-based with params", func(t *testing.T) {
			_, err := c.SignAndPushInvocationTx([]byte{byte(opcode.PUSH1)}, acc0, 30, 0, []client.SignerAccount{
				{
					Signer: transaction.Signer{
						Account: priv0.GetScriptHash(),
						Scopes:  transaction.CalledByEntry,
					},
					Account: acc0,
				},
				{
					Signer: transaction.Signer{
						Account: verifyWithParamsCtr,
						Scopes:  transaction.CalledByEntry,
					},
					Account: acc2,
				},
			})
			require.Error(t, err)
		})
		t.Run("signer0: sig; signer1: locked sig", func(t *testing.T) {
			pk, err := keys.NewPrivateKey()
			require.NoError(t, err)
			acc4 := &wallet.Account{
				Address: address.Uint160ToString(pk.GetScriptHash()),
				Contract: &wallet.Contract{
					Script:     pk.PublicKey().GetVerificationScript(),
					Parameters: []wallet.ContractParam{{Name: "parameter0", Type: smartcontract.SignatureType}},
				},
			}
			_, err = c.SignAndPushInvocationTx([]byte{byte(opcode.PUSH1)}, acc0, 30, 0, []client.SignerAccount{
				{
					Signer: transaction.Signer{
						Account: priv0.GetScriptHash(),
						Scopes:  transaction.CalledByEntry,
					},
					Account: acc0,
				},
				{
					Signer: transaction.Signer{
						Account: util.Uint160{1, 2, 3},
						Scopes:  transaction.CalledByEntry,
					},
					Account: acc4,
				},
			})
			require.Error(t, err)
		})
	})
}

func TestSignAndPushP2PNotaryRequest(t *testing.T) {
	chain, rpcSrv, httpSrv := initServerWithInMemoryChainAndServices(t, false, true)
	defer chain.Close()
	defer rpcSrv.Shutdown()

	c, err := client.New(context.Background(), httpSrv.URL, client.Options{})
	require.NoError(t, err)
	acc, err := wallet.NewAccount()
	require.NoError(t, err)

	t.Run("client wasn't initialized", func(t *testing.T) {
		_, err := c.SignAndPushP2PNotaryRequest(transaction.New([]byte{byte(opcode.RET)}, 123), []byte{byte(opcode.RET)}, -1, 0, 100, acc)
		require.NotNil(t, err)
	})

	require.NoError(t, c.Init())
	t.Run("bad account address", func(t *testing.T) {
		_, err := c.SignAndPushP2PNotaryRequest(nil, nil, 0, 0, 0, &wallet.Account{Address: "not-an-addr"})
		require.NotNil(t, err)
	})

	t.Run("bad fallback script", func(t *testing.T) {
		_, err := c.SignAndPushP2PNotaryRequest(nil, []byte{byte(opcode.ASSERT)}, -1, 0, 0, acc)
		require.NotNil(t, err)
	})

	t.Run("too large fallbackValidFor", func(t *testing.T) {
		_, err := c.SignAndPushP2PNotaryRequest(nil, []byte{byte(opcode.RET)}, -1, 0, 141, acc)
		require.NotNil(t, err)
	})

	t.Run("good", func(t *testing.T) {
		sender := testchain.PrivateKeyByID(0) // owner of the deposit in testchain
		acc := wallet.NewAccountFromPrivateKey(sender)
		expected := transaction.Transaction{
			Attributes:      []transaction.Attribute{{Type: transaction.NotaryAssistedT, Value: &transaction.NotaryAssisted{NKeys: 1}}},
			Script:          []byte{byte(opcode.RET)},
			ValidUntilBlock: chain.BlockHeight() + 5,
			Signers:         []transaction.Signer{{Account: util.Uint160{1, 5, 9}}},
			Scripts: []transaction.Witness{{
				InvocationScript:   []byte{1, 4, 7},
				VerificationScript: []byte{3, 6, 9},
			}},
		}
		mainTx := expected
		_ = expected.Hash()
		req, err := c.SignAndPushP2PNotaryRequest(&mainTx, []byte{byte(opcode.RET)}, -1, 0, 6, acc)
		require.NoError(t, err)

		// check that request was correctly completed
		require.Equal(t, expected, *req.MainTransaction) // main tx should be the same
		require.ElementsMatch(t, []transaction.Attribute{
			{
				Type:  transaction.NotaryAssistedT,
				Value: &transaction.NotaryAssisted{NKeys: 0},
			},
			{
				Type:  transaction.NotValidBeforeT,
				Value: &transaction.NotValidBefore{Height: chain.BlockHeight()},
			},
			{
				Type:  transaction.ConflictsT,
				Value: &transaction.Conflicts{Hash: mainTx.Hash()},
			},
		}, req.FallbackTransaction.Attributes)
		require.Equal(t, []transaction.Signer{
			{Account: chain.GetNotaryContractScriptHash()},
			{Account: acc.PrivateKey().GetScriptHash()},
		}, req.FallbackTransaction.Signers)

		// it shouldn't be an error to add completed fallback to the chain
		w, err := wallet.NewWalletFromFile(notaryPath)
		require.NoError(t, err)
		ntr := w.Accounts[0]
		err = ntr.Decrypt(notaryPass, w.Scrypt)
		require.NoError(t, err)
		req.FallbackTransaction.Scripts[0] = transaction.Witness{
			InvocationScript:   append([]byte{byte(opcode.PUSHDATA1), 64}, ntr.PrivateKey().SignHashable(uint32(testchain.Network()), req.FallbackTransaction)...),
			VerificationScript: []byte{},
		}
		b := testchain.NewBlock(t, chain, 1, 0, req.FallbackTransaction)
		require.NoError(t, chain.AddBlock(b))
		appLogs, err := chain.GetAppExecResults(req.FallbackTransaction.Hash(), trigger.Application)
		require.NoError(t, err)
		require.Equal(t, 1, len(appLogs))
		appLog := appLogs[0]
		require.Equal(t, vm.HaltState, appLog.VMState)
		require.Equal(t, appLog.GasConsumed, req.FallbackTransaction.SystemFee)
	})
}

func TestCalculateNotaryFee(t *testing.T) {
	chain, rpcSrv, httpSrv := initServerWithInMemoryChain(t)
	defer chain.Close()
	defer rpcSrv.Shutdown()

	c, err := client.New(context.Background(), httpSrv.URL, client.Options{})
	require.NoError(t, err)

	t.Run("client not initialized", func(t *testing.T) {
		_, err := c.CalculateNotaryFee(0)
		require.NoError(t, err) // Do not require client initialisation for this.
	})
}

func TestPing(t *testing.T) {
	chain, rpcSrv, httpSrv := initServerWithInMemoryChain(t)
	defer chain.Close()

	c, err := client.New(context.Background(), httpSrv.URL, client.Options{})
	require.NoError(t, err)
	require.NoError(t, c.Init())

	require.NoError(t, c.Ping())
	rpcSrv.Shutdown()
	httpSrv.Close()
	require.Error(t, c.Ping())
}

func TestCreateTxFromScript(t *testing.T) {
	chain, rpcSrv, httpSrv := initServerWithInMemoryChain(t)
	defer chain.Close()
	defer rpcSrv.Shutdown()

	c, err := client.New(context.Background(), httpSrv.URL, client.Options{})
	require.NoError(t, err)
	require.NoError(t, c.Init())

	priv := testchain.PrivateKey(0)
	acc := wallet.NewAccountFromPrivateKey(priv)
	t.Run("NoSystemFee", func(t *testing.T) {
		tx, err := c.CreateTxFromScript([]byte{byte(opcode.PUSH1)}, acc, -1, 10, nil)
		require.NoError(t, err)
		require.True(t, tx.ValidUntilBlock > chain.BlockHeight())
		require.EqualValues(t, 30, tx.SystemFee) // PUSH1
		require.True(t, len(tx.Signers) == 1)
		require.Equal(t, acc.PrivateKey().GetScriptHash(), tx.Signers[0].Account)
	})
	t.Run("ProvideSystemFee", func(t *testing.T) {
		tx, err := c.CreateTxFromScript([]byte{byte(opcode.PUSH1)}, acc, 123, 10, nil)
		require.NoError(t, err)
		require.True(t, tx.ValidUntilBlock > chain.BlockHeight())
		require.EqualValues(t, 123, tx.SystemFee)
		require.True(t, len(tx.Signers) == 1)
		require.Equal(t, acc.PrivateKey().GetScriptHash(), tx.Signers[0].Account)
	})
}

func TestCreateNEP17TransferTx(t *testing.T) {
	chain, rpcSrv, httpSrv := initServerWithInMemoryChain(t)
	defer chain.Close()
	defer rpcSrv.Shutdown()

	c, err := client.New(context.Background(), httpSrv.URL, client.Options{})
	require.NoError(t, err)
	require.NoError(t, c.Init())

	priv := testchain.PrivateKeyByID(0)
	acc := wallet.NewAccountFromPrivateKey(priv)

	gasContractHash, err := c.GetNativeContractHash(nativenames.Gas)
	require.NoError(t, err)

	t.Run("default scope", func(t *testing.T) {
		tx, err := c.CreateNEP17TransferTx(acc, util.Uint160{}, gasContractHash, 1000, 0, nil, nil)
		require.NoError(t, err)
		require.NoError(t, acc.SignTx(testchain.Network(), tx))
		require.NoError(t, chain.VerifyTx(tx))
		ic := chain.GetTestVM(trigger.Application, tx, nil)
		ic.VM.LoadScriptWithFlags(tx.Script, callflag.All)
		require.NoError(t, ic.VM.Run())
	})
	t.Run("none scope", func(t *testing.T) {
		_, err := c.CreateNEP17TransferTx(acc, util.Uint160{}, gasContractHash, 1000, 0, nil, []client.SignerAccount{{
			Signer: transaction.Signer{
				Account: priv.PublicKey().GetScriptHash(),
				Scopes:  transaction.None,
			},
		}})
		require.Error(t, err)
	})
	t.Run("customcontracts scope", func(t *testing.T) {
		tx, err := c.CreateNEP17TransferTx(acc, util.Uint160{}, gasContractHash, 1000, 0, nil, []client.SignerAccount{{
			Signer: transaction.Signer{
				Account:          priv.PublicKey().GetScriptHash(),
				Scopes:           transaction.CustomContracts,
				AllowedContracts: []util.Uint160{gasContractHash},
			},
		}})
		require.NoError(t, err)
		require.NoError(t, acc.SignTx(testchain.Network(), tx))
		require.NoError(t, chain.VerifyTx(tx))
		ic := chain.GetTestVM(trigger.Application, tx, nil)
		ic.VM.LoadScriptWithFlags(tx.Script, callflag.All)
		require.NoError(t, ic.VM.Run())
	})
}

func TestInvokeVerify(t *testing.T) {
	chain, rpcSrv, httpSrv := initServerWithInMemoryChain(t)
	defer chain.Close()
	defer rpcSrv.Shutdown()

	c, err := client.New(context.Background(), httpSrv.URL, client.Options{})
	require.NoError(t, err)
	require.NoError(t, c.Init())

	contract, err := util.Uint160DecodeStringLE(verifyContractHash)
	require.NoError(t, err)

	t.Run("positive, with signer", func(t *testing.T) {
		res, err := c.InvokeContractVerify(contract, smartcontract.Params{}, []transaction.Signer{{Account: testchain.PrivateKeyByID(0).PublicKey().GetScriptHash()}})
		require.NoError(t, err)
		require.Equal(t, "HALT", res.State)
		require.Equal(t, 1, len(res.Stack))
		require.True(t, res.Stack[0].Value().(bool))
	})

	t.Run("positive, historic, by height, with signer", func(t *testing.T) {
		h := chain.BlockHeight() - 1
		res, err := c.InvokeContractVerifyAtHeight(h, contract, smartcontract.Params{}, []transaction.Signer{{Account: testchain.PrivateKeyByID(0).PublicKey().GetScriptHash()}})
		require.NoError(t, err)
		require.Equal(t, "HALT", res.State)
		require.Equal(t, 1, len(res.Stack))
		require.True(t, res.Stack[0].Value().(bool))
	})

	t.Run("positive, historic, by block, with signer", func(t *testing.T) {
		res, err := c.InvokeContractVerifyAtBlock(chain.GetHeaderHash(int(chain.BlockHeight())-1), contract, smartcontract.Params{}, []transaction.Signer{{Account: testchain.PrivateKeyByID(0).PublicKey().GetScriptHash()}})
		require.NoError(t, err)
		require.Equal(t, "HALT", res.State)
		require.Equal(t, 1, len(res.Stack))
		require.True(t, res.Stack[0].Value().(bool))
	})

	t.Run("positive, historic, by stateroot, with signer", func(t *testing.T) {
		h := chain.BlockHeight() - 1
		sr, err := chain.GetStateModule().GetStateRoot(h)
		require.NoError(t, err)
		res, err := c.InvokeContractVerifyWithState(sr.Root, contract, smartcontract.Params{}, []transaction.Signer{{Account: testchain.PrivateKeyByID(0).PublicKey().GetScriptHash()}})
		require.NoError(t, err)
		require.Equal(t, "HALT", res.State)
		require.Equal(t, 1, len(res.Stack))
		require.True(t, res.Stack[0].Value().(bool))
	})

	t.Run("bad, historic, by hash: contract not found", func(t *testing.T) {
		var h uint32 = 1
		_, err = c.InvokeContractVerifyAtHeight(h, contract, smartcontract.Params{}, []transaction.Signer{{Account: testchain.PrivateKeyByID(0).PublicKey().GetScriptHash()}})
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), core.ErrUnknownVerificationContract.Error())) // contract wasn't deployed at block #1 yet
	})

	t.Run("bad, historic, by block: contract not found", func(t *testing.T) {
		_, err = c.InvokeContractVerifyAtBlock(chain.GetHeaderHash(1), contract, smartcontract.Params{}, []transaction.Signer{{Account: testchain.PrivateKeyByID(0).PublicKey().GetScriptHash()}})
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), core.ErrUnknownVerificationContract.Error())) // contract wasn't deployed at block #1 yet
	})

	t.Run("bad, historic, by stateroot: contract not found", func(t *testing.T) {
		var h uint32 = 1
		sr, err := chain.GetStateModule().GetStateRoot(h)
		require.NoError(t, err)
		_, err = c.InvokeContractVerifyWithState(sr.Root, contract, smartcontract.Params{}, []transaction.Signer{{Account: testchain.PrivateKeyByID(0).PublicKey().GetScriptHash()}})
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), core.ErrUnknownVerificationContract.Error())) // contract wasn't deployed at block #1 yet
	})

	t.Run("positive, with signer and witness", func(t *testing.T) {
		res, err := c.InvokeContractVerify(contract, smartcontract.Params{}, []transaction.Signer{{Account: testchain.PrivateKeyByID(0).PublicKey().GetScriptHash()}}, transaction.Witness{InvocationScript: []byte{byte(opcode.PUSH1), byte(opcode.RET)}})
		require.NoError(t, err)
		require.Equal(t, "HALT", res.State)
		require.Equal(t, 1, len(res.Stack))
		require.True(t, res.Stack[0].Value().(bool))
	})

	t.Run("error, invalid witness number", func(t *testing.T) {
		_, err := c.InvokeContractVerify(contract, smartcontract.Params{}, []transaction.Signer{{Account: testchain.PrivateKeyByID(0).PublicKey().GetScriptHash()}}, transaction.Witness{InvocationScript: []byte{byte(opcode.PUSH1), byte(opcode.RET)}}, transaction.Witness{InvocationScript: []byte{byte(opcode.RET)}})
		require.Error(t, err)
	})

	t.Run("false", func(t *testing.T) {
		res, err := c.InvokeContractVerify(contract, smartcontract.Params{}, []transaction.Signer{{Account: util.Uint160{}}})
		require.NoError(t, err)
		require.Equal(t, "HALT", res.State)
		require.Equal(t, 1, len(res.Stack))
		require.False(t, res.Stack[0].Value().(bool))
	})
}

func TestClient_GetNativeContracts(t *testing.T) {
	chain, rpcSrv, httpSrv := initServerWithInMemoryChain(t)
	defer chain.Close()
	defer rpcSrv.Shutdown()

	c, err := client.New(context.Background(), httpSrv.URL, client.Options{})
	require.NoError(t, err)
	require.NoError(t, c.Init())

	cs, err := c.GetNativeContracts()
	require.NoError(t, err)
	require.Equal(t, chain.GetNatives(), cs)
}

func TestClient_NEP11_ND(t *testing.T) {
	chain, rpcSrv, httpSrv := initServerWithInMemoryChain(t)
	defer chain.Close()
	defer rpcSrv.Shutdown()

	c, err := client.New(context.Background(), httpSrv.URL, client.Options{})
	require.NoError(t, err)
	require.NoError(t, c.Init())

	h, err := util.Uint160DecodeStringLE(nnsContractHash)
	require.NoError(t, err)
	acc := testchain.PrivateKeyByID(0).GetScriptHash()

	t.Run("Decimals", func(t *testing.T) {
		d, err := c.NEP11Decimals(h)
		require.NoError(t, err)
		require.EqualValues(t, 0, d) // non-divisible
	})
	t.Run("TotalSupply", func(t *testing.T) {
		s, err := c.NEP11TotalSupply(h)
		require.NoError(t, err)
		require.EqualValues(t, 1, s) // the only `neo.com` of acc0
	})
	t.Run("Symbol", func(t *testing.T) {
		sym, err := c.NEP11Symbol(h)
		require.NoError(t, err)
		require.Equal(t, "NNS", sym)
	})
	t.Run("TokenInfo", func(t *testing.T) {
		tok, err := c.NEP11TokenInfo(h)
		require.NoError(t, err)
		require.Equal(t, &wallet.Token{
			Name:     "NameService",
			Hash:     h,
			Decimals: 0,
			Symbol:   "NNS",
			Standard: manifest.NEP11StandardName,
		}, tok)
	})
	t.Run("BalanceOf", func(t *testing.T) {
		b, err := c.NEP11BalanceOf(h, acc)
		require.NoError(t, err)
		require.EqualValues(t, 1, b)
	})
	t.Run("OwnerOf", func(t *testing.T) {
		b, err := c.NEP11NDOwnerOf(h, []byte("neo.com"))
		require.NoError(t, err)
		require.EqualValues(t, acc, b)
	})
	t.Run("Properties", func(t *testing.T) {
		p, err := c.NEP11Properties(h, []byte("neo.com"))
		require.NoError(t, err)
		blockRegisterDomain, err := chain.GetBlock(chain.GetHeaderHash(14)) // `neo.com` domain was registered in 14th block
		require.NoError(t, err)
		require.Equal(t, 1, len(blockRegisterDomain.Transactions))
		expected := stackitem.NewMap()
		expected.Add(stackitem.Make([]byte("name")), stackitem.Make([]byte("neo.com")))
		expected.Add(stackitem.Make([]byte("expiration")), stackitem.Make(blockRegisterDomain.Timestamp+365*24*3600*1000)) // expiration formula
		require.EqualValues(t, expected, p)
	})
	t.Run("Transfer", func(t *testing.T) {
		_, err := c.TransferNEP11(wallet.NewAccountFromPrivateKey(testchain.PrivateKeyByID(0)), testchain.PrivateKeyByID(1).GetScriptHash(), h, "neo.com", nil, 0, nil)
		require.NoError(t, err)
	})
}

func TestClient_NEP11_D(t *testing.T) {
	chain, rpcSrv, httpSrv := initServerWithInMemoryChain(t)
	defer chain.Close()
	defer rpcSrv.Shutdown()

	c, err := client.New(context.Background(), httpSrv.URL, client.Options{})
	require.NoError(t, err)
	require.NoError(t, c.Init())

	priv0 := testchain.PrivateKeyByID(0).GetScriptHash()
	priv1 := testchain.PrivateKeyByID(1).GetScriptHash()
	token1ID, err := hex.DecodeString(nfsoToken1ID)
	require.NoError(t, err)

	t.Run("Decimals", func(t *testing.T) {
		d, err := c.NEP11Decimals(nfsoHash)
		require.NoError(t, err)
		require.EqualValues(t, 2, d) // Divisible.
	})
	t.Run("TotalSupply", func(t *testing.T) {
		s, err := c.NEP11TotalSupply(nfsoHash)
		require.NoError(t, err)
		require.EqualValues(t, 1, s) // the only NFSO of acc0
	})
	t.Run("Symbol", func(t *testing.T) {
		sym, err := c.NEP11Symbol(nfsoHash)
		require.NoError(t, err)
		require.Equal(t, "NFSO", sym)
	})
	t.Run("TokenInfo", func(t *testing.T) {
		tok, err := c.NEP11TokenInfo(nfsoHash)
		require.NoError(t, err)
		require.Equal(t, &wallet.Token{
			Name:     "NeoFS Object NFT",
			Hash:     nfsoHash,
			Decimals: 2,
			Symbol:   "NFSO",
			Standard: manifest.NEP11StandardName,
		}, tok)
	})
	t.Run("BalanceOf", func(t *testing.T) {
		b, err := c.NEP11BalanceOf(nfsoHash, priv0)
		require.NoError(t, err)
		require.EqualValues(t, 80, b)
	})
	t.Run("OwnerOf", func(t *testing.T) {
		b, err := c.NEP11DOwnerOf(nfsoHash, token1ID)
		require.NoError(t, err)
		require.Equal(t, []util.Uint160{priv1, priv0}, b)
	})
	t.Run("Properties", func(t *testing.T) {
		p, err := c.NEP11Properties(nfsoHash, token1ID)
		require.NoError(t, err)
		expected := stackitem.NewMap()
		expected.Add(stackitem.Make([]byte("name")), stackitem.NewBuffer([]byte("NeoFS Object "+base64.StdEncoding.EncodeToString(token1ID))))
		expected.Add(stackitem.Make([]byte("containerID")), stackitem.Make([]byte(base64.StdEncoding.EncodeToString(nfsoToken1ContainerID.BytesBE()))))
		expected.Add(stackitem.Make([]byte("objectID")), stackitem.Make([]byte(base64.StdEncoding.EncodeToString(nfsoToken1ObjectID.BytesBE()))))
		require.EqualValues(t, expected, p)
	})
	t.Run("Transfer", func(t *testing.T) {
		_, err := c.TransferNEP11D(wallet.NewAccountFromPrivateKey(testchain.PrivateKeyByID(0)),
			testchain.PrivateKeyByID(1).GetScriptHash(),
			nfsoHash, 20, token1ID, nil, 0, nil)
		require.NoError(t, err)
	})
}

func TestClient_NNS(t *testing.T) {
	chain, rpcSrv, httpSrv := initServerWithInMemoryChain(t)
	defer chain.Close()
	defer rpcSrv.Shutdown()

	c, err := client.New(context.Background(), httpSrv.URL, client.Options{})
	require.NoError(t, err)
	require.NoError(t, c.Init())

	t.Run("NNSIsAvailable, false", func(t *testing.T) {
		b, err := c.NNSIsAvailable(nnsHash, "neo.com")
		require.NoError(t, err)
		require.Equal(t, false, b)
	})
	t.Run("NNSIsAvailable, true", func(t *testing.T) {
		b, err := c.NNSIsAvailable(nnsHash, "neogo.com")
		require.NoError(t, err)
		require.Equal(t, true, b)
	})
	t.Run("NNSResolve, good", func(t *testing.T) {
		b, err := c.NNSResolve(nnsHash, "neo.com", nns.A)
		require.NoError(t, err)
		require.Equal(t, "1.2.3.4", b)
	})
	t.Run("NNSResolve, bad", func(t *testing.T) {
		_, err := c.NNSResolve(nnsHash, "neogo.com", nns.A)
		require.Error(t, err)
	})
	t.Run("NNSResolve, forbidden", func(t *testing.T) {
		_, err := c.NNSResolve(nnsHash, "neogo.com", nns.CNAME)
		require.Error(t, err)
	})
	t.Run("NNSGetAllRecords, good", func(t *testing.T) {
		rss, err := c.NNSGetAllRecords(nnsHash, "neo.com")
		require.NoError(t, err)
		require.Equal(t, []nns.RecordState{
			{
				Name: "neo.com",
				Type: nns.A,
				Data: "1.2.3.4",
			},
		}, rss)
	})
	t.Run("NNSGetAllRecords, bad", func(t *testing.T) {
		_, err := c.NNSGetAllRecords(nnsHash, "neopython.com")
		require.Error(t, err)
	})
}

func TestClient_IteratorSessions(t *testing.T) {
	chain, rpcSrv, httpSrv := initServerWithInMemoryChain(t)
	defer chain.Close()
	defer rpcSrv.Shutdown()

	c, err := client.New(context.Background(), httpSrv.URL, client.Options{})
	require.NoError(t, err)
	require.NoError(t, c.Init())

	storageHash, err := util.Uint160DecodeStringLE(storageContractHash)
	require.NoError(t, err)

	// storageItemsCount is the amount of storage items stored in Storage contract, it's hard-coded in the contract code.
	const storageItemsCount = 255
	expected := make([][]byte, storageItemsCount)
	for i := 0; i < storageItemsCount; i++ {
		expected[i] = stackitem.NewBigInteger(big.NewInt(int64(i))).Bytes()
	}
	sort.Slice(expected, func(i, j int) bool {
		if len(expected[i]) != len(expected[j]) {
			return len(expected[i]) < len(expected[j])
		}
		return bytes.Compare(expected[i], expected[j]) < 0
	})

	prepareSession := func(t *testing.T) (uuid.UUID, uuid.UUID) {
		res, err := c.InvokeFunction(storageHash, "iterateOverValues", []smartcontract.Parameter{}, nil)
		require.NoError(t, err)
		require.NotEmpty(t, res.Session)
		require.Equal(t, 1, len(res.Stack))
		require.Equal(t, stackitem.InteropT, res.Stack[0].Type())
		iterator, ok := res.Stack[0].Value().(result.Iterator)
		require.True(t, ok)
		require.NotEmpty(t, iterator.ID)
		return res.Session, iterator.ID
	}
	t.Run("traverse with max constraint", func(t *testing.T) {
		sID, iID := prepareSession(t)
		check := func(t *testing.T, start, end int) {
			max := end - start
			set, err := c.TraverseIterator(sID, iID, max)
			require.NoError(t, err)
			require.Equal(t, max, len(set))
			for i := 0; i < max; i++ {
				// According to the Storage contract code.
				require.Equal(t, expected[start+i], set[i].Value().([]byte), start+i)
			}
		}
		check(t, 0, 30)
		check(t, 30, 48)
		check(t, 48, 48)
		check(t, 48, 48+config.DefaultMaxIteratorResultItems)
		check(t, 48+config.DefaultMaxIteratorResultItems, 48+2*config.DefaultMaxIteratorResultItems-1)
		check(t, 48+2*config.DefaultMaxIteratorResultItems-1, 255)

		// Iterator ends on 255-th element, so no more elements should be returned.
		set, err := c.TraverseIterator(sID, iID, config.DefaultMaxIteratorResultItems)
		require.NoError(t, err)
		require.Equal(t, 0, len(set))
	})

	t.Run("traverse, request more than exists", func(t *testing.T) {
		sID, iID := prepareSession(t)
		for i := 0; i < storageItemsCount/config.DefaultMaxIteratorResultItems; i++ {
			set, err := c.TraverseIterator(sID, iID, config.DefaultMaxIteratorResultItems)
			require.NoError(t, err)
			require.Equal(t, config.DefaultMaxIteratorResultItems, len(set))
		}

		// Request more items than left untraversed.
		set, err := c.TraverseIterator(sID, iID, config.DefaultMaxIteratorResultItems)
		require.NoError(t, err)
		require.Equal(t, storageItemsCount%config.DefaultMaxIteratorResultItems, len(set))
	})

	t.Run("traverse, no max constraint", func(t *testing.T) {
		sID, iID := prepareSession(t)

		set, err := c.TraverseIterator(sID, iID, -1)
		require.NoError(t, err)
		require.Equal(t, storageItemsCount, len(set))

		// No more items should be left.
		set, err = c.TraverseIterator(sID, iID, -1)
		require.NoError(t, err)
		require.Equal(t, 0, len(set))
	})

	t.Run("traverse, concurrent access", func(t *testing.T) {
		sID, iID := prepareSession(t)
		wg := sync.WaitGroup{}
		wg.Add(storageItemsCount)
		check := func(t *testing.T) {
			set, err := c.TraverseIterator(sID, iID, 1)
			require.NoError(t, err)
			require.Equal(t, 1, len(set))
			wg.Done()
		}
		for i := 0; i < storageItemsCount; i++ {
			go check(t)
		}
		wg.Wait()
	})

	t.Run("terminate session", func(t *testing.T) {
		t.Run("manually", func(t *testing.T) {
			sID, iID := prepareSession(t)

			// Check session is created.
			set, err := c.TraverseIterator(sID, iID, 1)
			require.NoError(t, err)
			require.Equal(t, 1, len(set))

			ok, err := c.TerminateSession(sID)
			require.NoError(t, err)
			require.True(t, ok)

			ok, err = c.TerminateSession(sID)
			require.NoError(t, err)
			require.False(t, ok) // session has already been terminated.
		})
		t.Run("automatically", func(t *testing.T) {
			sID, iID := prepareSession(t)

			// Check session is created.
			set, err := c.TraverseIterator(sID, iID, 1)
			require.NoError(t, err)
			require.Equal(t, 1, len(set))

			require.Eventually(t, func() bool {
				rpcSrv.sessionsLock.Lock()
				defer rpcSrv.sessionsLock.Unlock()

				_, ok := rpcSrv.sessions[sID.String()]
				return !ok
			}, time.Duration(rpcSrv.config.SessionExpirationTime)*time.Second*3,
				// Sessions list is updated once per SessionExpirationTime, thus, no need to ask for update more frequently than
				// sessions cleaning occurs.
				time.Duration(rpcSrv.config.SessionExpirationTime)*time.Second/4)

			ok, err := c.TerminateSession(sID)
			require.NoError(t, err)
			require.False(t, ok) // session has already been terminated.
		})
	})
}

func TestClient_GetNotaryServiceFeePerKey(t *testing.T) {
	chain, rpcSrv, httpSrv := initServerWithInMemoryChain(t)
	defer chain.Close()
	defer rpcSrv.Shutdown()

	c, err := client.New(context.Background(), httpSrv.URL, client.Options{})
	require.NoError(t, err)
	require.NoError(t, c.Init())

	var defaultNotaryServiceFeePerKey int64 = 1000_0000
	actual, err := c.GetNotaryServiceFeePerKey()
	require.NoError(t, err)
	require.Equal(t, defaultNotaryServiceFeePerKey, actual)
}

func TestClient_GetOraclePrice(t *testing.T) {
	chain, rpcSrv, httpSrv := initServerWithInMemoryChain(t)
	defer chain.Close()
	defer rpcSrv.Shutdown()

	c, err := client.New(context.Background(), httpSrv.URL, client.Options{})
	require.NoError(t, err)
	require.NoError(t, c.Init())

	var defaultOracleRequestPrice int64 = 5000_0000
	actual, err := c.GetOraclePrice()
	require.NoError(t, err)
	require.Equal(t, defaultOracleRequestPrice, actual)
}

func TestClient_InvokeAndPackIteratorResults(t *testing.T) {
	chain, rpcSrv, httpSrv := initServerWithInMemoryChain(t)
	defer chain.Close()
	defer rpcSrv.Shutdown()

	c, err := client.New(context.Background(), httpSrv.URL, client.Options{})
	require.NoError(t, err)
	require.NoError(t, c.Init())

	// storageItemsCount is the amount of storage items stored in Storage contract, it's hard-coded in the contract code.
	const storageItemsCount = 255
	expected := make([][]byte, storageItemsCount)
	for i := 0; i < storageItemsCount; i++ {
		expected[i] = stackitem.NewBigInteger(big.NewInt(int64(i))).Bytes()
	}
	sort.Slice(expected, func(i, j int) bool {
		if len(expected[i]) != len(expected[j]) {
			return len(expected[i]) < len(expected[j])
		}
		return bytes.Compare(expected[i], expected[j]) < 0
	})

	storageHash, err := util.Uint160DecodeStringLE(storageContractHash)
	require.NoError(t, err)
	res, err := c.InvokeAndPackIteratorResults(storageHash, "iterateOverValues", []smartcontract.Parameter{}, nil)
	require.NoError(t, err)
	require.Equal(t, vm.HaltState.String(), res.State)
	require.Equal(t, 1, len(res.Stack))
	require.Equal(t, stackitem.ArrayT, res.Stack[0].Type())
	arr, ok := res.Stack[0].Value().([]stackitem.Item)
	require.True(t, ok)
	require.Equal(t, storageItemsCount, len(arr))

	for i := range arr {
		require.Equal(t, stackitem.ByteArrayT, arr[i].Type())
		require.Equal(t, expected[i], arr[i].Value().([]byte))
	}
}
