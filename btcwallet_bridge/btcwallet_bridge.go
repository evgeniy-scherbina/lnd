package btcwallet_bridge

import (
	"github.com/roasbeef/btcd/btcjson"
	"encoding/json"
	"github.com/lightningnetwork/lnd/lnwallet"
	"encoding/hex"
	"github.com/roasbeef/btcd/txscript"
	"github.com/roasbeef/btcutil"
	"math"
	"github.com/roasbeef/btcd/chaincfg/chainhash"
	"github.com/roasbeef/btcd/wire"
	"github.com/roasbeef/btcwallet/waddrmgr"
	"fmt"
)

type BtcWalletRPCClient struct {}

/*
func (rpcClient *BtcWalletRPCClient) GetNewAddress() ([]byte, error) {
	method := "getnewaddress"
	params := make([]interface{}, 0)
	return rpcClient.execute(method, params)
}
*/

func (rpcClient *BtcWalletRPCClient) ListUnspent(minConf, maxConf int32, addresses map[string]struct{}) ([]*btcjson.ListUnspentResult, error) {
	method := "listunspent"
	params := []interface{}{minConf, maxConf, addresses}
	rawBytes, err := rpcClient.execute(method, params...)
	if err != nil {
		return nil, err
	}

	result := make([]*btcjson.ListUnspentResult, 0)
	if err := json.Unmarshal(rawBytes, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (rpcClient *BtcWalletRPCClient) execute(method string, params ...interface{}) ([]byte, error) {
	cmd, err := btcjson.NewCmd(method, params...)
	if err != nil {
		return nil, err
	}

	marshalledJSON, err := btcjson.MarshalCmd(1, cmd)
	if err != nil {
		return nil, err
	}

	cfg := defaultConfig()
	result, err := sendPostRequest(marshalledJSON, cfg)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type BtcWalletBridge struct {
	wallet *BtcWalletRPCClient
}

//FetchInputInfo(prevOut *wire.OutPoint) (*wire.TxOut, error)

// ConfirmedBalance returns the sum of all the wallet's unspent outputs that
// have at least confs confirmations. If confs is set to zero, then all unspent
// outputs, including those currently in the mempool will be included in the
// final sum.
//
// This is a part of the WalletController interface.
func (b *BtcWalletBridge) ConfirmedBalance(confs int32) (btcutil.Amount, error) {
	var balance btcutil.Amount

	witnessOutputs, err := b.ListUnspentWitness(confs)
	if err != nil {
		return 0, err
	}

	for _, witnessOutput := range witnessOutputs {
		balance += witnessOutput.Value
	}

	return balance, nil
}

// NewAddress returns the next external or internal address for the wallet
// dictated by the value of the `change` parameter. If change is true, then an
// internal address will be returned, otherwise an external address should be
// returned.
//
// This is a part of the WalletController interface.
func (b *BtcWalletBridge) NewAddress(t lnwallet.AddressType, change bool) (btcutil.Address, error) {
	var keyScope waddrmgr.KeyScope

	switch t {
	case lnwallet.WitnessPubKey:
		keyScope = waddrmgr.KeyScopeBIP0084
	case lnwallet.NestedWitnessPubKey:
		keyScope = waddrmgr.KeyScopeBIP0049Plus
	default:
		return nil, fmt.Errorf("unknown address type")
	}

	// TODO(evg): don't ignore change flag
	/*
	if change {
		return b.wallet.NewChangeAddress(defaultAccount, keyScope)
	}
	*/

	return b.wallet.NewAddress(defaultAccount, keyScope)
}

//GetPrivKey(a btcutil.Address) (*btcec.PrivateKey, error)
//SendOutputs(outputs []*wire.TxOut,
//feeRate SatPerVByte) (*chainhash.Hash, error)

// ListUnspentWitness returns a slice of all the unspent outputs the wallet
// controls which pay to witness programs either directly or indirectly.
//
// This is a part of the WalletController interface.
func (b *BtcWalletBridge) ListUnspentWitness(minConfs int32) ([]*lnwallet.Utxo, error) {
	// First, grab all the unfiltered currently unspent outputs.
	maxConfs := int32(math.MaxInt32)
	unspentOutputs, err := b.wallet.ListUnspent(minConfs, maxConfs, nil)
	if err != nil {
		return nil, err
	}

	// Next, we'll run through all the regular outputs, only saving those
	// which are p2wkh outputs or a p2wsh output nested within a p2sh output.
	witnessOutputs := make([]*lnwallet.Utxo, 0, len(unspentOutputs))
	for _, output := range unspentOutputs {
		pkScript, err := hex.DecodeString(output.ScriptPubKey)
		if err != nil {
			return nil, err
		}

		var addressType lnwallet.AddressType
		if txscript.IsPayToWitnessPubKeyHash(pkScript) {
			addressType = lnwallet.WitnessPubKey
		} else if txscript.IsPayToScriptHash(pkScript) {
			// TODO(roasbeef): This assumes all p2sh outputs returned by the
			// wallet are nested p2pkh. We can't check the redeem script because
			// the btcwallet service does not include it.
			addressType = lnwallet.NestedWitnessPubKey
		}

		if addressType == lnwallet.WitnessPubKey ||
			addressType == lnwallet.NestedWitnessPubKey {

			txid, err := chainhash.NewHashFromStr(output.TxID)
			if err != nil {
				return nil, err
			}

			// We'll ensure we properly convert the amount given in
			// BTC to satoshis.
			amt, err := btcutil.NewAmount(output.Amount)
			if err != nil {
				return nil, err
			}

			utxo := &lnwallet.Utxo{
				AddressType: addressType,
				Value:       amt,
				PkScript:    pkScript,
				OutPoint: wire.OutPoint{
					Hash:  *txid,
					Index: output.Vout,
				},
			}
			witnessOutputs = append(witnessOutputs, utxo)
		}

	}

	return witnessOutputs, nil
}

//ListTransactionDetails() ([]*TransactionDetail, error)
//LockOutpoint(o wire.OutPoint)
//UnlockOutpoint(o wire.OutPoint)
//PublishTransaction(tx *wire.MsgTx) error
//// TODO(roasbeef): make distinct interface?
//SubscribeTransactions() (TransactionSubscription, error)
//IsSynced() (bool, int64, error)
//Start() error
//Stop() error
//BackEnd() string

/*
func main() {
	method := "getnewaddress"
	params := make([]interface{}, 0)
	cmd, err := btcjson.NewCmd(method, params...)
	if err != nil {
		fmt.Println("internal error:", err)
		return
	}

	marshalledJSON, err := btcjson.MarshalCmd(1, cmd)
	if err != nil {
		fmt.Println("internal error:", err)
		return
	}

	cfg := defaultConfig()
	result, err := sendPostRequest(marshalledJSON, cfg)
	if err != nil {
		fmt.Println("internal error:", err)
	}
	fmt.Println("result:", string(result))
}
*/

