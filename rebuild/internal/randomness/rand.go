package randomness

import (
	"bytes"
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/crypto"

	v1 "github.com/filecoin-project/venus/venus-shared/api/chain/v1"
	"github.com/filecoin-project/venus/venus-shared/types"
)

type Ticket struct {
	Ticket abi.Randomness
	Epoch  abi.ChainEpoch
}

type Seed struct {
	Seed  abi.Randomness
	Epoch abi.ChainEpoch
}

func New(capi v1.FullNode) (*Randomness, error) {
	return &Randomness{
		api: capi,
	}, nil
}

type Randomness struct {
	api v1.FullNode
}

func (r *Randomness) getRandomnessEntropy(mid abi.ActorID) ([]byte, error) {
	maddr, err := address.NewIDAddress(uint64(mid))
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := maddr.MarshalCBOR(&buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (r *Randomness) GetTicket(ctx context.Context, tsk types.TipSetKey, epoch abi.ChainEpoch, mid abi.ActorID) (Ticket, error) {
	entropy, err := r.getRandomnessEntropy(mid)
	if err != nil {
		return Ticket{}, err
	}

	if tsk == types.EmptyTSK {
		ts, err := r.api.ChainHead(ctx)
		if err != nil {
			return Ticket{}, err
		}

		tsk = ts.Key()
	}

	rand, err := r.api.StateGetRandomnessFromTickets(ctx, crypto.DomainSeparationTag_SealRandomness, epoch, entropy, tsk)
	if err != nil {
		return Ticket{}, err
	}

	return Ticket{
		Ticket: rand,
		Epoch:  epoch,
	}, nil
}

func (r *Randomness) GetSeed(ctx context.Context, tsk types.TipSetKey, epoch abi.ChainEpoch, mid abi.ActorID) (Seed, error) {
	entropy, err := r.getRandomnessEntropy(mid)
	if err != nil {
		return Seed{}, err
	}

	if tsk == types.EmptyTSK {
		ts, err := r.api.ChainHead(ctx)
		if err != nil {
			return Seed{}, err
		}

		tsk = ts.Key()
	}

	rand, err := r.api.StateGetRandomnessFromBeacon(ctx, crypto.DomainSeparationTag_InteractiveSealChallengeSeed, epoch, entropy, tsk)
	if err != nil {
		return Seed{}, err
	}

	return Seed{
		Seed:  rand,
		Epoch: epoch,
	}, nil
}
