package sharedforeststore

import (
	"fmt"

	"github.com/ipfs/go-cid"
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
		return nil, errors.WithStack(&CodecNotSupportedError{Cid: id})
	case cid.Raw:
		return nil, nil
	case cid.DagProtobuf:
		pnode, err := merkledag.DecodeProtobuf(data)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		links := pnode.Links()
		out := make([]cid.Cid, len(links))
		for i, l := range links {
			if l == nil {
				return nil, errors.Errorf("block %v contains empty links %v", id, links)
			}
			out[i] = l.Cid
		}
		return out, nil
	}
}
