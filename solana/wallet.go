package solana

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

type Wallet struct {
	Keypair *solana.PrivateKey
	PubKey  solana.PublicKey
}

func LoadWallet() (*Wallet, error) {
	if pk := os.Getenv("SOLANA_PRIVATE_KEY"); pk != "" {
		keypair, err := solana.PrivateKeyFromBase58(pk)
		if err != nil {
			return nil, fmt.Errorf("invalid SOLANA_PRIVATE_KEY: %w", err)
		}
		return &Wallet{Keypair: &keypair, PubKey: keypair.PublicKey()}, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(home, ".config", "solana", "id.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("no wallet found. Set SOLANA_PRIVATE_KEY or place keypair at %s: %w", path, err)
	}
	keypair, err := solana.PrivateKeyFromSolanaKeygenFileBytes(data)
	if err != nil {
		return nil, fmt.Errorf("parse keypair: %w", err)
	}
	return &Wallet{Keypair: &keypair, PubKey: keypair.PublicKey()}, nil
}

const (
	DevnetRPC  = "https://api.devnet.solana.com"
	MainnetRPC = "https://api.mainnet-beta.solana.com"
)

func NewRPC(endpoint string) *rpc.Client {
	if endpoint == "" {
		endpoint = os.Getenv("SOLANA_RPC")
	}
	if endpoint == "" {
		endpoint = DevnetRPC
	}
	return rpc.New(endpoint)
}

func IsDevnet(endpoint string) bool {
	if endpoint == "" {
		endpoint = os.Getenv("SOLANA_RPC")
	}
	if endpoint == "" {
		return true
	}
	return strings.Contains(strings.ToLower(endpoint), "devnet")
}

func RequestAirdrop(client *rpc.Client, pubkey solana.PublicKey, lamports uint64) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	sig, err := client.RequestAirdrop(ctx, pubkey, lamports, rpc.CommitmentConfirmed)
	if err != nil {
		return "", fmt.Errorf("airdrop failed: %w", err)
	}
	return sig.String(), nil
}

func GetBalance(client *rpc.Client, pubkey solana.PublicKey) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	bal, err := client.GetBalance(ctx, pubkey, rpc.CommitmentConfirmed)
	if err != nil {
		return 0, err
	}
	return float64(bal.Value) / 1e9, nil
}

type TokenBalance struct {
	Mint   string
	Symbol string
	Amount float64
}

func GetTokenBalances(client *rpc.Client, owner solana.PublicKey, mintToSymbol map[string]string) ([]TokenBalance, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	var out []TokenBalance
	for mintStr, symbol := range mintToSymbol {
		mint, err := solana.PublicKeyFromBase58(mintStr)
		if err != nil {
			continue
		}
		for _, programID := range []solana.PublicKey{solana.TokenProgramID, solana.Token2022ProgramID} {
			ata, _, err := solana.FindAssociatedTokenAddress(owner, mint)
			if err != nil {
				continue
			}
			if programID.Equals(solana.Token2022ProgramID) {
				ata, _, err = solana.FindAssociatedTokenAddressWithProgram(owner, mint, programID)
				if err != nil {
					continue
				}
			}
			balRes, err := client.GetTokenAccountBalance(ctx, ata, rpc.CommitmentConfirmed)
			if err != nil {
				continue
			}
			if balRes == nil || balRes.Value == nil {
				continue
			}
			ui := 0.0
			if balRes.Value.UiAmount != nil {
				ui = *balRes.Value.UiAmount
			} else if balRes.Value.UiAmountString != "" {
				fmt.Sscanf(balRes.Value.UiAmountString, "%f", &ui)
			}
			if ui > 0 {
				out = append(out, TokenBalance{Mint: mintStr, Symbol: symbol, Amount: ui})
				break
			}
		}
	}
	return out, nil
}
