package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	greptime "github.com/GreptimeTeam/greptimedb-ingester-go"
	"github.com/GreptimeTeam/greptimedb-ingester-go/table"
	"github.com/GreptimeTeam/greptimedb-ingester-go/table/types"
	"github.com/brianvoe/gofakeit/v6"
	"github.com/caarlos0/env/v6"
)

type config struct {
	Rate             float32 `env:"RATE" envDefault:"2"`
	IPv4Percent      int     `env:"IPV4_PERCENT" envDefault:"100"`
	StatusOkPercent  int     `env:"STATUS_OK_PERCENT" envDefault:"80"`
	PathMinLength    int     `env:"PATH_MIN" envDefault:"1"`
	PathMaxLength    int     `env:"PATH_MAX" envDefault:"5"`
	PercentageGet    int     `env:"GET_PERCENT" envDefault:"60"`
	PercentagePost   int     `env:"POST_PERCENT" envDefault:"30"`
	PercentagePut    int     `env:"PUT_PERCENT" envDefault:"0"`
	PercentagePatch  int     `env:"PATCH_PERCENT" envDefault:"0"`
	PercentageDelete int     `env:"DELETE_PERCENT" envDefault:"0"`
	MinRow           int     `env:"MIN_ROW" envDefault:"5"`
	MaxRow           int     `env:"MAX_ROW" envDefault:"100"`
	TableNum         int     `env:"TABLE_NUM" envDefault:"10"`
	BurstMultiplier  float32 `env:"BURST_MULTIPLIER" envDefault:"10"`
	BurstDuration    int     `env:"BURST_DURATION" envDefault:"30"`
	CycleDuration    int     `env:"CYCLE_DURATION" envDefault:"60"`
	Host             string  `env:"DB_HOST" envDefault:""`
	Database         string  `env:"DATABASE" envDefault:""`
	Port             int     `env:"DB_PORT" envDefault:"5001"`
	Username         string  `env:"DB_USERNAME" envDefault:""`
	Password         string  `env:"DB_PASSWORD" envDefault:""`
}

type TableConfig struct {
	Name            string
	SteadyRate      float32 // Steady rate (per second)
	BurstMultiplier float32 // Burst multiplier
	BurstDuration   int     // Burst duration (seconds)
	CycleDuration   int     // Cycle duration (seconds)
	LastStartTime   time.Time
}

func main() {
	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		panic(err)
	}

	log.Println("Username:", cfg.Username)
	db_cfg := greptime.NewConfig(cfg.Host).
		WithDatabase(cfg.Database).
		WithPort(cfg.Port).
		WithInsecure(false).
		WithAuth(cfg.Username, cfg.Password)
	client, err := greptime.NewClient(db_cfg)
	if err != nil {
		log.Panic(err)
	}

	checkMinMax(&cfg.PathMinLength, &cfg.PathMaxLength)

	gofakeit.Seed(time.Now().UnixNano())

	// Init table configs
	tables := []TableConfig{}
	for i := 0; i < cfg.TableNum; i++ {
		tables = append(tables, TableConfig{
			Name:            fmt.Sprintf("nginx_logs_%d", i),
			SteadyRate:      cfg.Rate,
			BurstMultiplier: gofakeit.Float32Range(cfg.BurstMultiplier/2, cfg.BurstMultiplier),
			BurstDuration:   gofakeit.Number(cfg.BurstDuration/2, cfg.BurstDuration),
			CycleDuration:   gofakeit.Number(cfg.CycleDuration/2, cfg.CycleDuration),
			LastStartTime:   time.Now(),
		})
	}

	var wg sync.WaitGroup
	for _, tableCfg := range tables {
		wg.Add(1)
		go writeTable(tableCfg, cfg, client, &wg)
	}

	// Wait for all goroutines to complete
	wg.Wait()

}

func writeTable(tableCfg TableConfig, cfg config, client *greptime.Client, wg *sync.WaitGroup) {
	defer wg.Done()

	gofakeit.Seed(time.Now().UnixNano())
	httpVersion := "HTTP/1.1"
	referrer := "-"

	for {
		elapsed := time.Since(tableCfg.LastStartTime).Seconds()

		// Determine current rate
		currentRate := tableCfg.SteadyRate
		maxRow := cfg.MaxRow
		if int(elapsed)%tableCfg.CycleDuration < tableCfg.BurstDuration {
			log.Println("Entering burst mode")
			currentRate = tableCfg.SteadyRate * tableCfg.BurstMultiplier
			maxRow = int(float32(maxRow) * tableCfg.BurstMultiplier)
		}

		// Control write frequency
		interval := time.Second / time.Duration(currentRate)
		time.Sleep(interval)

		tbl, err := table.New(tableCfg.Name)
		if err != nil {
			log.Panic(err)
		}

		// Define table schema
		tbl.AddFieldColumn("ip", types.STRING)
		tbl.AddFieldColumn("http_method", types.STRING)
		tbl.AddFieldColumn("path", types.STRING)
		tbl.AddFieldColumn("http_version", types.STRING)
		tbl.AddFieldColumn("status_code", types.INT32)
		tbl.AddFieldColumn("body_bytes_sent", types.INT32)
		tbl.AddFieldColumn("referrer", types.STRING)
		tbl.AddFieldColumn("user_agent", types.STRING)
		tbl.AddTimestampColumn("time_local", types.TIMESTAMP_MILLISECOND)

		// Generate random data
		rows := gofakeit.Number(cfg.MinRow, maxRow)
		log.Printf("Generating %d rows for table %s", rows, tableCfg.Name)

		var ip, httpMethod, path, userAgent string
		var statusCode, bodyBytesSent int
		var timeLocal time.Time
		for i := 0; i < rows; i++ {
			ip = weightedIPVersion(cfg.IPv4Percent)
			httpMethod = weightedHTTPMethod(cfg.PercentageGet, cfg.PercentagePost, cfg.PercentagePut, cfg.PercentagePatch, cfg.PercentageDelete)
			path = randomPath(cfg.PathMinLength, cfg.PathMaxLength)
			statusCode = weightedStatusCode(cfg.StatusOkPercent)
			bodyBytesSent = realisticBytesSent(statusCode)
			userAgent = gofakeit.UserAgent()
			timeLocal = time.Now()
			err = tbl.AddRow(ip, httpMethod, path, httpVersion, statusCode, bodyBytesSent, referrer, userAgent, timeLocal)
			if err != nil {
				panic(err)
			}
		}

		// Write data to database
		resp, err := client.Write(context.Background(), tbl)
		if err != nil {
			log.Println(err)
		} else {
			log.Println(resp)
		}
	}
}

func realisticBytesSent(statusCode int) int {
	if statusCode != 200 {
		return gofakeit.Number(30, 120)
	}

	return gofakeit.Number(800, 3100)
}

func weightedStatusCode(percentageOk int) int {
	roll := gofakeit.Number(0, 100)
	if roll <= percentageOk {
		return 200
	}

	return gofakeit.HTTPStatusCodeSimple()
}

func weightedHTTPMethod(percentageGet, percentagePost, percentagePut, percentagePatch, percentageDelete int) string {
	if percentageGet+percentagePost >= 100 {
		panic("HTTP method percentages add up to more than 100%")
	}

	roll := gofakeit.Number(0, 100)
	if roll <= percentageGet {
		return "GET"
	} else if roll <= percentagePost {
		return "POST"
	} else if roll <= percentagePut {
		return "PUT"
	} else if roll <= percentagePatch {
		return "PATCH"
	} else if roll <= percentageDelete {
		return "DELETE"
	}

	return gofakeit.HTTPMethod()
}

func weightedIPVersion(percentageIPv4 int) string {
	roll := gofakeit.Number(0, 100)
	if roll <= percentageIPv4 {
		return gofakeit.IPv4Address()
	} else {
		return gofakeit.IPv6Address()
	}
}

func randomPath(min, max int) string {
	var path strings.Builder
	length := gofakeit.Number(min, max)

	path.WriteString("/")

	for i := 0; i < length; i++ {
		if i > 0 {
			path.WriteString(gofakeit.RandomString([]string{"-", "-", "_", "%20", "/", "/", "/"}))
		}
		path.WriteString(gofakeit.BuzzWord())
	}

	path.WriteString(gofakeit.RandomString([]string{".hmtl", ".php", ".htm", ".jpg", ".png", ".gif", ".svg", ".css", ".js"}))

	result := path.String()
	return strings.Replace(result, " ", "%20", -1)
}

func checkMinMax(min, max *int) {
	if *min < 1 {
		*min = 1
	}
	if *max < 1 {
		*max = 1
	}
	if *min > *max {
		*min, *max = *max, *min
	}
}
