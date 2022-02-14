// Copyright 2018 Shift Devices AG
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package addresses

import (
	"crypto/sha256"
	"fmt"
	"math/big"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/digitalbitbox/bitbox-wallet-app/backend/coins/btc/blockchain"
	"github.com/digitalbitbox/bitbox-wallet-app/backend/coins/btc/types"
	ourbtcutil "github.com/digitalbitbox/bitbox-wallet-app/backend/coins/btc/util"
	"github.com/digitalbitbox/bitbox-wallet-app/backend/signing"
	"github.com/sirupsen/logrus"
)

// AccountAddress models an address that belongs to an account of the user.
// It contains all the information needed to receive and spend funds.
type AccountAddress struct {
	btcutil.Address

	// Configuration contains the absolute keypath and the extended public keys of the address.
	Configuration *signing.Configuration

	// HistoryStatus is used to determine if the address status changed, and to determine if the
	// address has been used before or not. The status corresponds to
	// https://github.com/kyuupichan/electrumx/blob/46f245891cb62845f9eec0f9549526a7e569eb03/docs/protocol-basics.rst#status.
	HistoryStatus string

	// redeemScript stores the redeem script of a BIP16 P2SH output or nil if address type is P2PKH.
	redeemScript []byte

	log *logrus.Entry
}

// NewAccountAddress creates a new account address.
func NewAccountAddress(
	accountConfiguration *signing.Configuration,
	keyPath signing.RelativeKeypath,
	net *chaincfg.Params,
	log *logrus.Entry,
) *AccountAddress {

	var address btcutil.Address
	var redeemScript []byte
	configuration, err := accountConfiguration.Derive(keyPath)
	if err != nil {
		log.WithError(err).Panic("Failed to derive the configuration.")
	}
	log = log.WithFields(logrus.Fields{
		"key-path":      configuration.AbsoluteKeypath().Encode(),
		"configuration": configuration.String(),
	})
	log.Debug("Creating new account address")

	publicKeyHash := btcutil.Hash160(configuration.PublicKey().SerializeCompressed())
	switch configuration.ScriptType() {
	case signing.ScriptTypeP2PKH:
		address, err = btcutil.NewAddressPubKeyHash(publicKeyHash, net)
		if err != nil {
			log.WithError(err).Panic("Failed to get P2PKH addr. from public key hash.")
		}
	case signing.ScriptTypeP2WPKHP2SH:
		var segwitAddress *btcutil.AddressWitnessPubKeyHash
		segwitAddress, err = btcutil.NewAddressWitnessPubKeyHash(publicKeyHash, net)
		if err != nil {
			log.WithError(err).Panic("Failed to get p2wpkh-p2sh addr. from publ. key hash.")
		}
		redeemScript, err = txscript.PayToAddrScript(segwitAddress)
		if err != nil {
			log.WithError(err).Panic("Failed to get redeem script for segwit address.")
		}
		address, err = btcutil.NewAddressScriptHash(redeemScript, net)
		if err != nil {
			log.WithError(err).Panic("Failed to get a P2SH address for segwit.")
		}
	case signing.ScriptTypeP2WPKH:
		address, err = btcutil.NewAddressWitnessPubKeyHash(publicKeyHash, net)
		if err != nil {
			log.WithError(err).Panic("Failed to get p2wpkh addr. from publ. key hash.")
		}
	case signing.ScriptTypeP2TR:
		// A taproot pubkey is an x-only pubkey, i.e. only the X-coordinate. For every X-coordinate,
		// there are two possible curve points. BIP340 defines the tie-breaker to be the oddness of
		// the Y-coordinate, choosing the one that is even.
		//
		// If a point P is odd, the negation of the point (-P) is even.
		//
		// See also:
		// https://github.com/bitcoin/bips/blob/a3a397c82384220fc871852c809f73898a4d547c/bip-0340.mediawiki
		// https://github.com/bitcoin/bips/blob/a3a397c82384220fc871852c809f73898a4d547c/bip-0086.mediawiki
		pubKey := configuration.PublicKey()
		isOdd := func(pubKey *btcec.PublicKey) bool {
			return pubKey.SerializeCompressed()[0] == 0x03
		}
		negate := func(pubKey *btcec.PublicKey) *btcec.PublicKey {
			// See also: https://bitcoin.stackexchange.com/a/107957
			return &btcec.PublicKey{
				Curve: pubKey.Curve,
				X:     pubKey.X,
				Y:     new(big.Int).Sub(pubKey.Curve.Params().P, pubKey.Y),
			}
			// Alternative code with the same result: -P = (n-1)*P.
			// x, y := pubKey.Curve.ScalarMult(
			// 	pubKey.X,
			// 	pubKey.Y,
			// 	new(big.Int).Sub(pubKey.Curve.Params().N, big.NewInt(1)).Bytes(),
			// )
			// return &btcec.PublicKey{Curve: pubKey.Curve, X: x, Y: y}
		}
		taggedHash := func(tag string, msg []byte) []byte {
			tagHash := sha256.Sum256([]byte(tag))
			h := sha256.New()
			h.Write(tagHash[:]) //nolint:errcheck
			h.Write(tagHash[:]) //nolint:errcheck
			h.Write(msg)        //nolint:errcheck
			return h.Sum(nil)
		}
		if isOdd(pubKey) {
			pubKey = negate(pubKey)
		}
		_, tweak := btcec.PrivKeyFromBytes(pubKey.Curve, taggedHash("TapTweak", pubKey.SerializeCompressed()[1:]))
		x, y := pubKey.Curve.Add(pubKey.X, pubKey.Y, tweak.X, tweak.Y)
		xOnlyPubKey := (&btcec.PublicKey{Curve: pubKey.Curve, X: x, Y: y}).SerializeCompressed()[1:]
		address, err = btcutil.NewAddressTaproot(xOnlyPubKey, net)
		if err != nil {
			log.WithError(err).Panic("Failed to get p2tr addr")
		}
	default:
		log.Panic(fmt.Sprintf("Unrecognized script type: %s", configuration.ScriptType()))
	}

	return &AccountAddress{
		Address:       address,
		Configuration: configuration,
		HistoryStatus: "",
		redeemScript:  redeemScript,
		log:           log,
	}
}

// ID implements accounts.Address.
func (address *AccountAddress) ID() string {
	return string(address.PubkeyScriptHashHex())
}

// EncodeForHumans implements accounts.Address.
func (address *AccountAddress) EncodeForHumans() string {
	return address.EncodeAddress()
}

// AbsoluteKeypath implements coin.AbsoluteKeypath.
func (address *AccountAddress) AbsoluteKeypath() signing.AbsoluteKeypath {
	return address.Configuration.AbsoluteKeypath()
}

func (address *AccountAddress) isUsed() bool {
	return address.HistoryStatus != ""
}

// PubkeyScript returns the pubkey script of this address. Use this in a tx output to receive funds.
func (address *AccountAddress) PubkeyScript() []byte {
	script, err := ourbtcutil.PkScriptFromAddress(address.Address)
	if err != nil {
		address.log.WithError(err).Panic("Failed to get the pubkey script for an address.")
	}
	return script
}

// PubkeyScriptHashHex returns the hash of the pubkey script in hex format.
// It is used to subscribe to notifications at the ElectrumX server.
func (address *AccountAddress) PubkeyScriptHashHex() blockchain.ScriptHashHex {
	return blockchain.NewScriptHashHex(address.PubkeyScript())
}

// ScriptForHashToSign returns whether this address is a segwit output and the script used when
// calculating the hash to be signed in a transaction. This info is needed when trying to spend
// from this address.
func (address *AccountAddress) ScriptForHashToSign() (bool, []byte) {
	switch address.Configuration.ScriptType() {
	case signing.ScriptTypeP2PKH:
		return false, address.PubkeyScript()
	case signing.ScriptTypeP2WPKHP2SH:
		return true, address.redeemScript
	case signing.ScriptTypeP2WPKH:
		return true, address.PubkeyScript()
	default:
		address.log.Panic("Unrecognized address type.")
	}
	panic("The end of the function cannot be reached.")
}

// SignatureScript returns the signature script (and witness) needed to spend from this address.
// The signatures have to be provided in the order of the configuration (and some can be nil).
func (address *AccountAddress) SignatureScript(
	signature types.Signature,
) ([]byte, wire.TxWitness) {
	publicKey := address.Configuration.PublicKey()
	switch address.Configuration.ScriptType() {
	case signing.ScriptTypeP2PKH:
		signatureScript, err := txscript.NewScriptBuilder().
			AddData(append(signature.SerializeDER(), byte(txscript.SigHashAll))).
			AddData(publicKey.SerializeCompressed()).
			Script()
		if err != nil {
			address.log.WithError(err).Panic("Failed to build signature script for P2PKH.")
		}
		return signatureScript, nil
	case signing.ScriptTypeP2WPKHP2SH:
		signatureScript, err := txscript.NewScriptBuilder().
			AddData(address.redeemScript).
			Script()
		if err != nil {
			address.log.WithError(err).Panic("Failed to build segwit signature script.")
		}
		txWitness := wire.TxWitness{
			append(signature.SerializeDER(), byte(txscript.SigHashAll)),
			publicKey.SerializeCompressed(),
		}
		return signatureScript, txWitness
	case signing.ScriptTypeP2WPKH:
		txWitness := wire.TxWitness{
			append(signature.SerializeDER(), byte(txscript.SigHashAll)),
			publicKey.SerializeCompressed(),
		}
		return []byte{}, txWitness
	case signing.ScriptTypeP2TR:
		// We assume SIGHASH_DEFAULT, which defaults to SIGHASH_ALL without needing to explicitly
		// append it to the signature. See:
		// https://github.com/bitcoin/bips/blob/97e02b2223b21753acefa813a4e59dbb6e849e77/bip-0341.mediawiki#taproot-key-path-spending-signature-validation
		txWitness := wire.TxWitness{
			signature.SerializeCompact(),
		}
		return []byte{}, txWitness
	default:
		address.log.Panic("Unrecognized address type.")
	}
	panic("The end of the function cannot be reached.")
}
