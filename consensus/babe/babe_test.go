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
	"reflect"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/core/blocktree"
	"github.com/ChainSafe/gossamer/core/types"
	"github.com/ChainSafe/gossamer/p2p"
	db "github.com/ChainSafe/gossamer/polkadb"
	"github.com/ChainSafe/gossamer/runtime"
	"github.com/ChainSafe/gossamer/trie"
)

const POLKADOT_RUNTIME_FP string = "../../substrate_test_runtime.compact.wasm"
const POLKADOT_RUNTIME_URL string = "https://github.com/noot/substrate/blob/add-blob/core/test-runtime/wasm/wasm32-unknown-unknown/release/wbuild/substrate-test-runtime/substrate_test_runtime.compact.wasm?raw=true"

var zeroHash, _ = common.HexToHash("0x00")

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

func newRuntime(t *testing.T) *runtime.Runtime {
	_, err := getRuntimeBlob()
	if err != nil {
		t.Fatalf("Fail: could not get polkadot runtime")
	}

	fp, err := filepath.Abs(POLKADOT_RUNTIME_FP)
	if err != nil {
		t.Fatal("could not create filepath")
	}

	tt := &trie.Trie{}

	r, err := runtime.NewRuntimeFromFile(fp, tt)
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
	babesession, err := NewSession([32]byte{}, [64]byte{}, rt, nil)
	if err != nil {
		t.Fatal(err)
	}
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

	_, err = babesession.runLottery(0)
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
	babesession, err := NewSession([32]byte{}, [64]byte{}, rt, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = babesession.configurationFromRuntime()
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

	if babesession.config == expected {
		t.Errorf("Fail: got %v expected %v\n", babesession.config, expected)
	}
}

func TestMedian_OddLength(t *testing.T) {
	us := []uint64{3, 2, 1, 4, 5}
	res, err := median(us)
	if err != nil {
		t.Fatal(err)
	}

	var expected uint64 = 3

	if res != expected {
		t.Errorf("Fail: got %v expected %v\n", res, expected)
	}

}

func TestMedian_EvenLength(t *testing.T) {
	us := []uint64{1, 4, 2, 4, 5, 6}
	res, err := median(us)
	if err != nil {
		t.Fatal(err)
	}

	var expected uint64 = 4

	if res != expected {
		t.Errorf("Fail: got %v expected %v\n", res, expected)
	}

}

func TestSlotOffset_Failing(t *testing.T) {
	var st uint64 = 1000001
	var se uint64 = 1000000

	_, err := slotOffset(st, se)
	if err == nil {
		t.Fatal("Fail: did not err for c>1")
	}

}

func TestSlotOffset(t *testing.T) {
	var st uint64 = 1000000
	var se uint64 = 1000001

	res, err := slotOffset(st, se)
	if err != nil {
		t.Fatal(err)
	}

	var expected uint64 = 1

	if res != expected {
		t.Errorf("Fail: got %v expected %v\n", res, expected)
	}

}

func createFlatBlockTree(t *testing.T, depth int) *blocktree.BlockTree {

	genesisBlock := types.Block{
		Header: types.BlockHeader{
			ParentHash: zeroHash,
			Number:     big.NewInt(0),
			Hash:       common.Hash{0x00},
		},
		Body: types.BlockBody{},
	}

	genesisBlock.SetBlockArrivalTime(uint64(1000))

	d := &db.BlockDB{
		Db: db.NewMemDatabase(),
	}

	bt := blocktree.NewBlockTreeFromGenesis(genesisBlock, d)
	previousHash := genesisBlock.Header.Hash
	previousAT := genesisBlock.GetBlockArrivalTime()

	for i := 1; i <= depth; i++ {
		hex := fmt.Sprintf("%06x", i)

		hash, err := common.HexToHash("0x" + hex)

		if err != nil {
			t.Error(err)
		}

		block := types.Block{
			Header: types.BlockHeader{
				ParentHash: previousHash,
				Hash:       hash,
				Number:     big.NewInt(int64(i)),
			},
			Body: types.BlockBody{},
		}

		block.SetBlockArrivalTime(previousAT + uint64(1000))

		bt.AddBlock(block)
		previousHash = hash
		previousAT = block.GetBlockArrivalTime()
	}

	return bt

}

func TestSlotTime(t *testing.T) {
	rt := newRuntime(t)
	bt := createFlatBlockTree(t, 100)
	babesession, err := NewSession([32]byte{}, [64]byte{}, rt, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = babesession.configurationFromRuntime()
	if err != nil {
		t.Fatal(err)
	}

	res, err := babesession.slotTime(103, bt, 20)
	if err != nil {
		t.Fatal(err)
	}

	var expected uint64 = 104000

	if res != expected {
		t.Errorf("Fail: got %v expected %v\n", res, expected)
	}

}

func TestStart(t *testing.T) {
	rt := newRuntime(t)
	babesession, err := NewSession([32]byte{}, [64]byte{}, rt, nil)
	if err != nil {
		t.Fatal(err)
	}
	babesession.authorityIndex = 0
	babesession.authorityWeights = []uint64{1}
	conf := &BabeConfiguration{
		SlotDuration:       1,
		EpochLength:        6,
		C1:                 1,
		C2:                 10,
		GenesisAuthorities: []AuthorityData{},
		Randomness:         0,
		SecondarySlots:     false,
	}
	babesession.config = conf

	err = babesession.Start()
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Duration(conf.SlotDuration) * time.Duration(conf.EpochLength) * time.Millisecond)
}

func TestBabeAnnounceMessage(t *testing.T) {
	rt := newRuntime(t)

	// Block Announce Channel called when Build-Block Creates a block
	blockAnnounceChan := make(chan p2p.BlockAnnounceMessage)
	babesession, err := NewSession([32]byte{}, [64]byte{}, rt, blockAnnounceChan)
	if err != nil {
		t.Fatal(err)
	}
	babesession.authorityIndex = 0
	babesession.authorityWeights = []uint64{1, 1, 1}

	err = babesession.Start()
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Duration(babesession.config.SlotDuration) * time.Duration(babesession.config.EpochLength) * time.Millisecond)

	for i := 0; i < int(babesession.config.EpochLength); i++ {
		blk := <-blockAnnounceChan

		expectedBlockAnnounceMsg := p2p.BlockAnnounceMessage{
			Number: big.NewInt(int64(i)),
		}

		if !reflect.DeepEqual(blk, expectedBlockAnnounceMsg) {
			t.Fatalf("Didn't receive the correct block: %+v\nExpected block: %+v", blk, expectedBlockAnnounceMsg)
		}
	}

}
