package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"sort"
	"strconv"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api/client"
	"github.com/filecoin-project/lotus/api/v1api"
	"github.com/filecoin-project/lotus/chain/actors/builtin/miner"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/shopspring/decimal"

	"check-sector-info/sqlexec"
	timeToHeight "check-sector-info/time-height"
)

func ConnectClient(apiUrl string) (v1api.FullNode, jsonrpc.ClientCloser, error) {
	header := http.Header{}
	ctx := context.Background()
	return client.NewFullNodeRPCV1(ctx, apiUrl, header)
}

var url = flag.String("l", "http://127.0.0.1:1234/rpc/v0", "lotusAPI")
var detail = flag.Bool("v", false, "print sector detail")
var minerStr = flag.String("m", "", "miner")
var clusterName = flag.String("c", "", "cluster name,example:xc64,hk01")
var date = flag.String("d", "", "dateTime, example:2023-11-01 00:00:00")

type SectorInfoByDate struct {
	Date     string
	DcCount  int
	CcCount  int
	OdCount  int
	DCPledge float64
	CcPledge float64
	OdPledge float64
}

func (s SortByDate) Len() int      { return len(s) }
func (s SortByDate) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s SortByDate) Less(i, j int) bool {
	return s[i].Date < s[j].Date
}

type SortByDate []SectorInfoByDate

func main() {
	flag.Parse()

	if (*minerStr != "" && *clusterName != "") || (*minerStr == "" && *clusterName == "") {
		fmt.Println("Error: Please provide either Miner or Cluster, but not both or neither.")
		return
	}

	ctx := context.Background()

	delegate, closer, err := ConnectClient(*url)
	if err != nil {
		log.Fatalf("connect to lotus api failed")
	}
	defer closer()
	var tsk types.TipSetKey

	if *date != "" {
		dateTime, err := timeToHeight.StrToTime(*date)
		if err != nil {
			fmt.Println("Error: Wrong datetime format")
			return
		}

		tsByHeight, err := delegate.ChainGetTipSetByHeight(ctx, timeToHeight.TimeToHeight(dateTime), types.EmptyTSK)
		if err != nil {
			fmt.Println("Error: Failed to obtain tsk of specified height:", err)
			return
		}

		tsk = tsByHeight.Key()

	} else {
		tsk = types.EmptyTSK
	}

	var addr address.Address
	if *minerStr != "" {
		fmt.Println("检测到你本次使用的是矿工号，推荐使用集群代号查询，可通过-h 查询使用帮助")
		addr, err = address.NewFromString(*minerStr)
		if err != nil {
			log.Fatalf("convert miner to addr failed,err:%s", err)
		}
	}

	if *minerStr == "" {
		dsn, err := sqlexec.ReadDSN()
		if err != nil {
			log.Fatalln(err)
		}

		db, err := sqlexec.InitDB(dsn)
		if err != nil {
			log.Fatalf("connect to ops db failed,err:%s", err)
		}

		m, err := sqlexec.GetMiner(db, *clusterName)
		if err != nil {
			log.Fatalf("Failed to query miner, please confirm whether the cluster is correct")
		}
		addr, err = address.NewFromString(m)
		if err != nil {
			log.Fatalf("convert miner to addr failed,err:%s", err)
		}
		fmt.Printf("cluster: %s,miner: %s,正在查询中，请稍等...\n", *clusterName, m)
	}

	sectorInfoList, err := delegate.StateMinerActiveSectors(ctx, addr, tsk)
	if err != nil {
		log.Fatalf("failed to get miner active sector,err:%s", err)
	}

	//sort.Sort(sortByEpoch(sectorInfoList))
	groups := make(map[string][]miner.SectorOnChainInfo)
	for _, sector := range sectorInfoList {

		var SectorType string
		var Expandable bool

		if fmt.Sprintf("%v", sector.DealWeight) == "0" && fmt.Sprintf("%v", sector.VerifiedDealWeight) == "0" {
			SectorType = "cc"
		}

		if fmt.Sprintf("%v", sector.DealWeight) == "0" && fmt.Sprintf("%v", sector.VerifiedDealWeight) != "0" {
			SectorType = "dc"
		}

		if fmt.Sprintf("%v", sector.DealWeight) != "0" && fmt.Sprintf("%v", sector.VerifiedDealWeight) == "0" {
			SectorType = "od"
		}

		var dealStartEpochs []int

		if len(sector.DeprecatedDealIDs) != 0 {
			for _, dealID := range sector.DeprecatedDealIDs {
				dealInfo, err := delegate.StateMarketStorageDeal(ctx, dealID, types.EmptyTSK)
				if err != nil {
					log.Printf("failed to get deal info, err: %s\n", err)
					continue
				}
				dealStartEpochs = append(dealStartEpochs, int(dealInfo.Proposal.StartEpoch))
			}

			allGreaterThanThreshold := true
			for _, epoch := range dealStartEpochs {
				if epoch <= 2383680 {
					allGreaterThanThreshold = false
					break
				}
			}

			Expandable = allGreaterThanThreshold
		}

		if *detail == true {
			fmt.Printf("type:%s,sector:%d,Activation:%s,date:%s,expandable:%v,Expiration:%d,date:%s,DealWeight:%s,VerifiedDealWeight:%s,InitialPledge:%s,dealid:%v,DealStartEpoch:%d\n",
				SectorType,
				sector.SectorNumber,
				sector.Activation,
				timeToHeight.HeightToTime(sector.Activation),
				Expandable,
				sector.Expiration,
				timeToHeight.HeightToTime(sector.Expiration),
				sector.DealWeight,
				sector.VerifiedDealWeight,
				divideBy10ToThe18thDecimal(sector.InitialPledge),
				sector.DeprecatedDealIDs,
				dealStartEpochs)
		}
		groups[timeToHeight.HeightToDay(sector.Expiration)] = append(groups[timeToHeight.HeightToDay(sector.Expiration)], *sector)
	}

	var cc, dc, od int
	var ccp, dcp, odp float64
	var sectorInfoByDate []SectorInfoByDate
	for day, group := range groups {

		var DCCount, CCCount, ODCount int
		var DCPledge, CCPledge, ODPledge float64

		for _, s := range group {
			str := fmt.Sprintf("%s", s.InitialPledge)
			a, err := strconv.ParseFloat(str, 64)
			f := a / math.Pow(10, 18)
			if err != nil {
				fmt.Println(err)
			}

			if fmt.Sprintf("%v", s.DealWeight) == "0" && fmt.Sprintf("%v", s.VerifiedDealWeight) == "0" {
				CCCount += 1
				CCPledge += f
			}

			if fmt.Sprintf("%v", s.DealWeight) == "0" && fmt.Sprintf("%v", s.VerifiedDealWeight) != "0" {
				DCCount += 1
				DCPledge += f
			}

			if fmt.Sprintf("%v", s.DealWeight) != "0" && fmt.Sprintf("%v", s.VerifiedDealWeight) == "0" {
				ODCount += 1
				ODPledge += f
			}

		}
		s := SectorInfoByDate{
			Date:     day,
			DcCount:  DCCount,
			CcCount:  CCCount,
			OdCount:  ODCount,
			DCPledge: DCPledge,
			CcPledge: CCPledge,
			OdPledge: ODPledge,
		}
		sectorInfoByDate = append(sectorInfoByDate, s)

		dc += DCCount
		cc += CCCount
		od += ODCount
		ccp += CCPledge
		dcp += DCPledge
		odp += ODPledge

	}
	sort.Sort(SortByDate(sectorInfoByDate))
	for _, s := range sectorInfoByDate {
		fmt.Printf("%s: cc sector %d个，质押：%.4f Fil,dc sector %d个，质押：%.4f Fil,od sector %d个，质押：%.4f Fil\t 共计sector %d个，质押：%.4f Fil\n",
			s.Date,
			s.CcCount,
			s.CcPledge,
			s.DcCount,
			s.DCPledge,
			s.OdCount,
			s.OdPledge,
			s.DcCount+s.CcCount+s.OdCount,
			s.CcPledge+s.DCPledge+s.OdPledge)
	}

	fmt.Println("==============集群总览===============")
	fmt.Printf("cc sector \t%d个，质押：%.4f Fil\nod sector \t%d个，质押：%.4f Fil\ndc sector \t%d个，质押：%.4f Fil\n",
		cc,
		ccp,
		od,
		odp,
		dc,
		dcp)
	fmt.Printf("共计sector \t%d个，质押：%.4f Fil\n", dc+cc+od, ccp+dcp+odp)
}

func divideBy10ToThe18thDecimal(ta abi.TokenAmount) decimal.Decimal {

	num, _ := decimal.NewFromString(fmt.Sprintf("%s", ta))

	tenToThe18th := decimal.NewFromInt(1e18)

	result := num.Div(tenToThe18th)

	return result
}
