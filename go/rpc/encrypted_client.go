package rpc

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/ecies"
)

const (
	// todo: this is a convenience for testnet testing and will eventually be retrieved from the L1
	enclavePublicKeyHex = "034d3b7e63a8bcd532ee3d1d6ecad9d67fca7821981a044551f0f0cbec74d0bc5e"
)

// for these methods, the RPC method's requests and responses should be encrypted
var sensitiveMethods = []string{RPCCall, RPCGetBalance, RPCGetTxReceipt, RPCSendRawTransaction, RPCGetTransactionByHash}

// EncRPCClient is a Client wrapper that implements Client but also has extra functionality for managing viewing key registration and decryption
type EncRPCClient struct {
	obscuroClient    Client
	enclavePublicKey *ecies.PublicKey // Used to encrypt messages destined to the enclave.
	viewingKey       *ViewingKey
}

// NewEncRPCClient sets up a client with a viewing key for encrypted communication (this submits the VK to the enclave)
func NewEncRPCClient(client Client, viewingKey *ViewingKey) (*EncRPCClient, error) {
	// todo: this is a convenience for testnet but needs to replaced by a parameter and/or retrieved from the target host
	enclPubECDSA, err := crypto.DecompressPubkey(common.Hex2Bytes(enclavePublicKeyHex))
	if err != nil {
		return nil, fmt.Errorf("failed to decompress key for RPC client: %w", err)
	}
	enclavePublicKey := ecies.ImportECDSAPublic(enclPubECDSA)

	encClient := &EncRPCClient{
		obscuroClient:    client,
		enclavePublicKey: enclavePublicKey,
		viewingKey:       viewingKey,
	}
	err = encClient.registerViewingKey()
	if err != nil {
		return nil, err
	}

	return encClient, nil
}

// Call handles JSON rpc requests without a context - see CallContext for details
func (c *EncRPCClient) Call(result interface{}, method string, args ...interface{}) error {
	return c.CallContext(nil, result, method, args...) //nolint:staticcheck
}

// CallContext is the main logic to execute JSON-RPC requests, the context can be nil.
// - if the method is sensitive it will encrypt the args before sending the request and then decrypts the response before returning
// - result must be a pointer so that package json can unmarshal into it. You can also pass nil, in which case the result is ignored.
// - callExec handles the delegated call, allows EncClient to use the same code for calling with or without a context
func (c *EncRPCClient) CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	assertResultIsPointer(result)
	if !IsSensitiveMethod(method) {
		// for non-sensitive methods or when viewing keys are disabled we just delegate directly to the geth RPC client
		return c.executeRPCCall(ctx, result, method, args...)
	}

	// encode the params into a json blob and encrypt them
	encryptedParams, err := c.encryptArgs(args...)
	if err != nil {
		return fmt.Errorf("failed to encrypt args for %s call - %w", method, err)
	}

	// we set up a generic rawResult to receive the response (then we can decrypt it as necessary into the requested result type)
	var rawResult interface{}
	err = c.executeRPCCall(ctx, &rawResult, method, encryptedParams)
	if err != nil {
		return fmt.Errorf("%s rpc call failed - %w", method, err)
	}

	// if caller not interested in response, we're done
	if result == nil {
		return nil
	}

	if rawResult == nil {
		// note: some methods return nil for 'not found', caller can check for this Error type to verify
		return ErrNilResponse
	}

	// method is sensitive, so we decrypt it before unmarshalling the result
	decrypted, err := c.decryptResponse(rawResult)
	if err != nil {
		return fmt.Errorf("failed to decrypt response for %s call - %w", method, err)
	}

	// process the decrypted result to get the desired type and set it on the result pointer
	err = c.setResult(decrypted, result)
	if err != nil {
		return fmt.Errorf("failed to extract result from %s response: %w", method, err)
	}

	return nil
}

func (c *EncRPCClient) executeRPCCall(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	if ctx == nil {
		return c.obscuroClient.Call(result, method, args...)
	}
	return c.obscuroClient.CallContext(ctx, result, method, args...)
}

func (c *EncRPCClient) Stop() {
	c.obscuroClient.Stop()
}

func (c *EncRPCClient) Account() *common.Address {
	return c.viewingKey.Account
}

func (c *EncRPCClient) encryptArgs(args ...interface{}) ([]byte, error) {
	if len(args) == 0 {
		return nil, nil
	}

	paramsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("could not json encode request params: %w", err)
	}

	return c.encryptParamBytes(paramsJSON)
}

func (c *EncRPCClient) encryptParamBytes(params []byte) ([]byte, error) {
	encryptedParams, err := ecies.Encrypt(rand.Reader, c.enclavePublicKey, params, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("could not encrypt the following request params with enclave public key: %s. Cause: %w", params, err)
	}
	return encryptedParams, nil
}

func (c *EncRPCClient) decryptResponse(resultBlob interface{}) ([]byte, error) {
	resultStr, ok := resultBlob.(string)
	if !ok {
		return nil, fmt.Errorf("expected hex string but result was of type %t instead, with value %s", resultBlob, resultBlob)
	}
	encryptedResult := common.Hex2Bytes(resultStr)

	decryptedResult, err := c.viewingKey.PrivateKey.Decrypt(encryptedResult, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt result with viewing key - %w", err)
	}

	return decryptedResult, nil
}

// setResult tries to cast/unmarshal data into the result pointer, based on its type
func (c *EncRPCClient) setResult(data []byte, result interface{}) error {
	switch result := result.(type) {
	case *string:
		*result = string(data)
		return nil

	case *interface{}:
		err := json.Unmarshal(data, result)
		if err != nil {
			// if unmarshal failed with generic return we can try to send it back as a string
			*result = string(data)
		}
		return nil

	default:
		// for any other type we attempt to json unmarshal it
		return json.Unmarshal(data, result)
	}
}

// registerViewingKey submits the viewing key with signature to the enclave, this must be called before the viewing key is usable
func (c *EncRPCClient) registerViewingKey() error {
	// TODO: Store signatures to be able to resubmit keys if they are evicted by the node?
	// We encrypt the viewing key bytes
	encryptedViewingKeyBytes, err := ecies.Encrypt(rand.Reader, c.enclavePublicKey, c.viewingKey.PublicKey, nil, nil)
	if err != nil {
		return fmt.Errorf("could not encrypt viewing key with enclave public key: %w", err)
	}

	var rpcErr error
	err = c.Call(&rpcErr, RPCAddViewingKey, encryptedViewingKeyBytes, c.viewingKey.SignedKey)
	if err != nil {
		return fmt.Errorf("could not add viewing key: %w", err)
	}
	return nil
}

// IsSensitiveMethod indicates whether the RPC method's requests and responses should be encrypted.
func IsSensitiveMethod(method string) bool {
	for _, m := range sensitiveMethods {
		if m == method {
			return true
		}
	}
	return false
}

func assertResultIsPointer(result interface{}) {
	// result MUST be an initialized pointer else call won't be able to return it
	if result != nil {
		// todo: replace these panics with an error for invalid usage (same behaviour as json.Unmarshal())
		if reflect.ValueOf(result).Kind() != reflect.Ptr {
			// we panic if result is not a pointer, this is a coding mistake and we want to fail fast during development
			panic("result MUST be a pointer else Call cannot populate it")
		}
		if reflect.ValueOf(result).IsNil() {
			// we panic if result is a nil pointer, cannot unmarshall json to it. Pointer must be initialized.
			// if you see this then the calling code probably used: `var resObj *ResType` instead of: `var resObj ResType`
			panic("result pointer must be initialized else Call cannot populate it")
		}
	}
}