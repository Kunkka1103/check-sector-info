package sqlexec

import (
	"bufio"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"os"
	"strings"
)

type Cluster struct {
	Name  string
	Miner string
}

func InitDB(DSN string) (DB *sql.DB, err error) {

	DB, err = sql.Open("mysql", DSN)
	if err != nil {
		return nil, err
	}

	info := fmt.Sprintf("dsn check success")
	log.Println(info)

	err = DB.Ping()
	if err != nil {
		return nil, err
	}

	info = fmt.Sprintf("database connect success")
	log.Println(info)

	return DB, nil
}
func ReadDSN() (dsn string, err error) {
	file, err := os.Open("dsn")
	if err != nil {
		return "", fmt.Errorf("failed to open dsn file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lines []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to read dsn file: %w", err)
	}

	if len(lines) != 1 {
		return "", fmt.Errorf("dsn file must contain exactly one non-empty line")
	}

	return lines[0], err
}

func Del(db *sql.DB, miner string, updateDate string) (sql string, err error) {
	sql = fmt.Sprintf("delete from filecoin_cluster_sector_expiration where miner='%s' and update_date='%s'", miner, updateDate)
	_, err = db.Exec(sql)
	return sql, err
}

func Insert(db *sql.DB, name string, miner string, date string, DCCount int, DCPledge float64, CCCount int, CCPledge float64, updateDate string) (sql string, err error) {
	sql = fmt.Sprintf("insert into filecoin_cluster_sector_expiration(name, miner, date, dc_count, dc_pledge, cc_count, cc_pledge,update_date) values('%s','%s','%s',%d,%.4f,%d,%.4f,'%s')",
		name, miner, date, DCCount, DCPledge, CCCount, CCPledge, updateDate)
	_, err = db.Exec(sql)
	return sql, err
}

func GetMiner(db *sql.DB, cluster string) (miner string, err error) {
	SQL := fmt.Sprintf("SELECT f0 FROM cluster_list WHERE name='%s'", cluster)
	err = db.QueryRow(SQL).Scan(&miner)
	return miner, err
}

func GetCluster(db *sql.DB) (clusterList []Cluster, err error) {
	GetSQL := "SELECT name,f0 FROM cluster_list"
	rows, err := db.Query(GetSQL)
	if err != nil {
		return nil, err
	}
	var c Cluster
	for rows.Next() {
		err = rows.Scan(&c.Name, &c.Miner)
		if err != nil {
			fmt.Println(err)
			continue
		}
		clusterList = append(clusterList, c)
	}
	return clusterList, nil
}