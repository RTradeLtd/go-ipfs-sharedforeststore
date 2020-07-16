package sharedforeststore

import (
	"context"
	"fmt"
	"testing"

	"github.com/gogo/protobuf/proto"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	ipld "github.com/ipfs/go-ipld-format"
	"github.com/ipfs/go-merkledag"
	pb "github.com/ipfs/go-unixfs/pb"
	"github.com/pkg/errors"
)

//setup creates a graph of nodes with the following structure:
//
//    A  B  C
//    \ / \ /
//     D   E
//      \ //
//       F
//
//Where A links to D, B has a diamond dependency on F, and E has a repeated link to F
func setup(t testing.TB) ([]cid.Cid, BlockGetter) {
	testString := "Hello World!"
	f, err := merkledag.NewRawNodeWPrefix([]byte(testString), cidBuilder)
	fatalIfErr(t, err)
	d, err := createFile(f)
	fatalIfErr(t, err)
	e, err := createFile(f, f)
	fatalIfErr(t, err)
	a, err := createFile(d)
	fatalIfErr(t, err)
	b, err := createFile(d, e)
	fatalIfErr(t, err)
	c, err := createFile(e)
	fatalIfErr(t, err)

	bs := []blocks.Block{a, b, c, d, e, f}
	return cidsFromBlocks(bs...), blockGetterFromBlocks(bs...)
}

var cidBuilder = merkledag.V1CidPrefix()

//createFile creates a file by linking the nodes together
func createFile(nodes ...ipld.Node) (*merkledag.ProtoNode, error) {

	size := uint64(0)
	links := make([]*ipld.Link, 0, len(nodes))
	blocks := make([]uint64, 0, len(nodes))

	for i, node := range nodes {
		link, err := ipld.MakeLink(node)
		if err != nil {
			return nil, err
		}
		links = append(links, link)
		fileSize, err := getFileSize(node)
		if err != nil {
			return nil, fmt.Errorf("node %v, %v", i, err)
		}
		size += fileSize
		blocks = append(blocks, link.Size)
	}

	protoNode := &merkledag.ProtoNode{}
	protoNode.SetCidBuilder(cidBuilder)
	protoNode.SetLinks(links)

	data, err := proto.Marshal(&pb.Data{
		Type:       pb.Data_File.Enum(),
		Filesize:   &size,
		Blocksizes: blocks,
	})
	if err != nil {
		return nil, err
	}
	protoNode.SetData(data)

	return protoNode, nil
}

// getFileSize returns the size of the file represented by the given node.
// Returns error if node's format is not supported.
func getFileSize(n ipld.Node) (uint64, error) {
	switch n := n.(type) {
	case *merkledag.RawNode:
		return uint64(len(n.RawData())), nil
	case *merkledag.ProtoNode:
		meta := &pb.Data{}
		if err := proto.Unmarshal(n.Data(), meta); err != nil {
			return 0, err
		}
		switch meta.GetType() {
		case pb.Data_File, pb.Data_Raw:
			return meta.GetFilesize(), nil
		default:
			return 0, fmt.Errorf("unsupported data type %v", meta.GetType().String())
		}
	default:
		return 0, errors.New("unknow node type")
	}
}

type mapBlockGetter map[cid.Cid][]byte

func (m mapBlockGetter) GetBlock(ctx context.Context, id cid.Cid) ([]byte, error) {
	block, has := m[id]
	if !has {
		return nil, errors.Errorf("CID: %v not found", id)
	}
	return block, nil
}

func blockGetterFromBlocks(bs ...blocks.Block) BlockGetter {
	m := make(mapBlockGetter)
	for _, b := range bs {
		m[b.Cid()] = b.RawData()
	}
	return m
}

func cidsFromBlocks(bs ...blocks.Block) []cid.Cid {
	cids := make([]cid.Cid, len(bs))
	for i, b := range bs {
		cids[i] = b.Cid()
	}
	return cids
}

func fatalIfErr(t testing.TB, err error, args ...interface{}) {
	t.Helper()
	if err != nil {
		t.Fatal(append([]interface{}{err}, args...)...)
	}
}
