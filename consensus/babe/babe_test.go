// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package babe

import (
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/common"
	tx "github.com/ChainSafe/gossamer/common/transaction"
	"github.com/ChainSafe/gossamer/core/types"
	"github.com/ChainSafe/gossamer/polkadb"
	"github.com/ChainSafe/gossamer/runtime"
	"github.com/ChainSafe/gossamer/trie"
)

const POLKADOT_RUNTIME_FP string = "../../substrate_test_runtime.compact.wasm"
const POLKADOT_RUNTIME_URL string = "https://github.com/noot/substrate/blob/add-blob/core/test-runtime/wasm/wasm32-unknown-unknown/release/wbuild/substrate-test-runtime/substrate_test_runtime.compact.wasm?raw=true"

// getRuntimeBlob checks if the polkadot runtime wasm file exists and if not, it fetches it from github
func getRuntimeBlob() (n int64, err error) {
	if Exists(POLKADOT_RUNTIME_FP) {
		return 0, nil
	}

	out, err := os.Create(POLKADOT_RUNTIME_FP)
	if err != nil {
		return 0, err
	}
	defer out.Close()

	resp, err := http.Get(POLKADOT_RUNTIME_URL)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	n, err = io.Copy(out, resp.Body)
	return n, err
}

// Exists reports whether the named file or directory exists.
func Exists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// func newTrie() (*trie.Trie, error) {
// 	hasher, err := trie.NewHasher()
// 	if err != nil {
// 		return nil, err
// 	}

// 	stateDB, err := polkadb.NewBadgerDB("./test_data/state")
// 	if err != nil {
// 		return nil, err
// 	}

// 	trie := &trie.Trie{
// 		Database: &trie.StateDB{
// 			Db:     stateDB,
// 			Hasher: hasher,
// 		},
// 		NodeRoot: nil,
// 	}

// 	trie.Database.Batch = trie.Database.Db.NewBatch()

// 	return trie, nil
// }

func newRuntime(t *testing.T) *runtime.Runtime {
	fmt.Println("CREATING NEW RUNTIMEBLOB")
	_, err := getRuntimeBlob()
	if err != nil {
		t.Fatalf("Fail: could not get polkadot runtime")
	}

	fp, err := filepath.Abs(POLKADOT_RUNTIME_FP)
	if err != nil {
		t.Fatal("could not create filepath")
	}

	// DB, err := polkadb.NewDatabaseService()
	// if err != nil {
	// 	t.Fatal(err)
	// }

	fmt.Println("CREATING NEW STATEDB")
	db := &trie.StateDB{
		Db: polkadb.NewMemDatabase(),
	}
	tt := trie.NewEmptyTrie(db)

	fmt.Println("TRIE: ", tt)

	// tt, err := trie.NewEmptyTrie()
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// tt := trie.NewEmptyTrie(DB.StateDB)

	r, err := runtime.NewRuntime(fp, tt)
	if err != nil {
		t.Fatal(err)
	} else if r == nil {
		t.Fatal("did not create new VM")
	}

	return r
}

func TestCalculateThreshold(t *testing.T) {
	// C = 1
	var C1 uint64 = 1
	var C2 uint64 = 1
	var authorityIndex uint64 = 0
	authorityWeights := []uint64{1, 1, 1}

	expected := new(big.Int).Lsh(big.NewInt(1), 128)

	threshold, err := calculateThreshold(C1, C2, authorityIndex, authorityWeights)
	if err != nil {
		t.Fatal(err)
	}

	if threshold.Cmp(expected) != 0 {
		t.Fatalf("Fail: got %d expected %d", threshold, expected)
	}

	// C = 1/2
	C2 = 2

	theta := float64(1) / float64(3)
	c := float64(C1) / float64(C2)
	pp := 1 - c
	pp_exp := math.Pow(pp, theta)
	p := 1 - pp_exp
	p_rat := new(big.Rat).SetFloat64(p)
	q := new(big.Int).Lsh(big.NewInt(1), 128)
	expected = q.Mul(q, p_rat.Num()).Div(q, p_rat.Denom())

	threshold, err = calculateThreshold(C1, C2, authorityIndex, authorityWeights)
	if err != nil {
		t.Fatal(err)
	}

	if threshold.Cmp(expected) != 0 {
		t.Fatalf("Fail: got %d expected %d", threshold, expected)
	}
}

func TestCalculateThreshold_AuthorityWeights(t *testing.T) {
	var C1 uint64 = 5
	var C2 uint64 = 17
	var authorityIndex uint64 = 3
	authorityWeights := []uint64{3, 1, 4, 6, 10}

	theta := float64(6) / float64(24)
	c := float64(C1) / float64(C2)
	pp := 1 - c
	pp_exp := math.Pow(pp, theta)
	p := 1 - pp_exp
	p_rat := new(big.Rat).SetFloat64(p)
	q := new(big.Int).Lsh(big.NewInt(1), 128)
	expected := q.Mul(q, p_rat.Num()).Div(q, p_rat.Denom())

	threshold, err := calculateThreshold(C1, C2, authorityIndex, authorityWeights)
	if err != nil {
		t.Fatal(err)
	}

	if threshold.Cmp(expected) != 0 {
		t.Fatalf("Fail: got %d expected %d", threshold, expected)
	}
}

func TestRunLottery(t *testing.T) {
	rt := newRuntime(t)
	babesession := NewSession([32]byte{}, [64]byte{}, rt)
	babesession.authorityIndex = 0
	babesession.authorityWeights = []uint64{1, 1, 1}
	conf := &BabeConfiguration{
		SlotDuration:       1000,
		EpochLength:        6,
		C1:                 3,
		C2:                 10,
		GenesisAuthorities: []AuthorityData{},
		Randomness:         0,
		SecondarySlots:     false,
	}
	babesession.config = conf

	_, err := babesession.runLottery(0)
	if err != nil {
		t.Fatal(err)
	}
}

func TestCalculateThreshold_Failing(t *testing.T) {
	var C1 uint64 = 5
	var C2 uint64 = 4
	var authorityIndex uint64 = 3
	authorityWeights := []uint64{3, 1, 4, 6, 10}

	_, err := calculateThreshold(C1, C2, authorityIndex, authorityWeights)
	if err == nil {
		t.Fatal("Fail: did not err for c>1")
	}
}

func TestConfigurationFromRuntime(t *testing.T) {
	rt := newRuntime(t)
	babesession := NewSession([32]byte{}, [64]byte{}, rt)
	res, err := babesession.configurationFromRuntime()
	if err != nil {
		t.Fatal(err)
	}

	// see: https://github.com/paritytech/substrate/blob/7b1d822446982013fa5b7ad5caff35ca84f8b7d0/core/test-runtime/src/lib.rs#L621
	expected := &BabeConfiguration{
		SlotDuration:       1000,
		EpochLength:        6,
		C1:                 3,
		C2:                 10,
		GenesisAuthorities: []AuthorityData{},
		Randomness:         0,
		SecondarySlots:     false,
	}

	if res == expected {
		t.Errorf("Fail: got %v expected %v\n", res, expected)
	}
}

func TestBuildBlock(t *testing.T) {
	rt := newRuntime(t)
	babesession := NewSession([32]byte{}, [64]byte{}, rt)
	_, err := babesession.configurationFromRuntime()
	if err != nil {
		t.Fatal(err)
	}

	// Create 2 transactions & push to TxQueue in babesession
	e1 := &types.Extrinsic{0x01, 0x02, 0x03}
	v1 := &tx.Validity{Priority: 2}
	tx1 := tx.NewValidTransaction(e1, v1)
	babesession.PushToTxQueue(tx1)

	e2 := &types.Extrinsic{0x04, 0x05, 0x06, 0x07}
	v2 := &tx.Validity{Priority: 1}
	tx2 := tx.NewValidTransaction(e2, v2)
	babesession.PushToTxQueue(tx2)

	// Create a block to put the transactions into
	fmt.Println("@@@@@@@")
	zeroHash, err := common.HexToHash("0x00")
	if err != nil {
		t.Fatalf("Can't convert hex 0x00 to hash")
	}

	fmt.Println("@@@@@@@")

	block := types.Block{
		Header: types.BlockHeader{
			ParentHash: zeroHash,
			Number:     big.NewInt(0),
		},
		Body: types.BlockBody{},
	}

	fmt.Println("@@@@@@@")

	// Create slot for block
	slot := Slot{
		start:    uint64(time.Now().Unix()),
		duration: uint64(10000),
		number:   1,
	}

	fmt.Println("@@@@@@@@@@@@@@@@@@@@@")

	resultBlock, err := babesession.buildBlock(block, slot, common.Hash{0x00})
	fmt.Println("@@@@@@@@@@@@@@@@@@@@@")
	if err != nil {
		t.Fatal("buildblock test failed: ", err)
	}

	fmt.Println("@@@@@@@@@@@@@@@@@@@@@")
	t.Log("Got back block: ", resultBlock)

	// e2 := []byte{'d', 'e', 'f'}

	// babesession.PushToTxQueue()

	// babesession.buildBlock()

	// babesession.PushToTxQueue()

}
