// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package ethash

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"time"

	"git.egem.io/team/go-egem/common"
	"git.egem.io/team/go-egem/common/math"
	"git.egem.io/team/go-egem/consensus"
	"git.egem.io/team/go-egem/consensus/misc"
	"git.egem.io/team/go-egem/core/state"
	"git.egem.io/team/go-egem/core/types"
	//"git.egem.io/team/go-egem/log"
	"git.egem.io/team/go-egem/params"
	set "gopkg.in/fatih/set.v0"
)

// Ethash proof-of-work protocol constants.
var (
	maxUncles              = 2                // Maximum number of uncles allowed in a single block
	allowedFutureBlockTime = 15 * time.Second // Max time from current time allowed for blocks, before they're considered future blocks
)

//  EGEM Variables
var (
	egem0BlockReward          *big.Int = big.NewInt(8e+18)                                                 //  8 EGEM Block reward in wei for successfully mining a block.     (ERA0)
	egem1BlockReward          *big.Int = big.NewInt(4e+18)                                                 //  4 EGEM Block reward in wei for successfully mining a block.     (ERA1)
	egem2BlockReward          *big.Int = big.NewInt(2e+18)                                                 //  2 EGEM Block reward in wei for successfully mining a block.     (ERA2)
	egem3BlockReward          *big.Int = big.NewInt(1e+18)                                                 //  1 EGEM Block reward in wei for successfully mining a block.     (ERA3)
	egem4BlockReward          *big.Int = big.NewInt(500000000000000000)                                    //  0.5 EGEM Block reward in wei for successfully mining a block.   (ERA4)
	egem5BlockReward          *big.Int = big.NewInt(250000000000000000)                                    //  0.25 EGEM Block reward in wei for successfully mining a block.  (ERA5)
	egem6BlockReward          *big.Int = big.NewInt(125000000000000000)                                    //  0.125 EGEM Block reward in wei for successfully mining a block. (ERA6)
	egem0DevReward            *big.Int = big.NewInt(250000000000000000)                                    //  Era0 1  EGEM per block.
	egem1DevReward            *big.Int = big.NewInt(187500000000000000)                                    //  Era1 0.75 EGEM per block.
	egem2DevReward            *big.Int = big.NewInt(125000000000000000)                                    //  Era2 0.5 EGEM per block.
	egem3DevReward            *big.Int = big.NewInt(62500000000000000)                                     //  Era3 0.25 EGEM per block
	egem4DevReward            *big.Int = big.NewInt(25000000000000000)                                     //  Era4 0.1 EGEM per block.
	egem5DevReward            *big.Int = big.NewInt(12500000000000000)                                     //  Era5 0.05 EGEM per block.
	egem6DevReward            *big.Int = big.NewInt(6250000000000000)                                      //  Era6 0.025 EGEM per block.
	egemRewardSwitchBlockEra0 *big.Int = big.NewInt(5000)                                                  //  5K Block transition
	egemRewardSwitchBlockEra1 *big.Int = big.NewInt(2500000)                                               //  2.5M block transition
	egemRewardSwitchBlockEra2 *big.Int = big.NewInt(5000000)                                               //  5M block transition
	egemRewardSwitchBlockEra3 *big.Int = big.NewInt(7500000)                                               //  7.5M block transition
	egemRewardSwitchBlockEra4 *big.Int = big.NewInt(10000000)                                              //  10M block transtiton
	egemRewardSwitchBlockEra5 *big.Int = big.NewInt(12500000)                                              //  12.5M block transtiton
	egemRewardSwitchBlockEra6 *big.Int = big.NewInt(15000000)                                              //  15M block transtiton
	devFund0                           = common.HexToAddress("0x3fa6576610cac6c68e88ee68de07b104c9524fda") // invalid as of block 350k
	devFund1                           = common.HexToAddress("0xfc0f0a5F06cB00c9EB435127142ac79ac6F48B94") // invalid as of block 350k
	devFund2                           = common.HexToAddress("0x0666bf13ab1902de7dee4f8193c819118d7e21a6") // invalid as of block 350k
	devFund3                           = common.HexToAddress("0xcEf0890408b4FC0DC025c8F581c77383529D38B6") // invalid as of block 350k
)

// Fork Variables
// Moves funds to multisigs.
var (
	egemSwitchBlock            = uint64(350000) //  350k block transtiton
	egemSwitchBlockB  *big.Int = big.NewInt(350000)
	devFund0F                  = common.HexToAddress("0x1140e31A4A7ae014E55f6c235af027C5CFABCA17") // riddlez 0x1140e31A4A7ae014E55f6c235af027C5CFABCA17
	devFund1F                  = common.HexToAddress("0x63e9ceFD428D37430205c0ab8fa2a34A21F911Ac") // beast/tbates 0x63e9ceFD428D37430205c0ab8fa2a34A21F911Ac
	devFund2F                  = common.HexToAddress("0x2025ed239a8dec4de0034a252d5c5e385b73fcd0") // osoese 0x2025ed239a8dec4de0034a252d5c5e385b73fcd0
	devFund3F                  = common.HexToAddress("0xe485aA04bb231f331B85BF64614737c6495CC4b3") // jal 0xe485aA04bb231f331B85BF64614737c6495CC4b3
	egemSwitchBlock2           = uint64(1850000)                                                   //  1.85m block transtiton
	egemSwitchBlockB2 *big.Int = big.NewInt(1850000)                                               //  1.85m block transtiton
	egemSwitchBlock3           = uint64(2100000)                                                   //  2.1m block transtiton for node fund fix.
	egemSwitchBlockB3 *big.Int = big.NewInt(2100000)
	egemSwitchBlock4           = uint64(2400000)                                                    //  2.4m uncle finder pay enabled.
	egemSwitchBlockB4 *big.Int = big.NewInt(2400000)                                                //  2.4m uncle finder pay enabled.
	nodeFund                   = common.HexToAddress("0x87045b7badac9c2da19f5b0ee2bcea943a786644 ") // node multisig failed due to space.
	nodeFundFixed              = common.HexToAddress("0x87045b7badac9c2da19f5b0ee2bcea943a786644")  // node multisig fixed space... 0x87045b7badac9c2da19f5b0ee2bcea943a786644
)

// Various error messages to mark blocks invalid. These should be private to
// prevent engine specific errors from being referenced in the remainder of the
// codebase, inherently breaking if the engine is swapped out. Please put common
// error types into the consensus package.
var (
	errLargeBlockTime    = errors.New("timestamp too big")
	errZeroBlockTime     = errors.New("timestamp equals parent's")
	errTooManyUncles     = errors.New("too many uncles")
	errDuplicateUncle    = errors.New("duplicate uncle")
	errUncleIsAncestor   = errors.New("uncle is ancestor")
	errDanglingUncle     = errors.New("uncle's parent is not ancestor")
	errInvalidDifficulty = errors.New("non-positive difficulty")
	errInvalidMixDigest  = errors.New("invalid mix digest")
	errInvalidPoW        = errors.New("invalid proof-of-work")
)

// Author implements consensus.Engine, returning the header's coinbase as the
// proof-of-work verified author of the block.
func (ethash *Ethash) Author(header *types.Header) (common.Address, error) {
	return header.Coinbase, nil
}

// VerifyHeader checks whether a header conforms to the consensus rules of the
// stock Ethereum ethash engine.
func (ethash *Ethash) VerifyHeader(chain consensus.ChainReader, header *types.Header, seal bool) error {
	// If we're running a full engine faking, accept any input as valid
	if ethash.config.PowMode == ModeFullFake {
		return nil
	}
	// Short circuit if the header is known, or it's parent not
	number := header.Number.Uint64()
	if chain.GetHeader(header.Hash(), number) != nil {
		return nil
	}
	parent := chain.GetHeader(header.ParentHash, number-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	// Sanity checks passed, do a proper verification
	return ethash.verifyHeader(chain, header, parent, false, seal)
}

// VerifyHeaders is similar to VerifyHeader, but verifies a batch of headers
// concurrently. The method returns a quit channel to abort the operations and
// a results channel to retrieve the async verifications.
func (ethash *Ethash) VerifyHeaders(chain consensus.ChainReader, headers []*types.Header, seals []bool) (chan<- struct{}, <-chan error) {
	// If we're running a full engine faking, accept any input as valid
	if ethash.config.PowMode == ModeFullFake || len(headers) == 0 {
		abort, results := make(chan struct{}), make(chan error, len(headers))
		for i := 0; i < len(headers); i++ {
			results <- nil
		}
		return abort, results
	}

	// Spawn as many workers as allowed threads
	workers := runtime.GOMAXPROCS(0)
	if len(headers) < workers {
		workers = len(headers)
	}

	// Create a task channel and spawn the verifiers
	var (
		inputs = make(chan int)
		done   = make(chan int, workers)
		errors = make([]error, len(headers))
		abort  = make(chan struct{})
	)
	for i := 0; i < workers; i++ {
		go func() {
			for index := range inputs {
				errors[index] = ethash.verifyHeaderWorker(chain, headers, seals, index)
				done <- index
			}
		}()
	}

	errorsOut := make(chan error, len(headers))
	go func() {
		defer close(inputs)
		var (
			in, out = 0, 0
			checked = make([]bool, len(headers))
			inputs  = inputs
		)
		for {
			select {
			case inputs <- in:
				if in++; in == len(headers) {
					// Reached end of headers. Stop sending to workers.
					inputs = nil
				}
			case index := <-done:
				for checked[index] = true; checked[out]; out++ {
					errorsOut <- errors[out]
					if out == len(headers)-1 {
						return
					}
				}
			case <-abort:
				return
			}
		}
	}()
	return abort, errorsOut
}

func (ethash *Ethash) verifyHeaderWorker(chain consensus.ChainReader, headers []*types.Header, seals []bool, index int) error {
	var parent *types.Header
	if index == 0 {
		parent = chain.GetHeader(headers[0].ParentHash, headers[0].Number.Uint64()-1)
	} else if headers[index-1].Hash() == headers[index].ParentHash {
		parent = headers[index-1]
	}
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	if chain.GetHeader(headers[index].Hash(), headers[index].Number.Uint64()) != nil {
		return nil // known block
	}
	return ethash.verifyHeader(chain, headers[index], parent, false, seals[index])
}

// VerifyUncles verifies that the given block's uncles conform to the consensus
// rules of the stock Ethereum ethash engine.
func (ethash *Ethash) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	// If we're running a full engine faking, accept any input as valid
	if ethash.config.PowMode == ModeFullFake {
		return nil
	}
	// Verify that there are at most 2 uncles included in this block
	if len(block.Uncles()) > maxUncles {
		return errTooManyUncles
	}
	// Gather the set of past uncles and ancestors
	uncles, ancestors := set.New(), make(map[common.Hash]*types.Header)

	number, parent := block.NumberU64()-1, block.ParentHash()
	for i := 0; i < 7; i++ {
		ancestor := chain.GetBlock(parent, number)
		if ancestor == nil {
			break
		}
		ancestors[ancestor.Hash()] = ancestor.Header()
		for _, uncle := range ancestor.Uncles() {
			uncles.Add(uncle.Hash())
		}
		parent, number = ancestor.ParentHash(), number-1
	}
	ancestors[block.Hash()] = block.Header()
	uncles.Add(block.Hash())

	// Verify each of the uncles that it's recent, but not an ancestor
	for _, uncle := range block.Uncles() {
		// Make sure every uncle is rewarded only once
		hash := uncle.Hash()
		if uncles.Has(hash) {
			return errDuplicateUncle
		}
		uncles.Add(hash)

		// Make sure the uncle has a valid ancestry
		if ancestors[hash] != nil {
			return errUncleIsAncestor
		}
		if ancestors[uncle.ParentHash] == nil || uncle.ParentHash == block.ParentHash() {
			return errDanglingUncle
		}
		if err := ethash.verifyHeader(chain, uncle, ancestors[uncle.ParentHash], true, true); err != nil {
			return err
		}
	}
	return nil
}

// verifyHeader checks whether a header conforms to the consensus rules of the
// stock Ethereum ethash engine.
// See YP section 4.3.4. "Block Header Validity"
func (ethash *Ethash) verifyHeader(chain consensus.ChainReader, header, parent *types.Header, uncle bool, seal bool) error {
	// Ensure that the header's extra-data section is of a reasonable size
	if uint64(len(header.Extra)) > params.MaximumExtraDataSize {
		return fmt.Errorf("extra-data too long: %d > %d", len(header.Extra), params.MaximumExtraDataSize)
	}
	// Verify the header's timestamp
	if uncle {
		if header.Time.Cmp(math.MaxBig256) > 0 {
			return errLargeBlockTime
		}
	} else {
		if header.Time.Cmp(big.NewInt(time.Now().Add(allowedFutureBlockTime).Unix())) > 0 {
			return consensus.ErrFutureBlock
		}
	}
	if header.Time.Cmp(parent.Time) <= 0 {
		return errZeroBlockTime
	}
	// Verify the block's difficulty based in it's timestamp and parent's difficulty
	expected := ethash.CalcDifficulty(chain, header.Time.Uint64(), parent)

	if expected.Cmp(header.Difficulty) != 0 {
		return fmt.Errorf("invalid difficulty: have %v, want %v", header.Difficulty, expected)
	}
	// Verify that the gas limit is <= 2^63-1
	cap := uint64(0x7fffffffffffffff)
	if header.GasLimit > cap {
		return fmt.Errorf("invalid gasLimit: have %v, max %v", header.GasLimit, cap)
	}
	// Verify that the gasUsed is <= gasLimit
	if header.GasUsed > header.GasLimit {
		return fmt.Errorf("invalid gasUsed: have %d, gasLimit %d", header.GasUsed, header.GasLimit)
	}

	// Verify that the gas limit remains within allowed bounds
	diff := int64(parent.GasLimit) - int64(header.GasLimit)
	if diff < 0 {
		diff *= -1
	}
	limit := parent.GasLimit / params.GasLimitBoundDivisor

	if uint64(diff) >= limit || header.GasLimit < params.MinGasLimit {
		return fmt.Errorf("invalid gas limit: have %d, want %d += %d", header.GasLimit, parent.GasLimit, limit)
	}
	// Verify that the block number is parent's +1
	if diff := new(big.Int).Sub(header.Number, parent.Number); diff.Cmp(big.NewInt(1)) != 0 {
		return consensus.ErrInvalidNumber
	}
	// Verify the engine specific seal securing the block
	if seal {
		if err := ethash.VerifySeal(chain, header); err != nil {
			return err
		}
	}
	// If all checks passed, validate any special fields for hard forks
	if err := misc.VerifyDAOHeaderExtraData(chain.Config(), header); err != nil {
		return err
	}
	if err := misc.VerifyForkHashes(chain.Config(), header, uncle); err != nil {
		return err
	}
	return nil
}

// CalcDifficulty is the difficulty adjustment algorithm. It returns
// the difficulty that a new block should have when created at time
// given the parent block's time and difficulty.
func (ethash *Ethash) CalcDifficulty(chain consensus.ChainReader, time uint64, parent *types.Header) *big.Int {
	if parent.Number.Cmp(egemSwitchBlockB4) == 1 {
		// Final adjustment
		return calcDifficultyEGEM4(time, parent)
	} else if parent.Number.Cmp(egemSwitchBlockB3) == 1 {
		// Fixed Fork difficulty
		return calcDifficultyEGEM3(time, parent)
	} else if parent.Number.Cmp(egemSwitchBlockB2) == 1 {
		// Fork difficulty
		return calcDifficultyEGEM2(time, parent)
	} else {
		// Genesis difficulty
		return calcDifficultyEGEM(time, parent)
	}
}

// Some weird constants to avoid constant memory allocs for them.
var (
	big1   = big.NewInt(1)
	big2   = big.NewInt(2)
	big3   = big.NewInt(3)
	big4   = big.NewInt(4)
	big5   = big.NewInt(5)
	big6   = big.NewInt(6)
	big7   = big.NewInt(7)
	big8   = big.NewInt(8)
	big9   = big.NewInt(9)
	big32  = big.NewInt(32)
	big99  = big.NewInt(99)
	big100 = big.NewInt(100)
)

// EGEM Difficulty Algo
// * +/- adjustment per block
// including a randomizer
//

func calcDifficultyEGEM(time uint64, parent *types.Header) *big.Int {
	diff := new(big.Int)
	adjustUp := new(big.Int).Div(parent.Difficulty, big7)
	adjustDown := new(big.Int).Div(parent.Difficulty, big3)

	bigTime := new(big.Int)
	bigParentTime := new(big.Int)

	bigTime.SetUint64(time)
	bigParentTime.Set(parent.Time)

	if bigTime.Sub(bigTime, bigParentTime).Cmp(params.DurationLimit) < 0 {
		diff.Add(parent.Difficulty, big7)
		diff.Add(diff, adjustUp)
	} else {
		diff.Sub(parent.Difficulty, big3)
		diff.Sub(diff, adjustDown)
	}

	if diff.Cmp(params.MinimumDifficulty) < 0 {
		diff.Set(params.MinimumDifficulty)
	}
	// context := []interface{}{
	// 	"picked", diff,
	// }
	// log.Info("Next Block Difficulty ", context...)
	return diff
}

func calcDifficultyEGEM2(time uint64, parent *types.Header) *big.Int {
	diff := new(big.Int)
	adjustUp := new(big.Int).Div(parent.Difficulty, big9)
	adjustDown := new(big.Int).Div(parent.Difficulty, big8)

	bigTime := new(big.Int)
	bigParentTime := new(big.Int)

	bigTime.SetUint64(time)
	bigParentTime.Set(parent.Time)

	if bigTime.Sub(bigTime, bigParentTime).Cmp(params.DurationLimitNew) < 0 {
		diff.Add(parent.Difficulty, big9)
		diff.Add(diff, adjustUp)
	} else {
		diff.Sub(parent.Difficulty, big8)
		diff.Sub(diff, adjustDown)
	}

	if diff.Cmp(params.MinimumDifficultyNew) < 0 {
		diff.Set(params.MinimumDifficultyNew)
	}
	// context := []interface{}{
	// 	"picked", diff,
	// }
	// log.Info("Next Block Difficulty ", context...)
	return diff
}

func calcDifficultyEGEM3(time uint64, parent *types.Header) *big.Int {
	diff := new(big.Int)
	adjustUp := new(big.Int).Div(parent.Difficulty, big99)
	adjustDown := new(big.Int).Div(parent.Difficulty, big100)

	bigTime := new(big.Int)
	bigParentTime := new(big.Int)

	bigTime.SetUint64(time)
	bigParentTime.Set(parent.Time)

	if bigTime.Sub(bigTime, bigParentTime).Cmp(params.DurationLimitFixed) < 0 {
		diff.Add(parent.Difficulty, big99)
		diff.Add(diff, adjustUp)
	} else {
		diff.Sub(parent.Difficulty, big100)
		diff.Sub(diff, adjustDown)
	}

	if diff.Cmp(params.MinimumDifficultyNew) < 0 {
		diff.Set(params.MinimumDifficultyNew)
	}
	// context := []interface{}{
	// 	"picked", diff,
	// }
	// log.Info("Next Block Difficulty ", context...)
	return diff
}

func calcDifficultyEGEM4(time uint64, parent *types.Header) *big.Int {
	diff := new(big.Int)
	adjustUp := new(big.Int).Div(parent.Difficulty, big99)
	adjustDown := new(big.Int).Div(parent.Difficulty, big100)

	bigTime := new(big.Int)
	bigParentTime := new(big.Int)

	bigTime.SetUint64(time)
	bigParentTime.Set(parent.Time)

	if bigTime.Sub(bigTime, bigParentTime).Cmp(params.DurationLimitFinal) < 0 {
		diff.Add(parent.Difficulty, big99)
		diff.Add(diff, adjustUp)
	} else {
		diff.Sub(parent.Difficulty, big100)
		diff.Sub(diff, adjustDown)
	}

	if diff.Cmp(params.MinimumDifficultyNew) < 0 {
		diff.Set(params.MinimumDifficultyNew)
	}
	// context := []interface{}{
	// 	"picked", diff,
	// }
	// log.Info("Next Block Difficulty ", context...)
	return diff
}

// VerifySeal implements consensus.Engine, checking whether the given block satisfies
// the PoW difficulty requirements.
func (ethash *Ethash) VerifySeal(chain consensus.ChainReader, header *types.Header) error {
	// If we're running a fake PoW, accept any seal as valid
	if ethash.config.PowMode == ModeFake || ethash.config.PowMode == ModeFullFake {
		time.Sleep(ethash.fakeDelay)
		if ethash.fakeFail == header.Number.Uint64() {
			return errInvalidPoW
		}
		return nil
	}
	// If we're running a shared PoW, delegate verification to it
	if ethash.shared != nil {
		return ethash.shared.VerifySeal(chain, header)
	}
	// Ensure that we have a valid difficulty for the block
	if header.Difficulty.Sign() <= 0 {
		return errInvalidDifficulty
	}
	// Recompute the digest and PoW value and verify against the header
	number := header.Number.Uint64()

	cache := ethash.cache(number)
	size := datasetSize(number)
	if ethash.config.PowMode == ModeTest {
		size = 32 * 1024
	}
	digest, result := hashimotoLight(size, cache.cache, header.HashNoNonce().Bytes(), header.Nonce.Uint64())
	// Caches are unmapped in a finalizer. Ensure that the cache stays live
	// until after the call to hashimotoLight so it's not unmapped while being used.
	runtime.KeepAlive(cache)

	if !bytes.Equal(header.MixDigest[:], digest) {
		return errInvalidMixDigest
	}
	target := new(big.Int).Div(maxUint256, header.Difficulty)
	if new(big.Int).SetBytes(result).Cmp(target) > 0 {
		return errInvalidPoW
	}
	return nil
}

// Prepare implements consensus.Engine, initializing the difficulty field of a
// header to conform to the ethash protocol. The changes are done inline.
func (ethash *Ethash) Prepare(chain consensus.ChainReader, header *types.Header) error {
	parent := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	header.Difficulty = ethash.CalcDifficulty(chain, header.Time.Uint64(), parent)
	return nil
}

// Finalize implements consensus.Engine, accumulating the block and uncle rewards,
// setting the final state and assembling the block.
func (ethash *Ethash) Finalize(chain consensus.ChainReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header, receipts []*types.Receipt) (*types.Block, error) {

	a := header.Number.Uint64()
	b := egemSwitchBlock
	c := egemSwitchBlock2
	d := egemSwitchBlock3
	e := egemSwitchBlock4

	if a > e {
		// Accumulate any block and uncle rewards and commit the final state root
		accumulateRewards5(chain.Config(), state, header, uncles)
		header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))

		// Header seems complete, assemble into a block and return
		return types.NewBlock(header, txs, uncles, receipts), nil

	} else if a > d {
		// Accumulate any block and uncle rewards and commit the final state root
		accumulateRewards4(chain.Config(), state, header, uncles)
		header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))

		// Header seems complete, assemble into a block and return
		return types.NewBlock(header, txs, uncles, receipts), nil

	} else if a > c {
		// Accumulate any block and uncle rewards and commit the final state root
		accumulateRewards3(chain.Config(), state, header, uncles)
		header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))

		// Header seems complete, assemble into a block and return
		return types.NewBlock(header, txs, uncles, receipts), nil

	} else if a > b {
		// Accumulate any block and uncle rewards and commit the final state root
		accumulateRewards2(chain.Config(), state, header, uncles)
		header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))

		// Header seems complete, assemble into a block and return
		return types.NewBlock(header, txs, uncles, receipts), nil

	} else {
		// Accumulate any block and uncle rewards and commit the final state root
		accumulateRewards(chain.Config(), state, header, uncles)
		header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))

		// Header seems complete, assemble into a block and return
		return types.NewBlock(header, txs, uncles, receipts), nil

	}
}

// AccumulateRewards credits the coinbase of the given block with the mining
// reward. The total reward consists of the static block reward and rewards for
// included uncles. The coinbase of each uncle block is also rewarded.
func accumulateRewards(config *params.ChainConfig, state *state.StateDB, header *types.Header, uncles []*types.Header) {

	// Select the correct block reward based on chain progression
	block0Reward := egem0BlockReward
	block1Reward := egem1BlockReward
	block2Reward := egem2BlockReward
	block3Reward := egem3BlockReward
	block4Reward := egem4BlockReward
	block5Reward := egem5BlockReward
	block6Reward := egem6BlockReward
	d0Reward := egem0DevReward
	d1Reward := egem1DevReward
	d2Reward := egem2DevReward
	d3Reward := egem3DevReward
	d4Reward := egem4DevReward
	d5Reward := egem5DevReward
	d6Reward := egem6DevReward

	// Accumulate the rewards for the miner and any included uncles
	if header.Number.Cmp(egemRewardSwitchBlockEra6) == 1 {
		reward := new(big.Int).Set(block6Reward)
		r := new(big.Int)
		for _, uncle := range uncles {
			r.Add(uncle.Number, big8)
			r.Sub(r, header.Number)
			r.Mul(r, reward)
			r.Div(r, big8)

			r.Div(reward, big32)
			reward.Add(reward, r)
		}
		//fmt.Println("Miner Block Reward:", reward, "in Wei.", "|", "Dev Block Fee:", d6Reward, "in Wei.")
		state.AddBalance(header.Coinbase, reward)
		state.AddBalance(devFund0, d6Reward)
		state.AddBalance(devFund1, d6Reward)
		state.AddBalance(devFund2, d6Reward)
		state.AddBalance(devFund3, d6Reward)

	} else if header.Number.Cmp(egemRewardSwitchBlockEra5) == 1 {
		reward := new(big.Int).Set(block5Reward)
		r := new(big.Int)
		for _, uncle := range uncles {
			r.Add(uncle.Number, big8)
			r.Sub(r, header.Number)
			r.Mul(r, reward)
			r.Div(r, big8)

			r.Div(reward, big32)
			reward.Add(reward, r)
		}
		//fmt.Println("Miner Block Reward:", reward, "in Wei.", "|", "Dev Block Fee:", d5Reward, "in Wei.")
		state.AddBalance(header.Coinbase, reward)
		state.AddBalance(devFund0, d5Reward)
		state.AddBalance(devFund1, d5Reward)
		state.AddBalance(devFund2, d5Reward)
		state.AddBalance(devFund3, d5Reward)

	} else if header.Number.Cmp(egemRewardSwitchBlockEra4) == 1 {
		reward := new(big.Int).Set(block4Reward)
		r := new(big.Int)
		for _, uncle := range uncles {
			r.Add(uncle.Number, big8)
			r.Sub(r, header.Number)
			r.Mul(r, reward)
			r.Div(r, big8)

			r.Div(reward, big32)
			reward.Add(reward, r)
		}
		//fmt.Println("Miner Block Reward:", reward, "in Wei.", "|", "Dev Block Fee:", d4Reward, "in Wei.")
		state.AddBalance(header.Coinbase, reward)
		state.AddBalance(devFund0, d4Reward)
		state.AddBalance(devFund1, d4Reward)
		state.AddBalance(devFund2, d4Reward)
		state.AddBalance(devFund3, d4Reward)

	} else if header.Number.Cmp(egemRewardSwitchBlockEra3) == 1 {
		reward := new(big.Int).Set(block3Reward)
		r := new(big.Int)
		for _, uncle := range uncles {
			r.Add(uncle.Number, big8)
			r.Sub(r, header.Number)
			r.Mul(r, reward)
			r.Div(r, big8)

			r.Div(reward, big32)
			reward.Add(reward, r)
		}
		//fmt.Println("Miner Block Reward:", reward, "in Wei.", "|", "Dev Block Fee:", d3Reward, "in Wei.")
		state.AddBalance(header.Coinbase, reward)
		state.AddBalance(devFund0, d3Reward)
		state.AddBalance(devFund1, d3Reward)
		state.AddBalance(devFund2, d3Reward)
		state.AddBalance(devFund3, d3Reward)

	} else if header.Number.Cmp(egemRewardSwitchBlockEra2) == 1 {
		reward := new(big.Int).Set(block2Reward)
		r := new(big.Int)
		for _, uncle := range uncles {
			r.Add(uncle.Number, big8)
			r.Sub(r, header.Number)
			r.Mul(r, reward)
			r.Div(r, big8)

			r.Div(reward, big32)
			reward.Add(reward, r)
		}
		//fmt.Println("Miner Block Reward:", reward, "in Wei.", "|", "Dev Block Fee:", d2Reward, "in Wei.")
		state.AddBalance(header.Coinbase, reward)
		state.AddBalance(devFund0, d2Reward)
		state.AddBalance(devFund1, d2Reward)
		state.AddBalance(devFund2, d2Reward)
		state.AddBalance(devFund3, d2Reward)

	} else if header.Number.Cmp(egemRewardSwitchBlockEra1) == 1 {
		reward := new(big.Int).Set(block1Reward)
		r := new(big.Int)
		for _, uncle := range uncles {
			r.Add(uncle.Number, big8)
			r.Sub(r, header.Number)
			r.Mul(r, reward)
			r.Div(r, big8)

			r.Div(reward, big32)
			reward.Add(reward, r)
		}
		//fmt.Println("Miner Block Reward:", reward, "in Wei.", "|", "Dev Block Fee:", d1Reward, "in Wei.")
		state.AddBalance(header.Coinbase, reward)
		state.AddBalance(devFund0, d1Reward)
		state.AddBalance(devFund1, d1Reward)
		state.AddBalance(devFund2, d1Reward)
		state.AddBalance(devFund3, d1Reward)

	} else if header.Number.Cmp(egemRewardSwitchBlockEra0) == 1 {
		reward := new(big.Int).Set(block0Reward)
		r := new(big.Int)
		for _, uncle := range uncles {
			r.Add(uncle.Number, big8)
			r.Sub(r, header.Number)
			r.Mul(r, reward)
			r.Div(r, big8)

			r.Div(reward, big32)
			reward.Add(reward, r)
		}
		//fmt.Println("Miner Block Reward:", reward, "in Wei.", "|", "Dev Block Fee:", d0Reward, "in Wei.")
		state.AddBalance(header.Coinbase, reward)
		state.AddBalance(devFund0, d0Reward)
		state.AddBalance(devFund1, d0Reward)
		state.AddBalance(devFund2, d0Reward)
		state.AddBalance(devFund3, d0Reward)

	} else {
		reward := new(big.Int).Set(block0Reward)
		r := new(big.Int)
		for _, uncle := range uncles {
			r.Add(uncle.Number, big8)
			r.Sub(r, header.Number)
			r.Mul(r, reward)
			r.Div(r, big8)

			r.Div(reward, big32)
			reward.Add(reward, r)
		}
		//fmt.Println("Miner Block Reward:", reward, "in Wei.")
		state.AddBalance(header.Coinbase, reward)
	}

}

// AccumulateRewards credits the coinbase of the given block with the mining
// reward. The total reward consists of the static block reward and rewards for
// included uncles. The coinbase of each uncle block is also rewarded.
func accumulateRewards2(config *params.ChainConfig, state *state.StateDB, header *types.Header, uncles []*types.Header) {

	// Select the correct block reward based on chain progression
	block0Reward := egem0BlockReward
	block1Reward := egem1BlockReward
	block2Reward := egem2BlockReward
	block3Reward := egem3BlockReward
	block4Reward := egem4BlockReward
	block5Reward := egem5BlockReward
	block6Reward := egem6BlockReward
	d0Reward := egem0DevReward
	d1Reward := egem1DevReward
	d2Reward := egem2DevReward
	d3Reward := egem3DevReward
	d4Reward := egem4DevReward
	d5Reward := egem5DevReward
	d6Reward := egem6DevReward

	// Accumulate the rewards for the miner and any included uncles
	if header.Number.Cmp(egemRewardSwitchBlockEra6) == 1 {
		reward := new(big.Int).Set(block6Reward)
		r := new(big.Int)
		for _, uncle := range uncles {
			r.Add(uncle.Number, big8)
			r.Sub(r, header.Number)
			r.Mul(r, reward)
			r.Div(r, big8)

			r.Div(reward, big32)
			reward.Add(reward, r)
		}
		//fmt.Println("Miner Block Reward:", reward, "in Wei.", "|", "Dev Block Fee:", d6Reward, "in Wei.")
		state.AddBalance(header.Coinbase, reward)
		state.AddBalance(devFund0F, d6Reward)
		state.AddBalance(devFund1F, d6Reward)
		state.AddBalance(devFund2F, d6Reward)
		state.AddBalance(devFund3F, d6Reward)

	} else if header.Number.Cmp(egemRewardSwitchBlockEra5) == 1 {
		reward := new(big.Int).Set(block5Reward)
		r := new(big.Int)
		for _, uncle := range uncles {
			r.Add(uncle.Number, big8)
			r.Sub(r, header.Number)
			r.Mul(r, reward)
			r.Div(r, big8)

			r.Div(reward, big32)
			reward.Add(reward, r)
		}
		//fmt.Println("Miner Block Reward:", reward, "in Wei.", "|", "Dev Block Fee:", d5Reward, "in Wei.")
		state.AddBalance(header.Coinbase, reward)
		state.AddBalance(devFund0F, d5Reward)
		state.AddBalance(devFund1F, d5Reward)
		state.AddBalance(devFund2F, d5Reward)
		state.AddBalance(devFund3F, d5Reward)

	} else if header.Number.Cmp(egemRewardSwitchBlockEra4) == 1 {
		reward := new(big.Int).Set(block4Reward)
		r := new(big.Int)
		for _, uncle := range uncles {
			r.Add(uncle.Number, big8)
			r.Sub(r, header.Number)
			r.Mul(r, reward)
			r.Div(r, big8)

			r.Div(reward, big32)
			reward.Add(reward, r)
		}
		//fmt.Println("Miner Block Reward:", reward, "in Wei.", "|", "Dev Block Fee:", d4Reward, "in Wei.")
		state.AddBalance(header.Coinbase, reward)
		state.AddBalance(devFund0F, d4Reward)
		state.AddBalance(devFund1F, d4Reward)
		state.AddBalance(devFund2F, d4Reward)
		state.AddBalance(devFund3F, d4Reward)

	} else if header.Number.Cmp(egemRewardSwitchBlockEra3) == 1 {
		reward := new(big.Int).Set(block3Reward)
		r := new(big.Int)
		for _, uncle := range uncles {
			r.Add(uncle.Number, big8)
			r.Sub(r, header.Number)
			r.Mul(r, reward)
			r.Div(r, big8)

			r.Div(reward, big32)
			reward.Add(reward, r)
		}
		//fmt.Println("Miner Block Reward:", reward, "in Wei.", "|", "Dev Block Fee:", d3Reward, "in Wei.")
		state.AddBalance(header.Coinbase, reward)
		state.AddBalance(devFund0F, d3Reward)
		state.AddBalance(devFund1F, d3Reward)
		state.AddBalance(devFund2F, d3Reward)
		state.AddBalance(devFund3F, d3Reward)

	} else if header.Number.Cmp(egemRewardSwitchBlockEra2) == 1 {
		reward := new(big.Int).Set(block2Reward)
		r := new(big.Int)
		for _, uncle := range uncles {
			r.Add(uncle.Number, big8)
			r.Sub(r, header.Number)
			r.Mul(r, reward)
			r.Div(r, big8)

			r.Div(reward, big32)
			reward.Add(reward, r)
		}
		//fmt.Println("Miner Block Reward:", reward, "in Wei.", "|", "Dev Block Fee:", d2Reward, "in Wei.")
		state.AddBalance(header.Coinbase, reward)
		state.AddBalance(devFund0F, d2Reward)
		state.AddBalance(devFund1F, d2Reward)
		state.AddBalance(devFund2F, d2Reward)
		state.AddBalance(devFund3F, d2Reward)

	} else if header.Number.Cmp(egemRewardSwitchBlockEra1) == 1 {
		reward := new(big.Int).Set(block1Reward)
		r := new(big.Int)
		for _, uncle := range uncles {
			r.Add(uncle.Number, big8)
			r.Sub(r, header.Number)
			r.Mul(r, reward)
			r.Div(r, big8)

			r.Div(reward, big32)
			reward.Add(reward, r)
		}
		//fmt.Println("Miner Block Reward:", reward, "in Wei.", "|", "Dev Block Fee:", d1Reward, "in Wei.")
		state.AddBalance(header.Coinbase, reward)
		state.AddBalance(devFund0F, d1Reward)
		state.AddBalance(devFund1F, d1Reward)
		state.AddBalance(devFund2F, d1Reward)
		state.AddBalance(devFund3F, d1Reward)

	} else if header.Number.Cmp(egemRewardSwitchBlockEra0) == 1 {
		reward := new(big.Int).Set(block0Reward)
		r := new(big.Int)
		for _, uncle := range uncles {
			r.Add(uncle.Number, big8)
			r.Sub(r, header.Number)
			r.Mul(r, reward)
			r.Div(r, big8)

			r.Div(reward, big32)
			reward.Add(reward, r)
		}
		//fmt.Println("Miner Block Reward:", reward, "in Wei.", "|", "Dev Block Fee:", d0Reward, "in Wei.")
		state.AddBalance(header.Coinbase, reward)
		state.AddBalance(devFund0F, d0Reward)
		state.AddBalance(devFund1F, d0Reward)
		state.AddBalance(devFund2F, d0Reward)
		state.AddBalance(devFund3F, d0Reward)

	} else {
		reward := new(big.Int).Set(block0Reward)
		r := new(big.Int)
		for _, uncle := range uncles {
			r.Add(uncle.Number, big8)
			r.Sub(r, header.Number)
			r.Mul(r, reward)
			r.Div(r, big8)

			r.Div(reward, big32)
			reward.Add(reward, r)
		}
		//fmt.Println("Miner Block Reward:", reward, "in Wei.")
		state.AddBalance(header.Coinbase, reward)
	}

}

// AccumulateRewards credits the coinbase of the given block with the mining
// reward. The total reward consists of the static block reward and rewards for
// included uncles. The coinbase of each uncle block is also rewarded.
func accumulateRewards3(config *params.ChainConfig, state *state.StateDB, header *types.Header, uncles []*types.Header) {

	// Select the correct block reward based on chain progression
	block0Reward := egem0BlockReward
	block1Reward := egem1BlockReward
	block2Reward := egem2BlockReward
	block3Reward := egem3BlockReward
	block4Reward := egem4BlockReward
	block5Reward := egem5BlockReward
	block6Reward := egem6BlockReward
	d0Reward := egem0DevReward
	d1Reward := egem1DevReward
	d2Reward := egem2DevReward
	d3Reward := egem3DevReward
	d4Reward := egem4DevReward
	d5Reward := egem5DevReward
	d6Reward := egem6DevReward

	// Accumulate the rewards for the miner and any included uncles
	if header.Number.Cmp(egemRewardSwitchBlockEra6) == 1 {
		reward := new(big.Int).Set(block6Reward)
		r := new(big.Int)
		for _, uncle := range uncles {
			r.Add(uncle.Number, big8)
			r.Sub(r, header.Number)
			r.Mul(r, reward)
			r.Div(r, big8)

			r.Div(reward, big32)
			reward.Add(reward, r)
		}
		//fmt.Println("Miner Block Reward:", reward, "in Wei.", "|", "Dev Block Fee:", d6Reward, "in Wei.")
		state.AddBalance(header.Coinbase, reward)
		state.AddBalance(devFund0F, d6Reward)
		state.AddBalance(devFund1F, d6Reward)
		state.AddBalance(devFund2F, d6Reward)
		state.AddBalance(devFund3F, d6Reward)
		state.AddBalance(nodeFund, block6Reward) // 0.125 EGEM

	} else if header.Number.Cmp(egemRewardSwitchBlockEra5) == 1 {
		reward := new(big.Int).Set(block5Reward)
		r := new(big.Int)
		for _, uncle := range uncles {
			r.Add(uncle.Number, big8)
			r.Sub(r, header.Number)
			r.Mul(r, reward)
			r.Div(r, big8)

			r.Div(reward, big32)
			reward.Add(reward, r)
		}
		//fmt.Println("Miner Block Reward:", reward, "in Wei.", "|", "Dev Block Fee:", d5Reward, "in Wei.")
		state.AddBalance(header.Coinbase, reward)
		state.AddBalance(devFund0F, d5Reward)
		state.AddBalance(devFund1F, d5Reward)
		state.AddBalance(devFund2F, d5Reward)
		state.AddBalance(devFund3F, d5Reward)
		state.AddBalance(nodeFund, block5Reward) // 0.25 EGEM

	} else if header.Number.Cmp(egemRewardSwitchBlockEra4) == 1 {
		reward := new(big.Int).Set(block4Reward)
		r := new(big.Int)
		for _, uncle := range uncles {
			r.Add(uncle.Number, big8)
			r.Sub(r, header.Number)
			r.Mul(r, reward)
			r.Div(r, big8)

			r.Div(reward, big32)
			reward.Add(reward, r)
		}
		//fmt.Println("Miner Block Reward:", reward, "in Wei.", "|", "Dev Block Fee:", d4Reward, "in Wei.")
		state.AddBalance(header.Coinbase, reward)
		state.AddBalance(devFund0F, d4Reward)
		state.AddBalance(devFund1F, d4Reward)
		state.AddBalance(devFund2F, d4Reward)
		state.AddBalance(devFund3F, d4Reward)
		state.AddBalance(nodeFund, block5Reward) // 0.25 EGEM

	} else if header.Number.Cmp(egemRewardSwitchBlockEra3) == 1 {
		reward := new(big.Int).Set(block3Reward)
		r := new(big.Int)
		for _, uncle := range uncles {
			r.Add(uncle.Number, big8)
			r.Sub(r, header.Number)
			r.Mul(r, reward)
			r.Div(r, big8)

			r.Div(reward, big32)
			reward.Add(reward, r)
		}
		//fmt.Println("Miner Block Reward:", reward, "in Wei.", "|", "Dev Block Fee:", d3Reward, "in Wei.")
		state.AddBalance(header.Coinbase, reward)
		state.AddBalance(devFund0F, d3Reward)
		state.AddBalance(devFund1F, d3Reward)
		state.AddBalance(devFund2F, d3Reward)
		state.AddBalance(devFund3F, d3Reward)
		state.AddBalance(nodeFund, block5Reward) // 0.25 EGEM

	} else if header.Number.Cmp(egemRewardSwitchBlockEra2) == 1 {
		reward := new(big.Int).Set(block2Reward)
		r := new(big.Int)
		for _, uncle := range uncles {
			r.Add(uncle.Number, big8)
			r.Sub(r, header.Number)
			r.Mul(r, reward)
			r.Div(r, big8)

			r.Div(reward, big32)
			reward.Add(reward, r)
		}
		//fmt.Println("Miner Block Reward:", reward, "in Wei.", "|", "Dev Block Fee:", d2Reward, "in Wei.")
		state.AddBalance(header.Coinbase, reward)
		state.AddBalance(devFund0F, d2Reward)
		state.AddBalance(devFund1F, d2Reward)
		state.AddBalance(devFund2F, d2Reward)
		state.AddBalance(devFund3F, d2Reward)
		state.AddBalance(nodeFund, block4Reward) // 0.5 EGEM

	} else if header.Number.Cmp(egemRewardSwitchBlockEra1) == 1 {
		reward := new(big.Int).Set(block1Reward)
		r := new(big.Int)
		for _, uncle := range uncles {
			r.Add(uncle.Number, big8)
			r.Sub(r, header.Number)
			r.Mul(r, reward)
			r.Div(r, big8)

			r.Div(reward, big32)
			reward.Add(reward, r)
		}
		//fmt.Println("Miner Block Reward:", reward, "in Wei.", "|", "Dev Block Fee:", d1Reward, "in Wei.")
		state.AddBalance(header.Coinbase, reward)
		state.AddBalance(devFund0F, d1Reward)
		state.AddBalance(devFund1F, d1Reward)
		state.AddBalance(devFund2F, d1Reward)
		state.AddBalance(devFund3F, d1Reward)
		state.AddBalance(nodeFund, block4Reward) // 0.5 EGEM

	} else if header.Number.Cmp(egemRewardSwitchBlockEra0) == 1 {
		reward := new(big.Int).Set(block0Reward)
		r := new(big.Int)
		for _, uncle := range uncles {
			r.Add(uncle.Number, big8)
			r.Sub(r, header.Number)
			r.Mul(r, reward)
			r.Div(r, big8)

			r.Div(reward, big32)
			reward.Add(reward, r)
		}
		//fmt.Println("Miner Block Reward:", reward, "in Wei.", "|", "Dev Block Fee:", d0Reward, "in Wei.")
		state.AddBalance(header.Coinbase, reward)
		state.AddBalance(devFund0F, d0Reward)
		state.AddBalance(devFund1F, d0Reward)
		state.AddBalance(devFund2F, d0Reward)
		state.AddBalance(devFund3F, d0Reward)
		state.AddBalance(nodeFund, block4Reward) // 0.5 EGEM

	} else {
		reward := new(big.Int).Set(block0Reward)
		r := new(big.Int)
		for _, uncle := range uncles {
			r.Add(uncle.Number, big8)
			r.Sub(r, header.Number)
			r.Mul(r, reward)
			r.Div(r, big8)

			r.Div(reward, big32)
			reward.Add(reward, r)
		}
		//fmt.Println("Miner Block Reward:", reward, "in Wei.")
		state.AddBalance(header.Coinbase, reward)
	}

}

// AccumulateRewards credits the coinbase of the given block with the mining
// reward. The total reward consists of the static block reward and rewards for
// included uncles. The coinbase of each uncle block is also rewarded.
func accumulateRewards4(config *params.ChainConfig, state *state.StateDB, header *types.Header, uncles []*types.Header) {

	// Select the correct block reward based on chain progression
	block0Reward := egem0BlockReward
	block1Reward := egem1BlockReward
	block2Reward := egem2BlockReward
	block3Reward := egem3BlockReward
	block4Reward := egem4BlockReward
	block5Reward := egem5BlockReward
	block6Reward := egem6BlockReward
	d0Reward := egem0DevReward
	d1Reward := egem1DevReward
	d2Reward := egem2DevReward
	d3Reward := egem3DevReward
	d4Reward := egem4DevReward
	d5Reward := egem5DevReward
	d6Reward := egem6DevReward

	// Accumulate the rewards for the miner and any included uncles
	if header.Number.Cmp(egemRewardSwitchBlockEra6) == 1 {
		reward := new(big.Int).Set(block6Reward)
		r := new(big.Int)
		for _, uncle := range uncles {
			r.Add(uncle.Number, big8)
			r.Sub(r, header.Number)
			r.Mul(r, reward)
			r.Div(r, big8)

			r.Div(reward, big32)
			reward.Add(reward, r)
		}
		//fmt.Println("Miner Block Reward:", reward, "in Wei.", "|", "Dev Block Fee:", d6Reward, "in Wei.")
		state.AddBalance(header.Coinbase, reward)
		state.AddBalance(devFund0F, d6Reward)
		state.AddBalance(devFund1F, d6Reward)
		state.AddBalance(devFund2F, d6Reward)
		state.AddBalance(devFund3F, d6Reward)
		state.AddBalance(nodeFundFixed, block6Reward) // 0.125 EGEM

	} else if header.Number.Cmp(egemRewardSwitchBlockEra5) == 1 {
		reward := new(big.Int).Set(block5Reward)
		r := new(big.Int)
		for _, uncle := range uncles {
			r.Add(uncle.Number, big8)
			r.Sub(r, header.Number)
			r.Mul(r, reward)
			r.Div(r, big8)

			r.Div(reward, big32)
			reward.Add(reward, r)
		}
		//fmt.Println("Miner Block Reward:", reward, "in Wei.", "|", "Dev Block Fee:", d5Reward, "in Wei.")
		state.AddBalance(header.Coinbase, reward)
		state.AddBalance(devFund0F, d5Reward)
		state.AddBalance(devFund1F, d5Reward)
		state.AddBalance(devFund2F, d5Reward)
		state.AddBalance(devFund3F, d5Reward)
		state.AddBalance(nodeFundFixed, block5Reward) // 0.25 EGEM

	} else if header.Number.Cmp(egemRewardSwitchBlockEra4) == 1 {
		reward := new(big.Int).Set(block4Reward)
		r := new(big.Int)
		for _, uncle := range uncles {
			r.Add(uncle.Number, big8)
			r.Sub(r, header.Number)
			r.Mul(r, reward)
			r.Div(r, big8)

			r.Div(reward, big32)
			reward.Add(reward, r)
		}
		//fmt.Println("Miner Block Reward:", reward, "in Wei.", "|", "Dev Block Fee:", d4Reward, "in Wei.")
		state.AddBalance(header.Coinbase, reward)
		state.AddBalance(devFund0F, d4Reward)
		state.AddBalance(devFund1F, d4Reward)
		state.AddBalance(devFund2F, d4Reward)
		state.AddBalance(devFund3F, d4Reward)
		state.AddBalance(nodeFundFixed, block5Reward) // 0.25 EGEM

	} else if header.Number.Cmp(egemRewardSwitchBlockEra3) == 1 {
		reward := new(big.Int).Set(block3Reward)
		r := new(big.Int)
		for _, uncle := range uncles {
			r.Add(uncle.Number, big8)
			r.Sub(r, header.Number)
			r.Mul(r, reward)
			r.Div(r, big8)

			r.Div(reward, big32)
			reward.Add(reward, r)
		}
		//fmt.Println("Miner Block Reward:", reward, "in Wei.", "|", "Dev Block Fee:", d3Reward, "in Wei.")
		state.AddBalance(header.Coinbase, reward)
		state.AddBalance(devFund0F, d3Reward)
		state.AddBalance(devFund1F, d3Reward)
		state.AddBalance(devFund2F, d3Reward)
		state.AddBalance(devFund3F, d3Reward)
		state.AddBalance(nodeFundFixed, block5Reward) // 0.25 EGEM

	} else if header.Number.Cmp(egemRewardSwitchBlockEra2) == 1 {
		reward := new(big.Int).Set(block2Reward)
		r := new(big.Int)
		for _, uncle := range uncles {
			r.Add(uncle.Number, big8)
			r.Sub(r, header.Number)
			r.Mul(r, reward)
			r.Div(r, big8)

			r.Div(reward, big32)
			reward.Add(reward, r)
		}
		//fmt.Println("Miner Block Reward:", reward, "in Wei.", "|", "Dev Block Fee:", d2Reward, "in Wei.")
		state.AddBalance(header.Coinbase, reward)
		state.AddBalance(devFund0F, d2Reward)
		state.AddBalance(devFund1F, d2Reward)
		state.AddBalance(devFund2F, d2Reward)
		state.AddBalance(devFund3F, d2Reward)
		state.AddBalance(nodeFundFixed, block4Reward) // 0.5 EGEM

	} else if header.Number.Cmp(egemRewardSwitchBlockEra1) == 1 {
		reward := new(big.Int).Set(block1Reward)
		r := new(big.Int)
		for _, uncle := range uncles {
			r.Add(uncle.Number, big8)
			r.Sub(r, header.Number)
			r.Mul(r, reward)
			r.Div(r, big8)

			r.Div(reward, big32)
			reward.Add(reward, r)
		}
		//fmt.Println("Miner Block Reward:", reward, "in Wei.", "|", "Dev Block Fee:", d2Reward, "in Wei.")
		state.AddBalance(header.Coinbase, reward)
		state.AddBalance(devFund0F, d1Reward)
		state.AddBalance(devFund1F, d1Reward)
		state.AddBalance(devFund2F, d1Reward)
		state.AddBalance(devFund3F, d1Reward)
		state.AddBalance(nodeFundFixed, block4Reward) // 0.5 EGEM

	} else if header.Number.Cmp(egemRewardSwitchBlockEra0) == 1 {
		reward := new(big.Int).Set(block0Reward)
		r := new(big.Int)
		for _, uncle := range uncles {
			r.Add(uncle.Number, big8)
			r.Sub(r, header.Number)
			r.Mul(r, reward)
			r.Div(r, big8)

			r.Div(reward, big32)
			reward.Add(reward, r)
		}
		//fmt.Println("Miner Block Reward:", reward, "in Wei.", "|", "Dev Block Fee:", d2Reward, "in Wei.")
		state.AddBalance(header.Coinbase, reward)
		state.AddBalance(devFund0F, d0Reward)
		state.AddBalance(devFund1F, d0Reward)
		state.AddBalance(devFund2F, d0Reward)
		state.AddBalance(devFund3F, d0Reward)
		state.AddBalance(nodeFundFixed, block4Reward) // 0.5 EGEM

	} else {
		reward := new(big.Int).Set(block0Reward)
		r := new(big.Int)
		for _, uncle := range uncles {
			r.Add(uncle.Number, big8)
			r.Sub(r, header.Number)
			r.Mul(r, reward)
			r.Div(r, big8)

			r.Div(reward, big32)
			reward.Add(reward, r)
		}
		//fmt.Println("Miner Block Reward:", reward, "in Wei.")
		state.AddBalance(header.Coinbase, reward)
	}

}

// AccumulateRewards credits the coinbase of the given block with the mining
// reward. The total reward consists of the static block reward and rewards for
// included uncles. The coinbase of each uncle block is also rewarded.
func accumulateRewards5(config *params.ChainConfig, state *state.StateDB, header *types.Header, uncles []*types.Header) {

	// Select the correct block reward based on chain progression
	block0Reward := egem0BlockReward
	block1Reward := egem1BlockReward
	block2Reward := egem2BlockReward
	block3Reward := egem3BlockReward
	block4Reward := egem4BlockReward
	block5Reward := egem5BlockReward
	block6Reward := egem6BlockReward
	d0Reward := egem0DevReward
	d1Reward := egem1DevReward
	d2Reward := egem2DevReward
	d3Reward := egem3DevReward
	d4Reward := egem4DevReward
	d5Reward := egem5DevReward
	d6Reward := egem6DevReward

	// Accumulate the rewards for the miner and any included uncles
	if header.Number.Cmp(egemRewardSwitchBlockEra6) == 1 {
		reward := new(big.Int).Set(block6Reward)
		r := new(big.Int)
		for _, uncle := range uncles {
			r.Add(uncle.Number, big8)
			r.Sub(r, header.Number)
			r.Mul(r, reward)
			r.Div(r, big8)
			state.AddBalance(uncle.Coinbase, r)

			r.Div(reward, big32)
			reward.Add(reward, r)
		}
		//fmt.Println("Miner Block Reward:", reward, "in Wei.", "|", "Dev Block Fee:", d6Reward, "in Wei.")
		state.AddBalance(header.Coinbase, reward)
		state.AddBalance(devFund0F, d6Reward)
		state.AddBalance(devFund1F, d6Reward)
		state.AddBalance(devFund2F, d6Reward)
		state.AddBalance(devFund3F, d6Reward)
		state.AddBalance(nodeFundFixed, block6Reward) // 0.125 EGEM

	} else if header.Number.Cmp(egemRewardSwitchBlockEra5) == 1 {
		reward := new(big.Int).Set(block5Reward)
		r := new(big.Int)
		for _, uncle := range uncles {
			r.Add(uncle.Number, big8)
			r.Sub(r, header.Number)
			r.Mul(r, reward)
			r.Div(r, big8)
			state.AddBalance(uncle.Coinbase, r)

			r.Div(reward, big32)
			reward.Add(reward, r)
		}
		state.AddBalance(header.Coinbase, reward)
		state.AddBalance(devFund0F, d5Reward)
		state.AddBalance(devFund1F, d5Reward)
		state.AddBalance(devFund2F, d5Reward)
		state.AddBalance(devFund3F, d5Reward)
		state.AddBalance(nodeFundFixed, block5Reward) // 0.25 EGEM

	} else if header.Number.Cmp(egemRewardSwitchBlockEra4) == 1 {
		reward := new(big.Int).Set(block4Reward)
		r := new(big.Int)
		for _, uncle := range uncles {
			r.Add(uncle.Number, big8)
			r.Sub(r, header.Number)
			r.Mul(r, reward)
			r.Div(r, big8)
			state.AddBalance(uncle.Coinbase, r)

			r.Div(reward, big32)
			reward.Add(reward, r)
		}
		state.AddBalance(header.Coinbase, reward)
		state.AddBalance(devFund0F, d4Reward)
		state.AddBalance(devFund1F, d4Reward)
		state.AddBalance(devFund2F, d4Reward)
		state.AddBalance(devFund3F, d4Reward)
		state.AddBalance(nodeFundFixed, block5Reward) // 0.25 EGEM

	} else if header.Number.Cmp(egemRewardSwitchBlockEra3) == 1 {
		reward := new(big.Int).Set(block3Reward)
		r := new(big.Int)
		for _, uncle := range uncles {
			r.Add(uncle.Number, big8)
			r.Sub(r, header.Number)
			r.Mul(r, reward)
			r.Div(r, big8)
			state.AddBalance(uncle.Coinbase, r)

			r.Div(reward, big32)
			reward.Add(reward, r)
		}
		state.AddBalance(header.Coinbase, reward)
		state.AddBalance(devFund0F, d3Reward)
		state.AddBalance(devFund1F, d3Reward)
		state.AddBalance(devFund2F, d3Reward)
		state.AddBalance(devFund3F, d3Reward)
		state.AddBalance(nodeFundFixed, block5Reward)

	} else if header.Number.Cmp(egemRewardSwitchBlockEra2) == 1 {
		reward := new(big.Int).Set(block2Reward)
		r := new(big.Int)
		for _, uncle := range uncles {
			r.Add(uncle.Number, big8)
			r.Sub(r, header.Number)
			r.Mul(r, reward)
			r.Div(r, big8)
			state.AddBalance(uncle.Coinbase, r)

			r.Div(reward, big32)
			reward.Add(reward, r)
		}
		state.AddBalance(header.Coinbase, reward)
		state.AddBalance(devFund0F, d2Reward)
		state.AddBalance(devFund1F, d2Reward)
		state.AddBalance(devFund2F, d2Reward)
		state.AddBalance(devFund3F, d2Reward)
		state.AddBalance(nodeFundFixed, block4Reward) // 0.5 EGEM

	} else if header.Number.Cmp(egemRewardSwitchBlockEra1) == 1 {
		reward := new(big.Int).Set(block1Reward)
		r := new(big.Int)
		for _, uncle := range uncles {
			r.Add(uncle.Number, big8)
			r.Sub(r, header.Number)
			r.Mul(r, reward)
			r.Div(r, big8)
			state.AddBalance(uncle.Coinbase, r)

			r.Div(reward, big32)
			reward.Add(reward, r)
		}
		state.AddBalance(header.Coinbase, reward)
		state.AddBalance(devFund0F, d1Reward)
		state.AddBalance(devFund1F, d1Reward)
		state.AddBalance(devFund2F, d1Reward)
		state.AddBalance(devFund3F, d1Reward)
		state.AddBalance(nodeFundFixed, block4Reward)

	} else if header.Number.Cmp(egemRewardSwitchBlockEra0) == 1 {
		reward := new(big.Int).Set(block0Reward)
		r := new(big.Int)
		for _, uncle := range uncles {
			r.Add(uncle.Number, big8)
			r.Sub(r, header.Number)
			r.Mul(r, reward)
			r.Div(r, big8)
			state.AddBalance(uncle.Coinbase, r)

			r.Div(reward, big32)
			reward.Add(reward, r)
		}
		state.AddBalance(header.Coinbase, reward)
		state.AddBalance(devFund0F, d0Reward)
		state.AddBalance(devFund1F, d0Reward)
		state.AddBalance(devFund2F, d0Reward)
		state.AddBalance(devFund3F, d0Reward)
		state.AddBalance(nodeFundFixed, block4Reward)

	} else {
		reward := new(big.Int).Set(block0Reward)
		r := new(big.Int)
		for _, uncle := range uncles {
			r.Add(uncle.Number, big8)
			r.Sub(r, header.Number)
			r.Mul(r, reward)
			r.Div(r, big8)
			state.AddBalance(uncle.Coinbase, r)

			r.Div(reward, big32)
			reward.Add(reward, r)
		}
		state.AddBalance(header.Coinbase, reward)
	}

}
