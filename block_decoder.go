package sharedforeststore

import (
	"fmt"

	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	ipld "github.com/ipfs/go-ipld-format/"
	"github.com/ipfs/go-merkledag"
	"github.com/pkg/errors"
)

type CodecNotSupportedError struct {
	Cid cid.Cid
}

func (e *CodecNotSupportedError) Error() string {
	return fmt.Sprintf("codec %v is not supported in CID %v", e.Cid.Prefix().GetCodec(), e.Cid)
}

//LinkDecoder is the default function for DatabaseOptions.LinkDecoder.
//It decodes the required links for some common codecs.
func LinkDecoder(id cid.Cid, data []byte) ([]cid.Cid, error) {
	switch id.Prefix().GetCodec() {
	default:
		b, err := blocks.NewBlockWithCid(data, id)
		if err != nil {
			return nil, err
		}
		node, err := ipld.DefaultBlockDecoder.Decode(b)
		if err != nil {
			return nil, err
		}
		ls := node.Links()
		out := make([]cid.Cid, len(ls))
		for i, l := range ls {
			if l == nil {
				return nil, errors.Errorf("block %v contains empty links %v", id, ls)
			}
			out[i] = l.Cid
		}
		return out, nil
	case cid.Raw:
		return nil, nil
	case cid.DagProtobuf:
		pnode, err := merkledag.DecodeProtobuf(data)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		ls := pnode.Links()
		out := make([]cid.Cid, len(ls))
		for i, l := range ls {
			if l == nil {
				return nil, errors.Errorf("block %v contains empty links %v", id, ls)
			}
			out[i] = l.Cid
		}
		return out, nil
	}
}
