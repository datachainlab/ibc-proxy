package types

import (
	ics23 "github.com/confio/ics23/go"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/ibc-go/modules/core/exported"
)

var _ exported.Proof = (*MultiProof)(nil)

func (p *MultiProof) VerifyMembership(_ []*ics23.ProofSpec, _ exported.Root, _ exported.Path, _ []byte) error {
	panic("not implemented") // TODO: Implement
}

func (p *MultiProof) VerifyNonMembership(_ []*ics23.ProofSpec, _ exported.Root, _ exported.Path) error {
	panic("not implemented") // TODO: Implement
}

func (p *MultiProof) Empty() bool {
	return false
}

func (p *MultiProof) ValidateBasic() error {
	return nil
}

func UnmarshalProof(cdc codec.BinaryCodec, bz []byte) (exported.Proof, error) {
	var proof exported.Proof
	if err := cdc.UnmarshalInterface(bz, &proof); err != nil {
		return nil, err
	}
	return proof, nil
}
