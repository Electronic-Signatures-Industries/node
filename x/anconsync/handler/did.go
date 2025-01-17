package handler

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/anconprotocol/node/x/anconsync"
	"github.com/anconprotocol/node/x/anconsync/impl"
	"github.com/gin-gonic/gin"
	"github.com/hyperledger/aries-framework-go/pkg/doc/did"
	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/ipld/go-ipld-prime/node/basicnode"
	"github.com/multiformats/go-multibase"
	"github.com/multiformats/go-multicodec"
)

type AvailableDid string

const (
	DidTypeWeb AvailableDid = "web"
	DidTypeKey AvailableDid = "key"
)

const (
	defaultPath  = "/.well-known/did.json"
	documentPath = "/did.json"
)

// BuildDidWeb ....
func (dagctx *AnconSyncContext) BuildDidWeb(vanityName string, pubkey []byte) (*did.Doc, error) {
	ti := time.Now()
	// did web
	base := append([]byte("did:web:ipfs:user:"), []byte(vanityName)...)
	// did web # id

	//Authentication method 2018
	didWebVer := did.NewVerificationMethodFromBytes(
		string(base),
		"Secp256k1VerificationKey2018",
		string(base),
		pubkey,
	)

	ver := []did.VerificationMethod{}
	ver = append(ver, *didWebVer)

	//	serv := []did.Service{{}, {}}

	// Secp256k1SignatureAuthentication2018
	auth := []did.Verification{{}}

	didWebAuthVerification := did.NewEmbeddedVerification(didWebVer, did.Authentication)

	auth = append(auth, *didWebAuthVerification)

	doc := did.BuildDoc(
		did.WithVerificationMethod(ver),
		///		did.WithService(serv),
		did.WithAuthentication(auth),
		did.WithCreatedTime(ti),
		did.WithUpdatedTime(ti),
	)
	doc.ID = string(base)
	return doc, nil
}

// BuildDidKey ....
func (dagctx *AnconSyncContext) BuildDidKey() (*did.Doc, error) {

	pubKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}

	ti := time.Now()
	multi := append([]byte(multicodec.Secp256k1Pub.String()), pubKey...)
	code, _ := multibase.Encode(multibase.Base58BTC, multi)
	// did key
	base := append([]byte("did:key:"), code...)
	// did key # id
	id := append(base, []byte("#")...)

	didWebVer := did.NewVerificationMethodFromBytes(
		string(id),
		"Ed25519VerificationKey2018",
		string(base),
		[]byte(pubKey),
	)

	ver := []did.VerificationMethod{}
	ver = append(ver, *didWebVer)
	//	serv := []did.Service{{}, {}}

	// Secp256k1SignatureAuthentication2018
	auth := []did.Verification{{}}

	didWebAuthVerification := did.NewEmbeddedVerification(didWebVer, did.Authentication)

	auth = append(auth, *didWebAuthVerification)

	doc := did.BuildDoc(
		did.WithVerificationMethod(ver),
		did.WithAuthentication(auth),
		did.WithCreatedTime(ti),
		did.WithUpdatedTime(ti),
	)
	doc.ID = string(base)
	return doc, nil
}
func (dagctx *AnconSyncContext) ReadDidWebUrl(c *gin.Context) {
	did := c.Param("did")

	path := strings.Join([]string{"did:web:ipfs:user", did}, ":")

	value, err := dagctx.Store.DataStore.Get(c.Request.Context(), path)
	if err != nil {
		c.JSON(400, gin.H{
			"error": fmt.Errorf("did web not found %v", err),
		})
		return
	}

	lnk, err := anconsync.ParseCidLink(string(value))
	if err != nil {
		c.JSON(400, gin.H{
			"error": fmt.Errorf("invalid hash %v", err),
		})
		return
	}

	n, err := dagctx.Store.Load(ipld.LinkContext{LinkPath: ipld.ParsePath(c.Param("path"))}, cidlink.Link{Cid: lnk.Cid})
	if err != nil {
		c.JSON(400, gin.H{
			"error": fmt.Errorf("block not found%v", err),
		})
		return
	}
	data, err := anconsync.Encode(n)
	if err != nil {
		c.JSON(400, gin.H{
			"error": fmt.Errorf("failed encoding %v", err),
		})
		return
	}
	c.JSON(200, data)

}
func (dagctx *AnconSyncContext) ReadDid(c *gin.Context) {
	did := c.Param("did")
	// address, _, err := dagctx.ParseDIDWeb(did, true)
	// if err != nil {
	// 	c.JSON(400, gin.H{
	// 		"error": fmt.Errorf("%v", err),
	// 	})
	// 	return
	// }
	value, err := dagctx.Store.DataStore.Get(c.Request.Context(), did)
	if err != nil {
		c.JSON(400, gin.H{
			"error": fmt.Errorf("did web not found %v", err),
		})
		return
	}

	lnk, err := anconsync.ParseCidLink(string(value))
	if err != nil {
		c.JSON(400, gin.H{
			"error": fmt.Errorf("invalid hash %v", err),
		})
		return
	}

	n, err := dagctx.Store.Load(ipld.LinkContext{LinkPath: ipld.ParsePath(c.Param("path"))}, cidlink.Link{Cid: lnk.Cid})
	if err != nil {
		c.JSON(400, gin.H{
			"error": fmt.Errorf("block not found%v", err),
		})
		return
	}
	data, err := anconsync.Encode(n)
	if err != nil {
		c.JSON(400, gin.H{
			"error": fmt.Errorf("failed encoding %v", err),
		})
		return
	}
	c.JSON(200, data)
}

func (dagctx *AnconSyncContext) CreateDidKey(c *gin.Context) {
	var v map[string]string

	c.BindJSON(&v)
	// if v["pub"] == "" {
	// 	c.JSON(400, gin.H{
	// 		"error": fmt.Errorf("missing pub").Error(),
	// 	})
	// 	return
	// }

	domainName := ""
	pub := []byte{}
	cid, err := dagctx.AddDid(DidTypeKey, domainName, pub)
	if err != nil {
		c.JSON(400, gin.H{
			"error": fmt.Errorf("failed to create did").Error(),
		})
	}
	c.JSON(201, gin.H{
		"cid": cid,
	})
	impl.PushBlock(c.Request.Context(), dagctx.Exchange, dagctx.IPFSPeer, cid) 
}

func (dagctx *AnconSyncContext) CreateDidWeb(c *gin.Context) {
	var v map[string]string

	c.BindJSON(&v)
	if v["domainName"] == "" {
		c.JSON(400, gin.H{
			"error": fmt.Errorf("missing domainName").Error(),
		})
		return
	}
	if v["pub"] == "" {
		c.JSON(400, gin.H{
			"error": fmt.Errorf("missing pub").Error(),
		})
		return
	}

	domainName := v["domainName"]
	pub, err := hex.DecodeString((v["pub"]))
	cid, err := dagctx.AddDid(DidTypeWeb, domainName, pub)
	if err != nil {
		c.JSON(400, gin.H{
			"error": fmt.Errorf("failed to create did").Error(),
		})
	}
	c.JSON(201, gin.H{
		"cid": cid,
	})
	impl.PushBlock(c.Request.Context(), dagctx.Exchange, dagctx.IPFSPeer, cid) 
}

func (dagctx *AnconSyncContext) AddDid(didType AvailableDid, domainName string, pubbytes []byte) (ipld.Link, error) {

	var didDoc *did.Doc
	var err error
	ctx := context.Background()

	if didType == DidTypeWeb {

		exists, err := dagctx.Store.DataStore.Has(ctx, domainName)
		if err != nil {
			return nil, fmt.Errorf("invalid domain name: %v", domainName)
		}
		if exists {
			return nil, fmt.Errorf("invalid domain name: %v", domainName)
		}

		didDoc, err = dagctx.BuildDidWeb(domainName, pubbytes)
		if err != nil {
			return nil, err
		}

	} else if didType == DidTypeKey {
		didDoc, err = dagctx.BuildDidKey()
		if err != nil {
			return nil, err
		}

	} else {
		return nil, fmt.Errorf("Must create a did")
	}
	bz, err := didDoc.JSONBytes()
	n, err := anconsync.Decode(basicnode.Prototype.Any, string(bz))
	lnk := dagctx.Store.Store(ipld.LinkContext{}, n)
	if err != nil {
		return nil, err
	}

	dagctx.Store.DataStore.Put(ctx, didDoc.ID, []byte(lnk.String()))

	return lnk, nil
}

func (dagctx *AnconSyncContext) ParseDIDWeb(id string, useHTTP bool) (string, string, error) {
	var address, host string

	parsedDID, err := did.Parse(id)
	if err != nil {
		return address, host, fmt.Errorf("invalid did, does not conform to generic did standard --> %w", err)
	}

	pathComponents := strings.Split(parsedDID.MethodSpecificID, ":")

	pathComponents[0], err = url.QueryUnescape(pathComponents[0])
	if err != nil {
		return address, host, fmt.Errorf("error parsing did:web did")
	}

	host = strings.Split(pathComponents[0], ":")[0]

	protocol := "https://"
	if useHTTP {
		protocol = "http://"
	}

	switch len(pathComponents) {
	case 1:
		address = protocol + pathComponents[0] + defaultPath
	default:
		address = protocol + strings.Join(pathComponents, "/") + documentPath
	}

	return address, host, nil
}
