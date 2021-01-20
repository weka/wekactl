package debug

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"time"
	"wekactl/internal/connectors"
	"wekactl/internal/lib/jrpc"
	"wekactl/internal/lib/weka"
)

var jrpcArgs struct {
	Method   string
	Host     []string
	Port     int
	Username string
	Password string
}

var jrpcCmd = &cobra.Command{
	Use:   "jrpc",
	Short: "",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		ctx, _ := context.WithTimeout(cmd.Context(), time.Second * 3)
		jrpcBuilder := func(ip string) *jrpc.BaseClient {
			return connectors.NewJrpcClient(ctx, ip, weka.ManagementJrpcPort, jrpcArgs.Username, jrpcArgs.Password)
		}
		jpool := &jrpc.Pool{
			Ips:     jrpcArgs.Host,
			Clients: map[string]*jrpc.BaseClient{},
			Active:  "",
			Builder: jrpcBuilder,
			Ctx:     ctx,
		}
		result := json.RawMessage{}
		err := jpool.Call(weka.JrpcMethod(jrpcArgs.Method), struct{}{}, &result)
		if err != nil {
			log.Fatal().Msg(err.Error())
		}
		fmt.Printf("%s", result)
	},
}

func init() {
	jrpcCmd.Flags().StringVarP(&jrpcArgs.Method, "method", "m", "", "jrpc method")
	jrpcArgs.Host = *jrpcCmd.Flags().StringSlice( "host", []string{}, "jrpc host")
	jrpcCmd.Flags().StringVarP(&jrpcArgs.Username, "username", "", "", "jrpc username")
	jrpcCmd.Flags().StringVarP(&jrpcArgs.Password, "password", "", "", "jrpc password")
	jrpcCmd.Flags().IntVarP(&jrpcArgs.Port, "port", "p", 14000, "jrpc port")
	_ = createAutoScalingGroupCmd.MarkFlagRequired("method")
	_ = createAutoScalingGroupCmd.MarkFlagRequired("host")
	Debug.AddCommand(jrpcCmd)
}
