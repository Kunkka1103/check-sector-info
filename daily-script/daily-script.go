package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/lotus/api/client"
	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/chain/actors/builtin/miner"
	"github.com/filecoin-project/lotus/chain/types"

	"check-sector-info/sqlexec"
	timeToHeight "check-sector-info/time-height"
)

func ConnectClient(apiUrl string) (v0api.FullNode, jsonrpc.ClientCloser, error) {
	header := http.Header{}
	ctx := context.Background()
	return client.NewFullNodeRPCV0(ctx, apiUrl, header)
}

var url = flag.String("l", "http://127.0.0.1:1234/rpc/v0", "lotusAPI")
var dsn = flag.String("d", "", "ops dsn")

func main() {
	flag.Parse()

	//init lotus connext
	ctx := context.Background()

	delegate, closer, err := ConnectClient(*url)
	if err != nil {
		log.Fatalf("connect to lotus api failed,%s", err)
	}
	log.Println("lotus api connect success")
	defer closer()

	//init db
	db, err := sqlexec.InitDB(*dsn)
	if err != nil {
		log.Fatalf("init db failed,%s", err)
	}
	defer db.Close()
	log.Println("db init success")

	//get cluster
	clusterList, err := sqlexec.GetCluster(db)
	if err != nil {
		log.Fatalf("get cluster info failed ")
	}
	log.Printf("get cluster info success,number:%d", len(clusterList))

	t := time.Now()
	updateDate := t.Format("2006-01-02 00:00:00")

	//for loop check and insert
	for _, cluster := range clusterList {
		addr, err := address.NewFromString(cluster.Miner)
		if err != nil {
			log.Printf("%s %s convert to address.address failed,%s", cluster.Name, cluster.Miner, err)
			continue
		}

		sectorInfoList, err := delegate.StateMinerActiveSectors(ctx, addr, types.EmptyTSK)
		if err != nil {
			log.Println(err)
		}

		groups := make(map[string][]miner.SectorOnChainInfo)
		for _, sector := range sectorInfoList {
			//var flag bool
			//if len(sector.DealIDs) != 0 {
			//	flag = true
			//} else {
			//	flag = false
			//}

			//if *detail == true {
			//	fmt.Printf("dc:%v,sector:%d,Expiration:%d,date:%s,VerifiedDealWeight:%s,InitialPledge:%s,deadid:%v\n", flag, sector.SectorNumber, sector.Expiration, timeToHeight.HeightToTime(sector.Expiration), sector.VerifiedDealWeight, sector.InitialPledge, sector.DealIDs)
			//}
			groups[timeToHeight.HeightToDay(sector.Expiration)] = append(groups[timeToHeight.HeightToDay(sector.Expiration)], *sector)
		}

		var cc, dc int
		var ccp, dcp float64

		sql, err := sqlexec.Del(db, cluster.Miner, updateDate)
		if err != nil {
			log.Printf("%s %s sql exec failed,sql:%s,err:%s", cluster.Name, cluster.Miner, sql, err)
			continue
		}
		log.Printf("%s %s sql exec success,sql:%s", cluster.Name, cluster.Miner, sql)

		for day, group := range groups {

			var DCCount, CCCount int
			var DCPledge, CCPledge float64

			for _, s := range group {
				str := fmt.Sprintf("%s", s.InitialPledge)
				a, err := strconv.ParseFloat(str, 64)
				f := a / math.Pow(10, 18)
				if err != nil {
					fmt.Println(err)
				}
				if len(s.DealIDs) == 0 {
					CCCount += 1
					CCPledge += f
				} else {
					DCCount += 1
					DCPledge += f
				}

			}

			sql, err = sqlexec.Insert(db, cluster.Name, cluster.Miner, day, DCCount, DCPledge, CCCount, CCPledge, updateDate)
			if err != nil {
				log.Printf("%s %s sql exec failed,sql:%s,err:%s", cluster.Name, cluster.Miner, sql, err)
				continue
			}
			log.Printf("%s %s sql exec success,sql:%s", cluster.Name, cluster.Miner, sql)

			//fmt.Printf("dc sector %d个，质押：%.4f Fil,cc sector %d个，质押：%.4f Fil\t", DCCount, DCPledge, CCCount, CCPledge)
			//fmt.Printf("共计sector %d个，质押：%.4f Fil\n", DCCount+CCCount, DCPledge+CCPledge)

			dc += DCCount
			cc += CCCount
			ccp += CCPledge
			dcp += DCPledge

		}

	}

}
