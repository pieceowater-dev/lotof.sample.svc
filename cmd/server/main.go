package main

import (
	"app/internal/core/cfg"
	"app/internal/pkg"
	"github.com/gin-gonic/gin"
	gossiper "github.com/pieceowater-dev/lotof.lib.gossiper/v2"
	"google.golang.org/grpc"
)

func main() {
	appCfg := cfg.Inst()
	appRouter := pkg.NewRouter()

	serverManager := gossiper.NewServerManager()
	serverManager.AddServer(gossiper.NewGRPCServ(appCfg.GrpcPort, grpc.NewServer(), appRouter.InitGRPC))
	serverManager.AddServer(gossiper.NewRESTServ(appCfg.RestPort, gin.Default(), appRouter.InitREST))

	serverManager.StartAll()
	defer serverManager.StopAll()
}
