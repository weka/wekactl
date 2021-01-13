package debug

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"wekactl/internal/connectors"
)

var jrpcArgs struct {
	Method   string
	Host     string
	Port     int
	Username string
	Password string
}

var jrpcCmd = &cobra.Command{
	Use:   "jrpc",
	Short: "",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		client := connectors.NewJrpcClient(cmd.Context(), jrpcArgs.Host, jrpcArgs.Port, jrpcArgs.Username, jrpcArgs.Password)
		result := json.RawMessage{}
		err := client.Call(cmd.Context(), jrpcArgs.Method, struct{}{}, &result)
		if err != nil {
			log.Fatal().Msg(err.Error())
		}
		fmt.Printf("%s", result)
	},
}

func init() {
	jrpcCmd.Flags().StringVarP(&jrpcArgs.Method, "method", "m", "", "jrpc method")
	jrpcCmd.Flags().StringVarP(&jrpcArgs.Host, "host", "", "", "jrpc host")
	jrpcCmd.Flags().StringVarP(&jrpcArgs.Username, "username", "", "", "jrpc username")
	jrpcCmd.Flags().StringVarP(&jrpcArgs.Password, "password", "", "", "jrpc password")
	jrpcCmd.Flags().IntVarP(&jrpcArgs.Port, "port", "p", 14000, "jrpc port")
	_ = createAutoScalingGroupCmd.MarkFlagRequired("method")
	_ = createAutoScalingGroupCmd.MarkFlagRequired("host")
	Debug.AddCommand(jrpcCmd)
}
