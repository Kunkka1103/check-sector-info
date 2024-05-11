package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/filecoin-project/go-address"
	commcid "github.com/filecoin-project/go-fil-commcid"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"
)

func RPCCallError(method string, err error) error {
	return fmt.Errorf("rpc %s: %w", method, err)
}

var ErrEmptyAddressString = fmt.Errorf("empty address string")

func ShouldAddress(s string, checkEmpty bool, allowActor bool) (address.Address, error) {
	if checkEmpty && s == "" {
		return address.Undef, ErrEmptyAddressString
	}

	if allowActor {
		id, err := strconv.ParseUint(s, 10, 64)
		if err == nil {
			return address.NewIDAddress(id)
		}
	}

	return address.NewFromString(s)
}

func ShouldActor(s string, checkEmpty bool) (abi.ActorID, error) {
	addr, err := ShouldAddress(s, checkEmpty, true)
	if err != nil {
		return 0, err
	}

	actor, err := address.IDFromAddress(addr)
	if err != nil {
		return 0, fmt.Errorf("get actor id from addr: %w", err)
	}

	return abi.ActorID(actor), nil
}

func ShouldSectorNumber(s string) (abi.SectorNumber, error) {
	num, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse sector number: %w", err)
	}

	return abi.SectorNumber(num), nil
}

func OutputJSON(w io.Writer, v interface{}) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "\t")
	return enc.Encode(v)
}

func CID2ReplicaCommitment(sealedCID cid.Cid) ([32]byte, error) {
	var commR [32]byte

	if !sealedCID.Defined() {
		return commR, fmt.Errorf("undefined cid")
	}

	b, err := commcid.CIDToReplicaCommitmentV1(sealedCID)
	if err != nil {
		return commR, fmt.Errorf("convert to commitment: %w", err)
	}

	if size := len(b); size != 32 {
		return commR, fmt.Errorf("get %d bytes for commitment", size)
	}

	copy(commR[:], b[:])
	return commR, nil
}
