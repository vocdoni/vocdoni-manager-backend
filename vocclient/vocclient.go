package vocclient

import (
	"encoding/hex"
	"fmt"
	"math/rand"

	"go.vocdoni.io/dvote/api"
	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/log"
	"go.vocdoni.io/dvote/util"
	"go.vocdoni.io/dvote/vochain/scrutinizer/indexertypes"
	"go.vocdoni.io/manager/types"
	"go.vocdoni.io/proto/build/go/models"
	"google.golang.org/protobuf/proto"
)

type VocClient struct {
	pool       GatewayPool
	signingKey *ethereum.SignKeys
}

func New(gatewayUrls []string, signingKey *ethereum.SignKeys) (*VocClient, error) {

	gwPool, err := discoverGateways(gatewayUrls)
	if err != nil {
		return nil, err
	}
	return &VocClient{
		pool:       gwPool,
		signingKey: signingKey,
	}, nil
}

// FETCHING INFO APIS

func (c *VocClient) GetCurrentBlock() (blockHeight uint32, _ error) {
	var req api.MetaRequest
	req.Method = "getBlockHeight"
	resp, err := c.pool.Request(req, nil)
	if err != nil {
		return 0, err
	}
	if resp.Height == nil {
		return 0, fmt.Errorf("height is nil")
	}
	return *resp.Height, nil
}

func (c *VocClient) GetProcess(pid []byte) (*indexertypes.Process, error) {
	var req api.MetaRequest
	req.Method = "getProcessInfo"
	req.ProcessID = pid
	resp, err := c.pool.Request(req, c.signingKey)
	if err != nil {
		return nil, err
	}
	return resp.Process, nil
}

func (c *VocClient) GetProcessList(entityId []byte, searchTerm string, namespace uint32, status string, withResults bool, srcNetId string, from, listSize int) (processList []string, _ error) {
	var req api.MetaRequest
	req.Method = "getProcessList"
	req.EntityId = entityId
	req.SearchTerm = searchTerm
	req.Namespace = namespace
	req.SrcNetId = srcNetId
	req.Status = status
	req.WithResults = withResults
	req.From = from
	req.ListSize = listSize
	resp, err := c.pool.Request(req, c.signingKey)
	if err != nil {
		return nil, err
	}
	if resp.Message == "no results yet" {
		return nil, nil
	}
	return resp.ProcessList, nil
}

// FILE APIS

func (c *VocClient) AddFile(content []byte, contentType, name string) (URI string, _ error) {
	resp, err := c.pool.Request(api.MetaRequest{
		Method:  "addFile",
		Content: content,
		Type:    contentType,
		Name:    name,
	}, c.signingKey)
	if err != nil {
		return "", fmt.Errorf("could not AddFile %s: %v", name, err)
	}
	return resp.URI, nil
}

// CENSUS APIS

func (c *VocClient) AddCensus() (CensusID string, _ error) {
	var req api.MetaRequest

	// Create census
	log.Infof("Create census")
	req.Method = "addCensus"
	req.CensusType = models.Census_ARBO_BLAKE2B
	// TODO does rand.Int provide sufficient entropy?
	req.CensusID = fmt.Sprintf("census%d", rand.Int())
	resp, err := c.pool.Request(req, c.signingKey)
	if err != nil {
		return "", err
	}
	return resp.CensusID, nil
}

func (c *VocClient) AddClaim(censusID string, censusSigner *ethereum.SignKeys, censusPubKey string, censusValue []byte) (root types.HexBytes, _ error) {
	var req api.MetaRequest
	var hexpub string
	req.Method = "addClaim"
	req.Digested = false
	req.CensusID = censusID
	if censusSigner != nil {
		hexpub, _ = censusSigner.HexString()
	} else {
		hexpub = censusPubKey
	}
	pub, err := hex.DecodeString(hexpub)
	if err != nil {
		return types.HexBytes{}, err
	}
	req.CensusKey = pub
	req.CensusValue = censusValue
	resp, err := c.pool.Request(req, c.signingKey)
	if err != nil {
		return types.HexBytes{}, err
	}
	return resp.Root, nil
}

func (c *VocClient) AddClaimBulk(censusID string, censusSigners []*ethereum.SignKeys, censusPubKeys []string, censusValues [][]byte) (root types.HexBytes, invalidClaims []int, _ error) {
	var req api.MetaRequest
	req.CensusID = censusID
	censusSize := 0
	if censusSigners != nil {
		censusSize = len(censusSigners)
	} else {
		censusSize = len(censusPubKeys)
	}
	log.Infof("add bulk claims (size %d)", censusSize)
	req.Method = "addClaimBulk"
	req.CensusKey = []byte{}
	req.Digested = false
	currentSize := censusSize
	totalRequests := 0
	var hexpub string
	for currentSize > 0 {
		claims := [][]byte{}
		values := [][]byte{}
		for j := 0; j < 100; j++ {
			if currentSize < 1 {
				break
			}
			if censusSigners != nil {
				hexpub, _ = censusSigners[currentSize-1].HexString()
			} else {
				hexpub = censusPubKeys[currentSize-1]
			}
			pub, err := hex.DecodeString(hexpub)
			if err != nil {
				return types.HexBytes{}, []int{}, err
			}
			claims = append(claims, pub)
			values = append(values, censusValues[currentSize-1])
			currentSize--
		}
		req.CensusKeys = claims
		req.CensusValues = values
		resp, err := c.pool.Request(req, c.signingKey)
		if err != nil {
			return types.HexBytes{}, []int{}, err
		}
		root = resp.Root
		invalidClaims = append(invalidClaims, resp.InvalidClaims...)
		totalRequests++
		log.Infof("census creation progress: %d%%", (totalRequests*100*100)/(censusSize))
	}
	return root, invalidClaims, nil
}

func (c *VocClient) PublishCensus(censusID string, rootHash types.HexBytes) (uri string, _ error) {
	var req api.MetaRequest
	req.Method = "publish"
	req.CensusID = censusID
	req.RootHash = rootHash
	resp, err := c.pool.Request(req, c.signingKey)
	if err != nil {
		return "", err
	}
	uri = resp.URI
	if len(uri) < 40 {
		return "", fmt.Errorf("got invalid URI")
	}
	return resp.URI, nil
}

func (c *VocClient) GetRoot(censusID string) (root types.HexBytes, _ error) {
	var req api.MetaRequest
	req.Method = "getRoot"
	req.CensusID = censusID
	resp, err := c.pool.Request(req, c.signingKey)
	if err != nil {
		return types.HexBytes{}, err
	}
	return resp.Root, nil
}

// PROCESS APIS

func (c *VocClient) CreateProcess(
	entityID, censusRoot []byte,
	censusURI string,
	pid []byte,
	envelopeType *models.EnvelopeType,
	censusOrigin models.CensusOrigin,
	duration int) (blockHeight uint32, _ error) {
	var req api.MetaRequest
	req.Method = "submitRawTx"
	block, err := c.GetCurrentBlock()
	if err != nil {
		return 0, err
	}
	processData := &models.Process{
		EntityId:     entityID,
		CensusRoot:   censusRoot,
		CensusURI:    &censusURI,
		CensusOrigin: censusOrigin,
		BlockCount:   uint32(duration),
		ProcessId:    pid,
		StartBlock:   block + 4,
		EnvelopeType: envelopeType,
		Mode:         &models.ProcessMode{AutoStart: true, Interruptible: true},
		Status:       models.ProcessStatus_READY,
		VoteOptions:  &models.ProcessVoteOptions{MaxCount: 16, MaxValue: 8},
	}
	p := &models.NewProcessTx{
		Txtype:  models.TxType_NEW_PROCESS,
		Nonce:   util.RandomBytes(32),
		Process: processData,
	}
	stx := &models.SignedTx{}
	stx.Tx, err = proto.Marshal(&models.Tx{Payload: &models.Tx_NewProcess{NewProcess: p}})
	if err != nil {
		return 0, err
	}
	if stx.Signature, err = c.signingKey.Sign(stx.Tx); err != nil {
		return 0, err
	}
	if req.Payload, err = proto.Marshal(stx); err != nil {
		return 0, err
	}

	if _, err = c.pool.Request(req, nil); err != nil {
		return 0, err
	}
	return p.Process.StartBlock, nil
}
