package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	ipfs "github.com/ipfs/go-ipfs-api"
	files "github.com/whyrusleeping/go-multipart-files"
	rpc "github.com/whyrusleeping/zmsg/rpc"
)

func fatal(i interface{}) {
	fmt.Println(i)
	os.Exit(1)
}

func getBlkParent(blk string) (string, error) {
	req := &rpc.Request{
		Method: "getblock",
		Params: []string{blk},
	}

	var resp rpc.Response
	if err := rpc.Do(req, &resp); err != nil {
		return "", err
	}

	if resp.Error != nil {
		return "", resp.Error
	}

	res := resp.Result.(map[string]interface{})

	return res["previousblockhash"].(string), nil
}

func getBlkData(blk string) (string, error) {
	req := &rpc.Request{
		Method: "getblock",
		Params: []interface{}{blk, false},
	}

	var resp rpc.Response
	if err := rpc.Do(req, &resp); err != nil {
		return "", err
	}

	return resp.Result.(string), nil
}

func getBestBlock() (string, error) {
	req := &rpc.Request{
		Method: "getbestblockhash",
	}

	var out map[string]interface{}
	if err := rpc.Do(req, &out); err != nil {
		return "", err
	}

	return out["result"].(string), nil
}

func ipfsPutBlock(data string) (string, error) {
	req := ipfs.NewRequest("localhost:5001", "dag/put")
	req.Opts = map[string]string{
		"input-enc": "hex",
		"format":    "zcash",
	}

	r := strings.NewReader(data)
	rc := ioutil.NopCloser(r)
	fr := files.NewReaderFile("", "", rc, nil)
	slf := files.NewSliceFile("", "", []files.File{fr})
	fileReader := files.NewMultiFileReader(slf, true)
	req.Body = fileReader

	resp, err := req.Send(http.DefaultClient)
	if err != nil {
		return "", err
	}
	defer resp.Close()

	if resp.Error != nil {
		return "", resp.Error
	}

	var out map[string]string
	json.NewDecoder(resp.Output).Decode(&out)
	return out["Cid"], nil

}

func main() {
	sh := ipfs.NewShell("localhost:5001")
	_ = sh

	var cur string
	if len(os.Args) > 1 {
		cur = os.Args[1]
	} else {
		bestBlk, err := getBestBlock()
		if err != nil {
			fatal(err)
		}
		cur = bestBlk
	}

	for {
		blkdata, err := getBlkData(cur)
		if err != nil {
			fatal(err)
		}

		ipfshash, err := ipfsPutBlock(blkdata)
		if err != nil {
			fatal(err)
		}
		fmt.Printf("%s = %s\n", cur, ipfshash)

		parent, err := getBlkParent(cur)
		if err != nil {
			fatal(err)
		}
		cur = parent
	}
}
