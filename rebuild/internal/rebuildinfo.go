package internal

import (
	"fmt"
	"os"
	"rebuild/internal/randomness"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/venus/venus-shared/actors/policy"
	"github.com/filecoin-project/venus/venus-shared/types"
	mtypes "github.com/filecoin-project/venus/venus-shared/types/market"
	"github.com/ipfs/go-cid"
	"github.com/urfave/cli/v2"
)

type SectorPublicInfo struct {
	CommR      [32]byte
	SealedCID  cid.Cid
	Activation abi.ChainEpoch
	Expiration abi.ChainEpoch
}

type DealInfoV2 struct {
	*mtypes.DealInfoV2
	// if true, it indicates that the deal is an legacy builtin market deal,
	// otherwise it is a DDO deal
	IsBuiltinMarket bool
	// this is the flag for pieces from original implementations
	// if true, workers should use the piece data directly, instead of padding themselves
	IsCompatible bool
}

func (di DealInfoV2) DisplayID() string {
	if di.IsBuiltinMarket {
		return fmt.Sprintf("builtinmarket(%d)", di.DealID)
	}
	return fmt.Sprintf("ddo(%d)", di.AllocationID)
}

type SectorPieceV2 struct {
	Piece    abi.PieceInfo
	DealInfo *DealInfoV2
}

type SectorPieces []SectorPieceV2

type RebuildInfo struct {
	IsSnapUp     bool
	SnapUpPublic SectorPublicInfo
	Ticket       randomness.Ticket
	Seed         randomness.Seed
	Pieces       SectorPieces `json:"Pieces"`
}

var RebuildInfoCmd = &cli.Command{
	Name:  "info",
	Usage: "Show rebuild info",
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:     "height",
			Required: false,
		},
	},
	ArgsUsage: "<miner actor id> <sector number>",
	Action: func(cctx *cli.Context) error {
		if count := cctx.Args().Len(); count < 2 {
			return cli.ShowSubcommandHelp(cctx)
		}

		maddr, err := ShouldAddress(cctx.Args().Get(0), true, true)
		if err != nil {
			return fmt.Errorf("invalid miner actor id: %w", err)
		}

		mid, err := ShouldActor(cctx.Args().Get(0), true)
		if err != nil {
			return fmt.Errorf("invalid miner actor id: %w", err)
		}

		sectorNum, err := ShouldSectorNumber(cctx.Args().Get(1))
		if err != nil {
			return err
		}

		chain, closer, err := NewChainAPI(cctx.Context, cctx.String("chain"), cctx.String("token"))
		if err != nil {
			return err
		}
		defer closer()

		height := cctx.Int("height")
		var tsk types.TipSetKey
		if height == 0 {
			tsk = types.EmptyTSK
		} else {
			ts, err := chain.ChainGetTipSetByHeight(cctx.Context, abi.ChainEpoch(height), types.EmptyTSK)
			if err != nil {
				return RPCCallError("ChainGetTipSetByHeight", err)
			}
			tsk = ts.Key()
		}

		si, err := chain.StateSectorGetInfo(cctx.Context, maddr, sectorNum, types.EmptyTSK)
		if err != nil {
			return RPCCallError("StateSectorGetInfo", err)
		}

		pci, err := chain.StateSectorPreCommitInfo(cctx.Context, maddr, sectorNum, tsk)
		if err != nil {
			return RPCCallError("StateSectorPreCommitInfo", err)
		}

		if pci == nil {
			return fmt.Errorf("cannot find pre commit info")
		}

		randAPI, err := randomness.New(chain)
		if err != nil {
			return err
		}

		ticket, err := randAPI.GetTicket(cctx.Context, tsk, pci.Info.SealRandEpoch, mid)
		if err != nil {
			return RPCCallError("GetTicket", err)
		}

		seedEpoch := pci.PreCommitEpoch + policy.GetPreCommitChallengeDelay()
		seed, err := randAPI.GetSeed(cctx.Context, tsk, seedEpoch, mid)
		if err != nil {
			return RPCCallError("GetSeed", err)
		}

		commR, err := CID2ReplicaCommitment(si.SealedCID)
		if err != nil {
			return fmt.Errorf("invalid sealed cid %s: %w", si.SealedCID, err)
		}

		public := SectorPublicInfo{
			CommR:      commR,
			SealedCID:  si.SealedCID,
			Activation: si.Activation,
			Expiration: si.Expiration,
		}

		pieces := make(SectorPieces, 0, len(si.DealIDs))
		var offset abi.PaddedPieceSize
		for _, dealID := range si.DealIDs {
			deal, err := chain.StateMarketStorageDeal(cctx.Context, dealID, types.EmptyTSK)
			if err != nil {
				return fmt.Errorf("invalid sealed cid %s: %w", si.SealedCID, err)
			}

			pieces = append(pieces, SectorPieceV2{
				Piece: abi.PieceInfo{
					Size:     deal.Proposal.PieceSize,
					PieceCID: deal.Proposal.PieceCID,
				},
				DealInfo: &DealInfoV2{
					DealInfoV2: &mtypes.DealInfoV2{
						DealID:       dealID,
						PublishCid:   cid.Undef,
						AllocationID: 0,
						PieceCID:     deal.Proposal.PieceCID,
						PieceSize:    deal.Proposal.PieceSize,
						Client:       deal.Proposal.Client,
						Provider:     deal.Proposal.Provider,
						Offset:       offset,
						Length:       deal.Proposal.PieceSize,
						PayloadSize:  0,
						StartEpoch:   deal.Proposal.StartEpoch,
						EndEpoch:     deal.Proposal.EndEpoch,
					},
					IsBuiltinMarket: false,
					IsCompatible:    true,
				},
			})
			offset += deal.Proposal.PieceSize
		}
		ri := RebuildInfo{
			IsSnapUp:     si.SectorKeyCID != nil,
			SnapUpPublic: public,
			Ticket:       ticket,
			Seed:         seed,
			Pieces:       pieces,
		}
		if err := OutputJSON(os.Stdout, ri); err != nil {
			return fmt.Errorf("output json: %w", err)
		}

		return nil
	},
}
