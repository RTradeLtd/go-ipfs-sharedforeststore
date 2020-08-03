// Copyright 2020 RTrade Technologies Ltd
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sharedforeststore

import (
	"fmt"

	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	ipld "github.com/ipfs/go-ipld-format"
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
//Total size returned is zero if not decodable.
func LinkDecoder(id cid.Cid, data []byte) ([]cid.Cid, uint64, error) {
	switch id.Prefix().GetCodec() {
	default:
		b, err := blocks.NewBlockWithCid(data, id)
		if err != nil {
			return nil, 0, err
		}
		node, err := ipld.DefaultBlockDecoder.Decode(b)
		if err != nil {
			return nil, 0, err
		}
		ls := node.Links()
		out := make([]cid.Cid, len(ls))
		for i, l := range ls {
			if l == nil {
				return nil, 0, errors.Errorf("block %v contains empty links %v", id, ls)
			}
			out[i] = l.Cid
		}
		size, _ := node.Size()
		return out, size, nil
	case cid.Raw:
		return nil, uint64(len(data)), nil
	case cid.DagProtobuf:
		pnode, err := merkledag.DecodeProtobuf(data)
		if err != nil {
			return nil, 0, errors.WithStack(err)
		}
		ls := pnode.Links()
		out := make([]cid.Cid, len(ls))
		for i, l := range ls {
			if l == nil {
				return nil, 0, errors.Errorf("block %v contains empty links %v", id, ls)
			}
			out[i] = l.Cid
		}
		size, _ := pnode.Size()
		return out, size, nil
	}
}
