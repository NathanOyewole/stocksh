package solana

import (
	"context"
	"fmt"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

// SignAndSend takes a base64 versioned tx from Jupiter, signs it, and sends it.
func SignAndSend(client *rpc.Client, wallet *Wallet, b64Tx string) (string, error) {
	tx, err := solana.TransactionFromBase64(b64Tx)
	if err != nil {
		return "", fmt.Errorf("parse tx: %w", err)
	}

	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(wallet.PubKey) {
			return wallet.Keypair
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("sign: %w", err)
	}

	sig, err := client.SendTransactionWithOpts(
		context.Background(),
		tx,
		rpc.TransactionOpts{
			SkipPreflight:       false,
			PreflightCommitment: rpc.CommitmentConfirmed,
		},
	)
	if err != nil {
		return "", fmt.Errorf("send: %w", err)
	}

	return sig.String(), nil
}
