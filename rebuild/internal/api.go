package internal

import (
	"context"

	"github.com/filecoin-project/go-jsonrpc"

	v1 "github.com/filecoin-project/venus/venus-shared/api/chain/v1"
)

type API = v1.FullNode

func NewChainAPI(ctx context.Context, api, token string) (API, jsonrpc.ClientCloser, error) {
	client, closer, err := v1.DialFullNodeRPC(ctx, api, token, nil, v1.FullNodeWithRPCOtpions(jsonrpc.WithRetry(true)))
	if err != nil {
		return nil, nil, err
	}

	return client, closer, nil
}
