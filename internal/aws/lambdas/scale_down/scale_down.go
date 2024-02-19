package scale_down

import (
	"context"
	"github.com/weka/go-cloud-lib/protocol"
	"github.com/weka/go-cloud-lib/scale_down"
	"os"
	"strconv"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/db"
)

func Handler(ctx context.Context, info protocol.HostGroupInfoResponse) (response protocol.ScaleResponse, err error) {
	tableName := os.Getenv("TABLE_NAME")
	useDynamoDBEndpoint, err := strconv.ParseBool(os.Getenv("USE_DYNAMODB_ENDPOINT"))
	if err != nil {
		return protocol.ScaleResponse{}, err
	}
	if useDynamoDBEndpoint {
		creds, err2 := db.GetUsernameAndPassword(tableName)
		if err2 != nil {
			err = err2
			return
		}
		info.Username = creds.Username
		info.Password = creds.Password
	}

	info.Username, err = common.DecodeBase64(info.Username)
	if err != nil {
		return
	}

	info.Password, err = common.DecodeBase64(info.Password)
	if err != nil {
		return
	}

	return scale_down.ScaleDown(ctx, info)
}
